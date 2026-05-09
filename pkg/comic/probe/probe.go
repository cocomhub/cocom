// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/pkg/conv"
)

const downloadDir = "/opt/cocom/Downloads"

var (
	lastComic     int
	nhentaiMode   string
	lastComicOnce = func() func() {
		initialized := false
		return func() {
			if initialized {
				return
			}
			flag.IntVar(&lastComic, "last_comic", 634600, "last comic id")
			flag.StringVar(&nhentaiMode, "nhentai_mode", "v2", "nhentai extraction mode: v1|v2")
			flag.Parse()
			if lastComic == 0 {
				lastComic = 634600
			}
			switch nhentaiMode {
			case "v1", "v2":
			default:
				nhentaiMode = "v2"
			}
			initialized = true
		}
	}()
)

func ProbeComicJob(ctx context.Context) error {
	lastComicOnce()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		slog.Info("ProbeComic start", "mode", nhentaiMode)
		probeComic()

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			time.Sleep(10 * time.Second)
			if err := uploadComicTaskDownList(); err != nil {
				slog.Error("uploadComicTaskDownList failed", "err", err)
				continue
			}
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Minute):
		}
	}
}

func probeComic() {
	var cids []int
	tmpCids := make([]int, 0, 50)
	interval := time.Second
	sleep := func() {
		slog.Info("sleep", "interval", interval)
		time.Sleep(interval)
		interval = min(2*interval, 1*time.Minute)
	}
	for page := range 100000 {
	tryAgain:
		pageURL := fmt.Sprintf("https://nhentai.net/?page=%d", page+1)

		html, err := scraperNative(pageURL)
		if err != nil {
			slog.Error("ScraperNative failed: %w", err)
			sleep()
			goto tryAgain
		}
		if nhentaiMode == "v2" {
			ids, err := parseIDsFromIndexV2(html, lastComic)
			if err != nil {
				slog.Error("parseIDsFromIndexV2 failed: %w", err)
				sleep()
				goto tryAgain
			}
			if len(ids) == 0 {
				sleep()
				goto tryAgain
			}
			tmpCids = append(tmpCids, ids...)
			slog.Info("get comics(v2)", "page", page+1, "size", len(tmpCids), "cids", tmpCids)
		} else {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
			if err != nil {
				slog.Error("NewDocumentFromReader failed: %w", err)
				sleep()
				goto tryAgain
			}
			doc.Find(".gallery[data-tags*=\"6346\"]>.cover, .gallery[data-tags*=\"29963\"]>.cover").Each(func(i int, s *goquery.Selection) {
				href, exists := s.Attr("href")
				if !exists {
					return
				}
				id := strings.TrimPrefix(href, "/g/")
				id = strings.TrimSuffix(id, "/")
				cid, err := strconv.Atoi(id)
				if err != nil {
					return
				}
				if cid <= lastComic-100 {
					return
				}
				tmpCids = append(tmpCids, cid)
			})

			if len(tmpCids) == 0 {
				sleep()
				goto tryAgain
			}
			slog.Info("get comics", "page", page+1, "size", len(tmpCids), "cids", tmpCids)
		}
		interval = time.Second
		cids = append(cids, tmpCids...)
		tmpCids = tmpCids[:0]
		if len(cids) > 0 && cids[len(cids)-1] <= lastComic {
			break
		}
	}
	slog.Info("get comics", "cids", cids)

	slices.Reverse(cids)
	for _, cid := range cids {
		if cid <= lastComic {
			continue
		}

		comicInfo, err := getComicInfo(map[string]any{"cid": cid})
		if err == nil && comicInfo["archive"] != nil {
			slog.Info("getComicInfo", "cid", cid, "archive", comicInfo["archive"])
			continue
		}

	tryAgainComic:
		comicInfo, err = parseComicPage(cid)
		if err != nil || comicInfo["error"] != nil {
			slog.Error("parseComicPage failed", "err", err, "comicInfo", conv.JSON(comicInfo))
			sleep()
			goto tryAgainComic
		}

		if err := saveComicInfo(comicInfo); err != nil {
			slog.Error("saveComicInfo failed", "cid", cid, "err", err)
			sleep()
			goto tryAgainComic
		}

		if err := genDownList(comicInfo); err != nil {
			slog.Error("genDownList failed", "cid", cid, "err", err)
			sleep()
			goto tryAgainComic
		}

		lastComic = cid
		interval = time.Second
	}
	slog.Info("lastComic", "lastComic", lastComic)
}

func parseComicPageV1(cid int) (map[string]any, error) {
	url := fmt.Sprintf("https://nhentai.net/g/%d/", cid)
	html, err := scraperNative(url)
	if err != nil {
		return nil, fmt.Errorf("Scraper failed: %w", err)
	}
	comicInfoTxt := strings.Split(html, "window._gallery = JSON.parse(\"")[1]
	comicInfoTxt = strings.Split(comicInfoTxt, "\");\n\t</script>")[0]
	comicInfoTxt = strings.TrimSpace(comicInfoTxt)
	unquoted, err := strconv.Unquote(`"` + comicInfoTxt + `"`)
	if err != nil {
		return nil, fmt.Errorf("Unquote failed: %w", err)
	}
	slog.Info("comicInfoTxt", "comicInfoTxt", unquoted)
	comicInfo := map[string]any{}
	err = json.Unmarshal([]byte(unquoted), &comicInfo)
	if err != nil {
		return nil, fmt.Errorf("Unmarshal failed: %w", err)
	}
	comicInfo["cid"] = cid
	delete(comicInfo, "id")
	return comicInfo, nil
}

func parseComicPageV2(cid int) (map[string]any, error) {
	htmlURL := fmt.Sprintf("https://nhentai.net/g/%d/", cid)
	html, err := scraperNative(htmlURL)
	if err != nil {
		return nil, fmt.Errorf("Scraper failed: %w", err)
	}
	return parseComicPageV2FromHTML(html, cid)
}

func parseComicPageV2FromHTML(html string, cid int) (map[string]any, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("NewDocumentFromReader failed: %w", err)
	}
	target := ""
	doc.Find(`script[type="application/json"][data-sveltekit-fetched]`).Each(func(i int, s *goquery.Selection) {
		dataURL, ok := s.Attr("data-url")
		if !ok {
			return
		}
		if !strings.HasPrefix(dataURL, "/api/v2/galleries/") {
			return
		}
		if !strings.Contains(dataURL, fmt.Sprintf("/%d", cid)) {
			return
		}
		txt := strings.TrimSpace(s.Text())
		if txt != "" {
			target = txt
		}
	})
	if target == "" {
		return nil, fmt.Errorf("detail fetched json not found for %d", cid)
	}
	var fetched struct {
		Body string `json:"body"`
	}
	if err := json.Unmarshal([]byte(target), &fetched); err != nil {
		return nil, fmt.Errorf("unmarshal fetched json failed: %w", err)
	}
	if strings.TrimSpace(fetched.Body) == "" {
		return nil, fmt.Errorf("empty fetched body for %d", cid)
	}
	gallery := map[string]any{}
	if err := json.Unmarshal([]byte(fetched.Body), &gallery); err != nil {
		return nil, fmt.Errorf("unmarshal gallery body failed: %w", err)
	}
	if _, ok := gallery["images"]; !ok {
		img := map[string]any{}
		if v, ok := gallery["pages"]; ok {
			img["pages"] = v
		}
		if v, ok := gallery["cover"]; ok {
			img["cover"] = v
		}
		if v, ok := gallery["thumbnail"]; ok {
			img["thumbnail"] = v
		}
		if len(img) > 0 {
			gallery["images"] = img
		}
	}
	gallery = normalizeV2ToV1(gallery)
	slog.Info("normalize v2->v1", "cid", cid)
	gallery["cid"] = cid
	delete(gallery, "id")
	return gallery, nil
}

func parseComicPage(cid int) (map[string]any, error) {
	if nhentaiMode == "v2" {
		return parseComicPageV2(cid)
	}
	return parseComicPageV1(cid)
}

func getComicInfo(comicInfo map[string]any) (map[string]any, error) {
	url := fmt.Sprintf("http://127.0.0.1:15456/api/comic/getComicInfo?cid=%v", comicInfo["cid"])
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("NewRequest failed: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Do failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ReadAll failed: %w", err)
	}
	slog.Info("getComicInfo response", "status", resp.Status)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	type Response struct {
		Head struct {
			Code int `json:"code"`
		} `json:"head"`
		Body map[string]any `json:"body"`
	}
	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("Unmarshal failed: %w", err)
	}
	if response.Head.Code != 0 {
		return nil, fmt.Errorf("unexpected code: %d", response.Head.Code)
	}
	return response.Body, nil
}

func saveComicInfo(comicInfo map[string]any) error {
	url := "http://127.0.0.1:15456/api/comic/saveComicInfo"
	body, err := json.Marshal(comicInfo)
	if err != nil {
		return fmt.Errorf("Marshal failed: %w", err)
	}
	req, err := http.NewRequest("POST", url, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("NewRequest failed: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Do failed: %w", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ReadAll failed: %w", err)
	}
	slog.Info("saveComicInfo response", "status", resp.Status, "body", string(body))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	type Response struct {
		Head struct {
			Code int `json:"code"`
		} `json:"head"`
	}

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return fmt.Errorf("Unmarshal failed: %w", err)
	}
	if response.Head.Code != 0 {
		return fmt.Errorf("unexpected code: %d", response.Head.Code)
	}
	return nil
}

func parseIDsFromIndexV2(html string, lastComic int) ([]int, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("NewDocumentFromReader failed: %w", err)
	}
	var ids []int
	doc.Find(`script[type="application/json"][data-sveltekit-fetched]`).Each(func(i int, s *goquery.Selection) {
		dataURL, ok := s.Attr("data-url")
		if !ok || !strings.HasPrefix(dataURL, "/api/v2/galleries?page=") {
			return
		}
		txt := strings.TrimSpace(s.Text())
		if txt == "" {
			return
		}
		var fetched struct {
			Body string `json:"body"`
		}
		if err := json.Unmarshal([]byte(txt), &fetched); err != nil || len(fetched.Body) == 0 {
			return
		}
		var payload struct {
			Result []struct {
				ID     int   `json:"id"`
				TagIDs []int `json:"tag_ids"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(fetched.Body), &payload); err != nil {
			return
		}
		for _, item := range payload.Result {
			if item.ID <= lastComic-100 {
				continue
			}
			match := false
			for _, t := range item.TagIDs {
				if t == 6346 || t == 29963 {
					match = true
					break
				}
			}
			if match {
				ids = append(ids, item.ID)
			}
		}
	})
	return ids, nil
}

func normalizeV2ToV1(info map[string]any) map[string]any {
	img, _ := info["images"].(map[string]any)
	if img == nil {
		img = map[string]any{}
		info["images"] = img
	}
	normalizeOne := func(obj any) map[string]any {
		m, _ := obj.(map[string]any)
		if m == nil {
			return nil
		}
		res := map[string]any{}
		if v, ok := m["t"]; ok {
			if s, ok2 := v.(string); ok2 && s != "" {
				res["t"] = s
			}
		}
		if v, ok := m["w"]; ok {
			res["w"] = v
		}
		if v, ok := m["h"]; ok {
			res["h"] = v
		}
		if v, ok := m["width"]; ok {
			res["w"] = v
		}
		if v, ok := m["height"]; ok {
			res["h"] = v
		}
		if v, ok := m["path"]; ok {
			if p, ok2 := v.(string); ok2 {
				ext := strings.ToLower(strings.TrimPrefix(path.Ext(p), "."))
				switch ext {
				case "jpg", "jpeg":
					res["t"] = "j"
				case "png":
					res["t"] = "p"
				case "webp":
					res["t"] = "w"
				case "gif":
					res["t"] = "g"
				default:
					if _, exists := res["t"]; !exists {
						res["t"] = "j"
					}
				}
			}
		} else {
			if _, exists := res["t"]; !exists {
				res["t"] = "j"
			}
		}
		return res
	}
	if v, ok := info["pages"]; ok {
		switch arr := v.(type) {
		case []any:
			dst := make([]any, 0, len(arr))
			for _, it := range arr {
				dst = append(dst, normalizeOne(it))
			}
			img["pages"] = dst
		case []map[string]any:
			dst := make([]any, 0, len(arr))
			for _, it := range arr {
				dst = append(dst, normalizeOne(it))
			}
			img["pages"] = dst
		}
		delete(info, "pages")
	}
	if v, ok := info["cover"]; ok {
		if o := normalizeOne(v); o != nil {
			img["cover"] = o
		}
		delete(info, "cover")
	}
	if v, ok := info["thumbnail"]; ok {
		if o := normalizeOne(v); o != nil {
			img["thumbnail"] = o
		}
		delete(info, "thumbnail")
	}
	if v, ok := info["tags"].(([]any)); ok {
		for _, it := range v {
			if vv, ok := it.(map[string]any); ok {
				delete(vv, "slug")
			}
		}
	}
	delete(info, "comments")
	delete(info, "num_favorites")
	delete(info, "scanlator")
	return info
}

func genDownList(comicInfo map[string]any) error {
	body, err := json.Marshal(comicInfo)
	if err != nil {
		return fmt.Errorf("Marshal failed: %w", err)
	}
	var comicInfoObj api.ComicInfo
	err = json.Unmarshal(body, &comicInfoObj)
	if err != nil {
		return fmt.Errorf("Unmarshal failed: %w", err)
	}

	var downList strings.Builder
	for i := range comicInfoObj.Images.Pages {
		downList.WriteString(fmt.Sprintf("%s\n", comicInfoObj.PageOriginUrlByIndex(i)))
	}

	target := path.Join(downloadDir, "downList", fmt.Sprintf("%v.txt", comicInfo["cid"]))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("创建保存下载列表目录失败[%v][%s]. err:%w", comicInfo["cid"], filepath.Dir(target), err)
	}
	if err := os.WriteFile(target, []byte(downList.String()), 0o644); err != nil {
		return fmt.Errorf("保存下载列表失败[%v]. err:%w", comicInfo["cid"], err)
	}
	return nil
}

func scraperNative(url string) (string, error) {
	if !strings.Contains(url, ":18080") {
		url = strings.TrimPrefix(url, "http://")
		url = strings.TrimPrefix(url, "https://")
		url = "http://129.226.212.209:18080/" + url
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Error("ScraperNative new request failed", "url", url, "err", err)
		return "", err
	}
	slog.Info("ScraperNative request created", "url", url)
	req.Header.Set("accept", "*/*")
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("pragma", "no-cache")
	req.Header.Set("priority", "i")
	req.Header.Set("range", "bytes=0-")
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="143", "Chromium";v="143", "Not A(Brand";v="24"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"macOS"`)
	req.Header.Set("sec-fetch-dest", "video")
	req.Header.Set("sec-fetch-mode", "no-cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func uploadComicTaskDownList() error {
	cmd := exec.Command("bash", "-c", "tar czf downList.tgz downList && fileclient.sh upload downList.tgz")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = downloadDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bash failed: %w", err)
	}
	return nil
}

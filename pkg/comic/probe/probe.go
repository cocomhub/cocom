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
)

const downloadDir = "/opt/cocom/Downloads"

var (
	lastComic     int
	lastComicOnce = func() func() {
		initialized := false
		return func() {
			if initialized {
				return
			}
			flag.IntVar(&lastComic, "last_comic", 634600, "last comic id")
			flag.Parse()
			if lastComic == 0 {
				lastComic = 634600
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

		slog.Info("ProbeComic start")
		if err := probeComic(); err != nil {
			slog.Error("ProbeComic failed", "err", err)
			continue
		}

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

func probeComic() error {
	var comicIDs []int
	for page := range 100000 {
		pageURL := fmt.Sprintf("https://nhentai.net/?page=%d", page+1)

		html, err := scraperNative(pageURL)
		if err != nil {
			return fmt.Errorf("ScraperNative failed: %w", err)
		}
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			return fmt.Errorf("NewDocumentFromReader failed: %w", err)
		}

		doc.Find(".gallery[data-tags*=\"6346\"]>.cover, .gallery[data-tags*=\"29963\"]>.cover").Each(func(i int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if !exists {
				return
			}
			id := strings.TrimPrefix(href, "/g/")
			id = strings.TrimSuffix(id, "/")
			comicID, err := strconv.Atoi(id)
			if err != nil {
				return
			}
			if comicID <= lastComic-100 {
				return
			}
			comicIDs = append(comicIDs, comicID)
		})
		if len(comicIDs) > 0 && comicIDs[len(comicIDs)-1] <= lastComic {
			break
		}
	}
	slog.Info("get comics", "comics", comicIDs)

	slices.Reverse(comicIDs)
	for _, comicID := range comicIDs {
		if comicID <= lastComic {
			continue
		}

		comicInfo, err := getComicInfo(map[string]any{"comic_id": fmt.Sprintf("%d", comicID)})
		if err == nil && comicInfo["archive"] != nil {
			slog.Info("getComicInfo", "comicID", comicID, "archive", comicInfo["archive"])
			continue
		}

		comicInfo, err = parseComicPage(comicID)
		if err != nil {
			return fmt.Errorf("parseComicPage failed: %w", err)
		}

		if err := saveComicInfo(comicInfo); err != nil {
			slog.Error("saveComicInfo failed", "comicID", comicID, "err", err)
			return fmt.Errorf("saveComicInfo failed: %w", err)
		}

		if err := genDownList(comicInfo); err != nil {
			slog.Error("genDownList failed", "comicID", comicID, "err", err)
			return fmt.Errorf("genDownList failed: %w", err)
		}

		lastComic = comicID
	}
	slog.Info("lastComic", "lastComic", lastComic)
	return nil
}

func parseComicPage(comicID int) (map[string]any, error) {
	url := fmt.Sprintf("https://nhentai.net/g/%d/", comicID)
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
	comicInfo["comic_id"] = fmt.Sprintf("%d", comicID)
	comicInfo["comic_url"] = url
	return comicInfo, nil
}

func getComicInfo(comicInfo map[string]any) (map[string]any, error) {
	url := fmt.Sprintf("http://127.0.0.1:15456/api/comic/getComicInfo?id=%s", comicInfo["comic_id"].(string))
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

	target := path.Join(downloadDir, "downList", comicInfo["comic_id"].(string)+".txt")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("创建保存下载列表目录失败[%s][%s]. err:%w", comicInfo["comic_id"].(string), filepath.Dir(target), err)
	}
	if err := os.WriteFile(target, []byte(downList.String()), 0o644); err != nil {
		return fmt.Errorf("保存下载列表失败[%s]. err:%w", comicInfo["comic_id"].(string), err)
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

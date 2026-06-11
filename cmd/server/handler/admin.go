// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/cache"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/cmd/server/internal/mongo"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	"github.com/cocomhub/cocom/pkg/util"

	"go.mongodb.org/mongo-driver/bson"
)

// ---------- Compare ----------

type compareRequest struct {
	CID1 int `json:"cid1"`
	CID2 int `json:"cid2"`
}

type pageInfo struct {
	Page   int    `json:"page"`
	Name   string `json:"name"`
	MD5    string `json:"md5"`
	Exists bool   `json:"exists"`
}

type comparisonRow struct {
	Page     int    `json:"page"`
	Name     string `json:"name"`
	MD5Match bool   `json:"md5_match"`
	CID1MD5  string `json:"cid1_md5"`
	CID2MD5  string `json:"cid2_md5"`
}

type compareStats struct {
	Total      int     `json:"total"`
	Matched    int     `json:"matched"`
	Mismatched int     `json:"mismatched"`
	MatchRatio float64 `json:"match_ratio"`
}

// CompareComics 对比两个漫画的图片文件
// POST /api/admin/comic/compare
func CompareComics(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	var cr compareRequest
	if err := json.NewDecoder(req.Body).Decode(&cr); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "invalid request body")
		return
	}
	if cr.CID1 <= 0 || cr.CID2 <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "cid1 and cid2 are required")
		return
	}

	info1, info2, err := getTwoComicInfos(ctx, cr.CID1, cr.CID2)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, err.Error())
		return
	}

	pages1, err := readComicPages(cr.CID1, info1.SaveDir())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("read pages for cid %d failed: %s", cr.CID1, err))
		return
	}
	pages2, err := readComicPages(cr.CID2, info2.SaveDir())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("read pages for cid %d failed: %s", cr.CID2, err))
		return
	}

	comparison, stats := alignAndCompare(pages1, pages2)

	httpwrap.ResponseSucc(ctx, w, map[string]any{
		"cid1": map[string]any{
			"info":  info1,
			"pages": pages1,
		},
		"cid2": map[string]any{
			"info":  info2,
			"pages": pages2,
		},
		"comparison": comparison,
		"stats":      stats,
	})
}

// ---------- Link ----------

type linkRequest struct {
	MainCID int   `json:"main_cid"`
	SubCID  int   `json:"sub_cid"`  // 向后兼容
	SubCIDs []int `json:"sub_cids"` // 批量支持
}

// LinkComics 建立从属关系（支持批量）
// POST /api/admin/comic/link
func LinkComics(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	var lr linkRequest
	if err := json.NewDecoder(req.Body).Decode(&lr); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "invalid request body")
		return
	}

	// 优先使用 sub_cids 数组，兼容旧版 sub_cid
	subCIDs := lr.SubCIDs
	if len(subCIDs) == 0 && lr.SubCID > 0 {
		subCIDs = []int{lr.SubCID}
	}
	if lr.MainCID <= 0 || len(subCIDs) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "main_cid and sub_cids are required")
		return
	}

	// 去重 + 校验（自身链接、非法值）
	seen := make(map[int]bool)
	var deduped []int
	for _, sc := range subCIDs {
		if sc <= 0 || sc == lr.MainCID {
			w.WriteHeader(http.StatusBadRequest)
			httpwrap.ResponseFail(ctx, w, fmt.Sprintf("invalid sub_cid: %d", sc))
			return
		}
		if !seen[sc] {
			seen[sc] = true
			deduped = append(deduped, sc)
		}
	}

	// 批量链接，失败记录日志继续处理
	var errs []string
	for _, sc := range deduped {
		if err := linkSingleComic(ctx, lr.MainCID, sc); err != nil {
			errMsg := fmt.Sprintf("cid %d: %s", sc, err)
			errs = append(errs, errMsg)
			slog.ErrorContext(ctx, "LinkComics: batch link failed",
				slog.Int("main_cid", lr.MainCID),
				slog.Int("sub_cid", sc),
				slog.String("errmsg", err.Error()))
		}
	}

	cache.Reset()

	resp := map[string]any{
		"main_cid": lr.MainCID,
		"sub_cids": deduped,
		"status":   "linked",
	}
	if len(errs) > 0 {
		resp["errors"] = errs
	}

	httpwrap.ResponseSucc(ctx, w, resp)
}

// linkSingleComic 将一个从属 comic 链接到主 comic
func linkSingleComic(ctx context.Context, mainCID, subCID int) error {
	info1, info2, err := getTwoComicInfos(ctx, mainCID, subCID)
	if err != nil {
		return fmt.Errorf("get infos failed: %w", err)
	}

	// 将从属 comic 的 tags 合并到主 comic（按 id+type 去重）
	existingTags := make(map[string]bool)
	for _, t := range info1.Tags {
		key := fmt.Sprintf("%s:%d", t.Type, t.ID)
		existingTags[key] = true
	}
	for _, t := range info2.Tags {
		if t.Type == "language" || (t.Type == "custom" && t.Name == "textless") {
			continue
		}
		key := fmt.Sprintf("%s:%d", t.Type, t.ID)
		if !existingTags[key] {
			info1.Tags = append(info1.Tags, t)
			existingTags[key] = true
		}
	}

	// 更新主 comic 的 tags
	m1, err := util.ToMap(info1)
	if err != nil {
		return fmt.Errorf("encode main comic info failed: %w", err)
	}
	if err := comic.UpdateComicInfo(ctx, mainCID, m1); err != nil {
		return fmt.Errorf("update main comic info failed: %w", err)
	}

	// 如果主 comic 已有 redirect_to，备 comic 直接指向该目标
	targetCID := mainCID
	if info1.RedirectTo != nil && *info1.RedirectTo > 0 {
		targetCID = *info1.RedirectTo
	}

	// 设置从属 comic 的 RedirectTo
	info2.RedirectTo = &targetCID
	m2, err := util.ToMap(info2)
	if err != nil {
		return fmt.Errorf("encode sub comic info failed: %w", err)
	}
	if err := comic.UpdateComicInfo(ctx, subCID, m2); err != nil {
		return fmt.Errorf("update sub comic info failed: %w", err)
	}

	// 传播重定向链
	propagateRedirectChain(ctx, subCID, targetCID)

	return nil
}

// propagateRedirectChain 查找所有 redirect_to == subCID 的漫画，改为 redirect_to == targetCID
func propagateRedirectChain(ctx context.Context, subCID, targetCID int) {
	type redirectChainItem struct {
		CID int `bson:"cid"`
	}
	var chain []redirectChainItem
	chainBuilder := mongo.ComicInfoBuilder().
		FilterKV("redirect_to", subCID).
		Limit(100)
	if err := chainBuilder.All(ctx, &chain); err != nil {
		slog.WarnContext(ctx, "propagateRedirectChain: query failed",
			slog.Int("sub_cid", subCID),
			slog.String("errmsg", err.Error()))
		return
	}
	for _, rc := range chain {
		var rcInfo api.ComicInfo
		if err := comic.GetComicInfo(ctx, rc.CID, &rcInfo); err != nil {
			slog.WarnContext(ctx, "propagateRedirectChain: get comic info failed",
				slog.Int("cid", rc.CID),
				slog.String("errmsg", err.Error()))
			continue
		}
		rcInfo.RedirectTo = &targetCID
		rcMap, err := util.ToMap(rcInfo)
		if err != nil {
			slog.WarnContext(ctx, "propagateRedirectChain: encode comic info failed",
				slog.Int("cid", rc.CID))
			continue
		}
		if err := comic.UpdateComicInfo(ctx, rc.CID, rcMap); err != nil {
			slog.WarnContext(ctx, "propagateRedirectChain: update failed",
				slog.Int("cid", rc.CID),
				slog.String("errmsg", err.Error()))
		} else {
			slog.InfoContext(ctx, "propagateRedirectChain: updated",
				slog.Int("from_cid", rc.CID),
				slog.Int("old_main", subCID),
				slog.Int("new_main", targetCID))
		}
	}
}

// UnlinkComics 取消从属关系
// POST /api/admin/comic/unlink
func UnlinkComics(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	var lr linkRequest
	if err := json.NewDecoder(req.Body).Decode(&lr); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "invalid request body")
		return
	}

	info := api.ComicInfo{}
	if err := comic.GetComicInfo(ctx, lr.SubCID, &info); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, "get sub comic info failed")
		return
	}

	info.RedirectTo = nil
	m, err := util.ToMap(info)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, "encode comic info failed")
		return
	}
	if err := comic.UpdateComicInfo(ctx, lr.SubCID, m); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, "update comic info failed")
		return
	}

	cache.Reset()

	httpwrap.ResponseSucc(ctx, w, map[string]any{
		"sub_cid": lr.SubCID,
		"status":  "unlinked",
	})
}

// GetLinks 获取已链接的漫画列表
// GET /api/admin/comic/links?main_cid=1001&all=false
func GetLinks(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	mainCIDStr := req.URL.Query().Get("main_cid")
	all := req.URL.Query().Get("all") == "true"

	type linkedComic struct {
		CID          int    `bson:"cid"`
		RedirectTo   int    `bson:"redirect_to"`
		TitleEnglish string `bson:"title.english"`
	}

	var comics []linkedComic
	builder := mongo.ComicInfoBuilder().
		FilterKV("redirect_to", bson.M{"$ne": nil}).
		SortKV("cid", 1).
		NoLimit()

	if !all && mainCIDStr != "" {
		mainCID, err := strconv.Atoi(mainCIDStr)
		if err == nil && mainCID > 0 {
			builder.FilterKV("redirect_to", mainCID)
		}
	}

	if err := builder.All(ctx, &comics); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, "query links failed")
		return
	}

	type linkItem struct {
		SubCID   int    `json:"sub_cid"`
		SubTitle string `json:"sub_title"`
		MainCID  int    `json:"main_cid"`
	}

	links := make([]linkItem, 0, len(comics))
	for _, c := range comics {
		links = append(links, linkItem{
			SubCID:   c.CID,
			SubTitle: c.TitleEnglish,
			MainCID:  c.RedirectTo,
		})
	}

	httpwrap.ResponseSucc(ctx, w, map[string]any{
		"links": links,
		"total": len(links),
	})
}

// ---------- Helpers ----------

// DeleteComic 删除漫画
// POST /api/admin/comic/delete
func DeleteComic(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	var dr struct {
		CID int `json:"cid"`
	}
	if err := json.NewDecoder(req.Body).Decode(&dr); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "invalid request body")
		return
	}
	if dr.CID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "cid is required")
		return
	}

	if err := comic.DeleteComicByID(ctx, dr.CID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, err.Error())
		return
	}

	cache.Reset()

	httpwrap.ResponseSucc(ctx, w, map[string]any{
		"cid":    dr.CID,
		"status": "deleted",
	})
}

func getTwoComicInfos(ctx context.Context, cid1, cid2 int) (*api.ComicInfo, *api.ComicInfo, error) {
	info1 := api.ComicInfo{}
	if err := comic.GetComicInfo(ctx, cid1, &info1); err != nil {
		return nil, nil, fmt.Errorf("get cid %d info failed: %w", cid1, err)
	}
	info2 := api.ComicInfo{}
	if err := comic.GetComicInfo(ctx, cid2, &info2); err != nil {
		return nil, nil, fmt.Errorf("get cid %d info failed: %w", cid2, err)
	}
	return &info1, &info2, nil
}

func readComicPages(cid int, saveDir string) ([]pageInfo, error) {
	entries, err := os.ReadDir(saveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []pageInfo{}, nil
		}
		return nil, err
	}

	var pages []pageInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
			continue
		}
		fullPath := filepath.Join(saveDir, entry.Name())
		md5sum, err := fileMD5(fullPath)
		if err != nil {
			slog.Warn("readComicPages: md5 failed", slog.String("path", fullPath), slog.String("errmsg", err.Error()))
			md5sum = ""
		}
		pages = append(pages, pageInfo{
			Page:   0,
			Name:   entry.Name(),
			MD5:    md5sum,
			Exists: true,
		})
	}

	sort.Slice(pages, func(i, j int) bool {
		return pages[i].Name < pages[j].Name
	})
	for i := range pages {
		pages[i].Page = i + 1
	}
	return pages, nil
}

func fileMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func alignAndCompare(pages1, pages2 []pageInfo) ([]comparisonRow, compareStats) {
	m1 := make(map[string]pageInfo)
	for _, p := range pages1 {
		m1[p.Name] = p
	}
	m2 := make(map[string]pageInfo)
	for _, p := range pages2 {
		m2[p.Name] = p
	}

	allNames := make(map[string]bool)
	for _, p := range pages1 {
		allNames[p.Name] = true
	}
	for _, p := range pages2 {
		allNames[p.Name] = true
	}
	var names []string
	for n := range allNames {
		names = append(names, n)
	}
	sort.Strings(names)

	var comparison []comparisonRow
	stats := compareStats{}

	for _, name := range names {
		p1, ok1 := m1[name]
		p2, ok2 := m2[name]
		pageStr, _, _ := strings.Cut(name, ".")
		page, _ := strconv.Atoi(pageStr)
		row := comparisonRow{
			Page: page,
			Name: name,
		}

		if ok1 && ok2 {
			row.CID1MD5 = p1.MD5
			row.CID2MD5 = p2.MD5
			row.MD5Match = p1.MD5 == p2.MD5
			stats.Total++
			if row.MD5Match {
				stats.Matched++
			} else {
				stats.Mismatched++
			}
		} else if ok1 {
			row.CID1MD5 = p1.MD5
			row.CID2MD5 = ""
			row.MD5Match = false
			stats.Total++
			stats.Mismatched++
		} else {
			row.CID1MD5 = ""
			row.CID2MD5 = p2.MD5
			row.MD5Match = false
			stats.Total++
			stats.Mismatched++
		}
		comparison = append(comparison, row)
	}

	if stats.Total > 0 {
		stats.MatchRatio = float64(stats.Matched) / float64(stats.Total)
	}
	sort.SliceStable(comparison, func(i, j int) bool {
		return comparison[i].Page < comparison[j].Page
	})
	return comparison, stats
}

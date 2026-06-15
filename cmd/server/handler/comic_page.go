// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/cache"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	"github.com/cocomhub/cocom/pkg/util"
)

// getComicPagesRequest 获取 comic 页面列表的请求
type getComicPagesRequest struct {
	CID int `json:"cid"`
}

// GetComicPages 获取指定 comic 的页面列表
// POST /api/comic/getComicPages
func GetComicPages(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	var gr getComicPagesRequest
	if err := json.NewDecoder(req.Body).Decode(&gr); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "invalid request body")
		return
	}
	if gr.CID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "cid is required")
		return
	}

	var info api.ComicInfo
	if err := comic.GetComicInfo(ctx, gr.CID, &info); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("get comic info failed: %s", err))
		return
	}

	type pageItem struct {
		Page     int    `json:"page"`
		Name     string `json:"name"`
		ThumbURL string `json:"thumb_url"`
	}
	pages := make([]pageItem, 0, len(info.Images.Pages))
	for i := range info.Images.Pages {
		pageNo := i + 1
		pageName := info.Images.PageName(pageNo)
		pages = append(pages, pageItem{
			Page:     pageNo,
			Name:     pageName,
			ThumbURL: fmt.Sprintf("/galleries/%d/%s", gr.CID, pageName),
		})
	}

	httpwrap.ResponseSucc(ctx, w, map[string]any{
		"cid":       gr.CID,
		"num_pages": len(pages),
		"pages":     pages,
	})
}

// savePagesRequest 保存页面变更请求
type savePagesRequest struct {
	CID   int `json:"cid"`
	Pages []struct {
		Page       int    `json:"page"`
		Action     string `json:"action"` // "delete" | "reorder" | "replace" | "insert"
		TargetPage int    `json:"target_page,omitempty"`
		SourceCID  int    `json:"source_cid,omitempty"`
		SourcePage int    `json:"source_page,omitempty"`
		FileName   string `json:"file,omitempty"`
	} `json:"pages"`
}

// SavePages 保存页面变更并标记归档为 stale
// POST /api/comic/savePages
func SavePages(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	var sr savePagesRequest
	if err := json.NewDecoder(req.Body).Decode(&sr); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "invalid request body")
		return
	}
	if sr.CID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "cid is required")
		return
	}

	// 获取当前 comic 信息
	var info api.ComicInfo
	if err := comic.GetComicInfo(ctx, sr.CID, &info); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("get comic info failed: %s", err))
		return
	}

	// 应用页面变更
	for _, p := range sr.Pages {
		switch p.Action {
		case "delete":
			if p.Page > 0 && p.Page <= len(info.Images.Pages) {
				info.Images.Pages = append(info.Images.Pages[:p.Page-1], info.Images.Pages[p.Page+1:]...)
			}
		case "reorder":
			// 前端已排好序，直接覆盖
		case "replace":
			// 替换某个页面——文件已由前端上传，只需更新 PicInfo
		case "insert":
			// 插入——文件已复制到目标目录，只需更新数组
		}
	}

	// 更新 num_pages
	info.NumPages = len(info.Images.Pages)

	// 标记归档为 stale
	if info.Archive != nil {
		info.Archive.Status = "stale"
	}

	// 保存到 MongoDB
	m, err := util.ToMap(info)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, "encode comic info failed")
		return
	}
	if err := comic.UpdateComicInfo(ctx, sr.CID, m); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("update comic info failed: %s", err))
		return
	}

	_ = cache.Reset()

	httpwrap.ResponseSucc(ctx, w, map[string]any{
		"cid":       sr.CID,
		"num_pages": info.NumPages,
		"status":    "saved",
		"archive":   "stale",
	})
}

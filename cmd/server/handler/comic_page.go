// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

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
	// GetComicPages handler cid<=0 校验（与其它 Get 类 handler 对齐），避免负 cid 落到存储层。
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

// savePagesItem 单个页面变更项。
//
// 字段兼容两套历史命名：
//   - 位置：服务端 target_page/source_page，前端 to_page/source_pages（数组）；
//   - 文件名：服务端旧注释用 file_name、模板/前端用 file。
//
// 全部接受，避免「前端发了但服务端读不到」导致操作静默无效。
type savePagesItem struct {
	Page        int    `json:"page"`
	Action      string `json:"action"` // "delete" | "reorder" | "replace" | "insert"
	TargetPage  int    `json:"target_page,omitempty"`
	ToPage      int    `json:"to_page,omitempty"`
	SourceCID   int    `json:"source_cid,omitempty"`
	SourcePage  int    `json:"source_page,omitempty"`
	SourcePages []int  `json:"source_pages,omitempty"`
	FileName    string `json:"file,omitempty"`
	FileNameAlt string `json:"file_name,omitempty"`
}

// targetPos 返回目标位置（1-based）；无法确定时返回 0。
func (p savePagesItem) targetPos() int {
	switch {
	case p.TargetPage > 0:
		return p.TargetPage
	case p.ToPage > 0:
		return p.ToPage
	case len(p.SourcePages) > 0 && p.SourcePages[0] > 0:
		return p.SourcePages[0]
	case p.SourcePage > 0:
		return p.SourcePage
	default:
		return 0
	}
}

// fileName 返回本次操作关联的文件名（兼容 file / file_name 两种键）。
func (p savePagesItem) fileName() string {
	if p.FileName != "" {
		return p.FileName
	}
	return p.FileNameAlt
}

type savePagesRequest struct {
	CID   int             `json:"cid"`
	Pages []savePagesItem `json:"pages"`
}

// picTypeFromFileName 由文件名后缀推导 api.PicInfo.T（j/g/p/w），与 PicType() 互逆，
// 保证 PageName() 生成的磁盘文件名与实际文件一致。未知后缀按 jpg 处理。
func picTypeFromFileName(name string) string {
	switch strings.ToLower(strings.TrimPrefix(filepath.Ext(name), ".")) {
	case "gif":
		return "g"
	case "png":
		return "p"
	case "webp":
		return "w"
	default:
		return "j"
	}
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

	// 页面变更统一按「**所有位置字段都以加载时的原始页号解释**」处理，并且在**磁盘上真正
	// 落盘**：第 i 页的文件名是 <SaveDir>/<i>.<ext>（PageName 只被读取方使用），所以
	// 只改元数据会让页码与文件错位（画廊显示错误图片）。具体顺序：
	//   1) 删除：按原始页号一次性过滤（天然不受处理顺序影响）
	//   2) 交换（reorder 在前端是「交换两页」）
	//   3) 替换：用上传接口产生的暂存文件
	//   4) 插入：上传的单页，或从 source_cid 复制 source_pages
	//   5) 统一落盘（暂存目录中转，避免改名相互覆盖）
	saveDir := info.SaveDir()
	entries := buildPageEntries(saveDir, info.Images.Pages)

	deleted := make(map[int]struct{})
	for _, p := range sr.Pages {
		if p.Action == "delete" && p.Page >= 1 && p.Page <= len(info.Images.Pages) {
			deleted[p.Page] = struct{}{}
		}
	}
	entries = applyDeletes(entries, deleted)

	for _, p := range sr.Pages {
		switch p.Action {
		case "delete":
			// 已在上方统一处理。
		case "reorder":
			if err := applySwap(entries, p.Page, p.targetPos()); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				httpwrap.ResponseFail(ctx, w, err.Error())
				return
			}
		case "replace":
			if err := applyReplace(saveDir, entries, p.Page, p.fileName()); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				httpwrap.ResponseFail(ctx, w, err.Error())
				return
			}
		case "insert":
			news, err := buildInsertEntries(ctx, saveDir, p)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				httpwrap.ResponseFail(ctx, w, err.Error())
				return
			}
			entries = insertAt(entries, p.Page, news)
		default:
			w.WriteHeader(http.StatusBadRequest)
			httpwrap.ResponseFail(ctx, w, fmt.Sprintf("unsupported page action: %q", p.Action))
			return
		}
	}

	// 落盘：失败时不改动元数据，保持库内与磁盘一致（宁可报错也不静默错位）。
	if err := materializePages(saveDir, entries); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("apply page files failed: %s", err))
		return
	}

	pages := make([]api.PicInfo, 0, len(entries))
	for _, e := range entries {
		pages = append(pages, e.pic)
	}
	info.Images.Pages = pages
	info.NumPages = len(pages)

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
	// util.ToMap 走 JSON 编解码，Pages/NumPages 都带 omitempty：页面被删空时这两个键会**从 map 中消失**，
	// 而落库是 $set 部分更新（内存存储同样按 key 合并）——旧值因此残留，表现为「返回成功但页面没删掉」。
	// 这里显式回填（pages 嵌在 images 里，不在顶层），保证清空动作真正生效；
	// nil 归一为空切片，避免在库里留下 null。
	if info.Images.Pages == nil {
		info.Images.Pages = []api.PicInfo{}
	}
	if images, ok := m["images"].(map[string]any); ok {
		images["pages"] = info.Images.Pages
	}
	m["num_pages"] = info.NumPages
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

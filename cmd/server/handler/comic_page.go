// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
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

// movePage 把 1-based 的 from 位置元素移动到 1-based 的 to 位置；越界或原地时原样返回。
func movePage(pages []api.PicInfo, from, to int) []api.PicInfo {
	if from < 1 || from > len(pages) || to < 1 || to > len(pages) || from == to {
		return pages
	}
	next := make([]api.PicInfo, 0, len(pages))
	next = append(next, pages...)
	item := next[from-1]
	next = append(next[:from-1], next[from:]...)
	next = append(next, api.PicInfo{})
	copy(next[to:], next[to-1:])
	next[to-1] = item
	return next
}

// insertPage 把 page 插入到 1-based 的 pos 位置；pos<=0 或越界时追加到末尾。
func insertPage(pages []api.PicInfo, pos int, page api.PicInfo) []api.PicInfo {
	if pos < 1 || pos > len(pages)+1 {
		pos = len(pages) + 1
	}
	idx := pos - 1
	pages = append(pages, api.PicInfo{})
	copy(pages[idx+1:], pages[idx:])
	pages[idx] = page
	return pages
}

// collectDeletePages 返回按**降序**排列、已去重的待删页号（1-based，已做边界过滤）。
// 降序是正确性要求：调用方要在同一个切片上连续做 append(pages[:n-1], pages[n:]...)，
// 升序会把后面的页号指向已经移位的元素，导致删错页。
func collectDeletePages(items []savePagesItem, total int) []int {
	seen := make(map[int]struct{}, len(items))
	pages := make([]int, 0, len(items))
	for _, p := range items {
		if p.Action != "delete" || p.Page < 1 || p.Page > total {
			continue
		}
		if _, dup := seen[p.Page]; dup {
			continue
		}
		seen[p.Page] = struct{}{}
		pages = append(pages, p.Page)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(pages)))
	return pages
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

	// 删除必须先于其它动作、且按页号**降序**执行。
	//
	// p.Page 是「提交时的原始页号」（前端 state.changes 里记录的是用户看到的位置），
	// 但下面的循环是在**逐步变短的切片**上按页号取下标：若按请求顺序删「第 2 页」再删
	// 「第 3 页」，第二条实际删掉的是原第 4 页，原第 3 页反而残留 —— 静默删错页。
	// 降序执行后，每条删除的下标都仍然作用在原始位置上（去重避免同一页被删两次）。
	if deletePages := collectDeletePages(sr.Pages, len(info.Images.Pages)); len(deletePages) > 0 {
		for _, pageNo := range deletePages {
			info.Images.Pages = append(info.Images.Pages[:pageNo-1], info.Images.Pages[pageNo:]...)
		}
	}

	// 应用其余页面变更。未知动作显式返回 400：此前 reorder/replace/insert 是空实现，
	// 请求会「成功返回但元数据没变」（文件已在磁盘上却不在 pages 里），属静默数据不一致。
	for _, p := range sr.Pages {
		switch p.Action {
		case "delete":
			// 已在上方按降序统一处理。
		case "reorder":
			// 把 p.Page 移动到目标位置（target_page/to_page/source_pages[0]）。
			info.Images.Pages = movePage(info.Images.Pages, p.Page, p.targetPos())
		case "replace":
			// 文件已由前端上传，这里只更新对应页元信息（类型由新文件名后缀推导）。
			if p.Page < 1 || p.Page > len(info.Images.Pages) {
				w.WriteHeader(http.StatusBadRequest)
				httpwrap.ResponseFail(ctx, w, fmt.Sprintf("replace page out of range: %d", p.Page))
				return
			}
			info.Images.Pages[p.Page-1].T = picTypeFromFileName(p.fileName())
			info.Images.Pages[p.Page-1].Status = true
		case "insert":
			// 文件已复制到目标目录，需要在数组对应位置插入新页，否则页码与磁盘文件不一致。
			info.Images.Pages = insertPage(info.Images.Pages, p.targetPos(),
				api.PicInfo{T: picTypeFromFileName(p.fileName()), Status: true})
		default:
			w.WriteHeader(http.StatusBadRequest)
			httpwrap.ResponseFail(ctx, w, fmt.Sprintf("unsupported page action: %q", p.Action))
			return
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

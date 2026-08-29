// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/cache"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/cmd/server/internal/tag"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	"github.com/cocomhub/cocom/pkg/mutex"
)

const maxBatchSize = 500

// UpdateComicTags 更新单本漫画的 tag（添加/删除）
func UpdateComicTags(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	var updateReq api.UpdateTagsRequest
	if err := json.NewDecoder(req.Body).Decode(&updateReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "decode UpdateTagsRequest failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("invalid request body: %s", err))
		return
	}

	if updateReq.CID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "cid is required")
		return
	}

	if len(updateReq.Added) == 0 && len(updateReq.Removed) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "added or removed is required")
		return
	}

	unlock, err := mutex.Lock(ctx, fmt.Sprintf("comic/%d", updateReq.CID))
	if err != nil {
		w.WriteHeader(http.StatusTooManyRequests)
		slog.ErrorContext(ctx, "mutex lock failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("mutex lock failed: %s", err))
		return
	}
	defer unlock()

	info := api.ComicInfo{}
	if err = comic.GetComicInfo(ctx, updateReq.CID, &info); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.ErrorContext(ctx, "get comic info failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("get comic info failed: %s", err))
		return
	}

	diff := struct {
		Added   []api.Tag `json:"added"`
		Removed []api.Tag `json:"removed"`
		Current []api.Tag `json:"current"`
	}{Current: info.Tags}

	// 自动分配 ID：对 ID == 0 的新增 tag 从 1000000000 起始分配
	needAssign := false
	for _, at := range updateReq.Added {
		if at.ID == 0 {
			needAssign = true
			break
		}
	}
	if needAssign {
		maxID, getErr := tag.GetMaxTagID(ctx)
		if getErr != nil {
			slog.WarnContext(ctx, "GetMaxTagID failed, using 1000000000 as base", slog.String("errmsg", getErr.Error()))
			maxID = 0
		}
		nextID := max(maxID+1, 1000000000)
		for i := range updateReq.Added {
			if updateReq.Added[i].ID == 0 {
				updateReq.Added[i].ID = nextID
				nextID++
			}
		}
	}

	// 移除 tag：按 type+id 匹配（id>0）或 type+name 匹配
	removedSet := make(map[string]bool)
	for _, rt := range updateReq.Removed {
		key := tagKey(rt.Type, rt.ID, rt.Name)
		removedSet[key] = true
	}

	var kept []api.Tag
	for _, t := range info.Tags {
		key := tagKey(t.Type, t.ID, t.Name)
		if removedSet[key] {
			diff.Removed = append(diff.Removed, t)
		} else {
			kept = append(kept, t)
		}
	}
	info.Tags = kept

	// 添加 tag：去重后追加
	for _, at := range updateReq.Added {
		key := tagKey(at.Type, at.ID, at.Name)
		exists := false
		for _, t := range info.Tags {
			if tagKey(t.Type, t.ID, t.Name) == key {
				exists = true
				break
			}
		}
		if !exists {
			info.Tags = append(info.Tags, at)
			diff.Added = append(diff.Added, at)
		}
	}

	// 只有发生变更才写回
	if len(diff.Added) > 0 || len(diff.Removed) > 0 {
		// 构造仅包含标签变更的更新 map，避免全量 ToMap 导致
		// ComicInfo.Images(ComicImages 结构体) 与 ComicImpl.Images([]Image 切片)
		// 类型不兼容引发的 JSON 反序列化失败。
		m := map[string]any{
			"id":   strconv.Itoa(updateReq.CID),
			"cid":  updateReq.CID,
			"tags": info.Tags,
		}
		if err = comic.UpdateComicInfo(ctx, updateReq.CID, m); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			slog.ErrorContext(ctx, "update comic info failed", slog.String("errmsg", err.Error()))
			httpwrap.ResponseFail(ctx, w, fmt.Sprintf("update comic info failed: %s", err))
			return
		}

		// 增量更新 comicTag 集合
		for _, at := range diff.Added {
			if err := tag.UpdateComicTagIncremental(ctx, at.Type, at.ID, at.Name, at.URL, 1); err != nil {
				slog.WarnContext(ctx, "comicTag incremental add failed", slog.String("errmsg", err.Error()))
			}
		}
		for _, rt := range diff.Removed {
			if err := tag.UpdateComicTagIncremental(ctx, rt.Type, rt.ID, rt.Name, rt.URL, -1); err != nil {
				slog.WarnContext(ctx, "comicTag incremental remove failed", slog.String("errmsg", err.Error()))
			}
		}
	}

	diff.Current = info.Tags
	httpwrap.ResponseSucc(ctx, w, diff)
}

// GetSearchUniqueTags 获取搜索结果中去重后的 tag 列表
func GetSearchUniqueTags(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	query := req.URL.Query().Get("q")
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "query q is required")
		return
	}

	limit := int64(100)
	if l := req.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.ParseInt(l, 10, 64); err == nil && v > 0 {
			limit = v
		}
	}

	skip := int64(0)
	if s := req.URL.Query().Get("skip"); s != "" {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil && v >= 0 {
			skip = v
		}
	}

	tags, cidList, total, err := tag.GetSearchUniqueTags(ctx, query, limit, skip)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.ErrorContext(ctx, "GetSearchUniqueTags failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, "get search unique tags failed")
		return
	}

	httpwrap.ResponseSucc(ctx, w, api.SearchUniqueTagsResponse{
		Tags:    tags,
		CIDList: cidList,
		Total:   int(total),
	})
}

// GetRelatedTags 获取与指定 tag 关联的其他 tag
func GetRelatedTags(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	tagType := req.URL.Query().Get("type")
	tagName := req.URL.Query().Get("name")
	if tagType == "" || tagName == "" {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "type and name are required")
		return
	}

	limit := int64(30)
	if l := req.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.ParseInt(l, 10, 64); err == nil && v > 0 {
			limit = v
		}
	}

	tags, err := tag.GetRelatedTags(ctx, tagType, tagName, limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.ErrorContext(ctx, "GetRelatedTags failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, "get related tags failed")
		return
	}

	httpwrap.ResponseSucc(ctx, w, api.RelatedTagsResponse{Tags: tags})
}

// BatchAddTagToComics 批量添加 tag 到多个漫画
func BatchAddTagToComics(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	var batchReq api.BatchAddTagRequest
	if err := json.NewDecoder(req.Body).Decode(&batchReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "decode BatchAddTagRequest failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("invalid request body: %s", err))
		return
	}

	if len(batchReq.CIDList) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "cidList is required")
		return
	}

	if batchReq.Tag.ID <= 0 && batchReq.Tag.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "tag id or name is required")
		return
	}

	// 限制批量大小
	if len(batchReq.CIDList) > maxBatchSize {
		batchReq.CIDList = batchReq.CIDList[:maxBatchSize]
	}

	resp := api.BatchAddTagResponse{}
	var errorsList []int

	for _, cid := range batchReq.CIDList {
		unlock, err := mutex.Lock(ctx, fmt.Sprintf("comic/%d", cid))
		if err != nil {
			slog.ErrorContext(ctx, "mutex lock failed", slog.Int("cid", cid), slog.String("errmsg", err.Error()))
			errorsList = append(errorsList, cid)
			continue
		}

		info := api.ComicInfo{}
		if err = comic.GetComicInfo(ctx, cid, &info); err != nil {
			unlock()
			slog.WarnContext(ctx, "get comic info failed", slog.Int("cid", cid), slog.String("errmsg", err.Error()))
			errorsList = append(errorsList, cid)
			continue
		}

		// 检查 tag 是否已存在
		newKey := tagKey(batchReq.Tag.Type, batchReq.Tag.ID, batchReq.Tag.Name)
		exists := false
		for _, t := range info.Tags {
			if tagKey(t.Type, t.ID, t.Name) == newKey {
				exists = true
				break
			}
		}

		if !exists {
			info.Tags = append(info.Tags, batchReq.Tag)
			m := map[string]any{
				"id":   strconv.Itoa(cid),
				"cid":  cid,
				"tags": info.Tags,
			}
			if err = comic.UpdateComicInfo(ctx, cid, m); err != nil {
				unlock()
				slog.ErrorContext(ctx, "update comic info failed", slog.Int("cid", cid), slog.String("errmsg", err.Error()))
				errorsList = append(errorsList, cid)
				continue
			}
			resp.Updated++
		}
		unlock()
	}

	if len(errorsList) > 0 {
		resp.Errors = errorsList
	}

	// 批量操作结束后，增量更新 comicTag count
	t := batchReq.Tag
	if err := tag.UpdateComicTagIncremental(ctx, t.Type, t.ID, t.Name, t.URL, resp.Updated); err != nil {
		slog.WarnContext(ctx, "incremental tag update failed, falling back to full aggregate",
			slog.String("errmsg", err.Error()))
		if err := tag.AggregateTags(ctx); err != nil {
			slog.ErrorContext(ctx, "re-aggregate tags after batch add failed", slog.String("errmsg", err.Error()))
		}
	}
	if err := cache.Reset(); err != nil {
		slog.WarnContext(ctx, "cache reset after batch add failed", slog.String("errmsg", err.Error()))
	}

	httpwrap.ResponseSucc(ctx, w, resp)
}

// tagKey 生成 tag 的唯一键（用于去重判断）
func tagKey(tagType string, id int, name string) string {
	if id > 0 {
		return fmt.Sprintf("%s:%d", tagType, id)
	}
	return fmt.Sprintf("%s:%s", tagType, name)
}

// Copyright 2026 Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"log/slog"
	"net/http"
	"regexp"
	"strconv"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/cmd/server/internal/tag"
	comicpkg "github.com/cocomhub/cocom/pkg/comic"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchAutocomplete GET /api/search/autocomplete?q=xxx&limit=5
// 返回匹配的漫画标题和标签（混合下拉）
func SearchAutocomplete(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	query := req.URL.Query().Get("q")
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "query q is required")
		return
	}

	limit := int64(5)
	if l := req.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.ParseInt(l, 10, 64); err == nil && v > 0 {
			limit = v
		}
	}

	// --- 使用 DefaultStorage 搜索漫画标题 ---
	var comics []*api.AutocompleteComic
	if s := comic.GetDefaultStorage(); s != nil {
		filter := comicpkg.NewComicFilter().
			SetLimit(limit).
			SetTitleORPatterns(regexp.QuoteMeta(query))
		found, err := s.Find(ctx, filter)
		if err != nil {
			slog.ErrorContext(ctx, "search comics via default storage failed", slog.String("errmsg", err.Error()))
		} else {
			for _, c := range found {
				cid, _ := strconv.Atoi(c.GetID())
				comics = append(comics, &api.AutocompleteComic{
					CID:   cid,
					Title: c.GetTitlePretty(),
				})
			}
		}
	} else {
		// --- 原有 MongoDB 路径作为 fallback ---
		escapedQuery := primitive.Regex{Pattern: regexp.QuoteMeta(query), Options: "i"}
		infos, err := comic.GetRangeComicInfos(ctx, limit, 0,
			"$or", []bson.M{
				{"title.english": bson.M{"$regex": escapedQuery}},
				{"title.japanese": bson.M{"$regex": escapedQuery}},
				{"title.pretty": bson.M{"$regex": escapedQuery}},
			})
		if err == nil {
			for _, info := range infos {
				comics = append(comics, &api.AutocompleteComic{
					CID:   info.CID,
					Title: info.Title.Pretty,
				})
			}
		}
	}

	// 搜索标签
	tags, err := tag.SearchTags(ctx, "", query, limit)
	if err != nil {
		slog.ErrorContext(ctx, "search tags failed", slog.String("errmsg", err.Error()))
		tags = nil
	}

	httpwrap.ResponseSucc(ctx, w, api.AutocompleteResponse{
		Comics: comics,
		Tags:   tags,
	})
}

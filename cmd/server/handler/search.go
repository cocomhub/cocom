// Copyright 2026 The Cocomhub Authors. All rights reserved.
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
		if v, err := strconv.ParseInt(l, 10, 64); err == nil && v > 0 && v <= 20 {
			limit = v
		}
	}

	// 搜索漫画标题
	escapedQuery := primitive.Regex{Pattern: regexp.QuoteMeta(query), Options: "i"}
	infos, err := comic.GetRangeComicInfos(ctx, limit, 0,
		"$or", []bson.M{
			{"title.english": bson.M{"$regex": escapedQuery}},
			{"title.japanese": bson.M{"$regex": escapedQuery}},
			{"title.pretty": bson.M{"$regex": escapedQuery}},
		})
	if err != nil {
		slog.ErrorContext(ctx, "search autocomplete comics failed", slog.String("errmsg", err.Error()))
		// 不中断，只返回空漫画列表
		infos = nil
	}

	comics := make([]*api.AutocompleteComic, 0, len(infos))
	for _, info := range infos {
		title := info.Title.English
		if title == "" {
			title = info.Title.Pretty
		}
		if title == "" {
			title = info.Title.Japanese
		}
		comics = append(comics, &api.AutocompleteComic{
			CID:   info.CID,
			Title: title,
		})
	}

	// 搜索标签名
	tags, err := tag.SearchTags(ctx, "", query, limit)
	if err != nil {
		slog.ErrorContext(ctx, "search autocomplete tags failed", slog.String("errmsg", err.Error()))
		tags = nil
	}

	httpwrap.ResponseSucc(ctx, w, api.AutocompleteResponse{
		Comics: comics,
		Tags:   tags,
	})
}
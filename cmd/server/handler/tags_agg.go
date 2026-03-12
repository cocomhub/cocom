// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/cmd/server/internal/tag"
	"github.com/cocomhub/cocom/pkg/httpwrap"
)

func AggregateTags(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	if err := tag.AggregateTags(ctx); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.ErrorContext(ctx, "aggregate tags failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, "aggregate tags failed")
		return
	}
	httpwrap.ResponseSucc(ctx, w, "ok")
}

func GetTags(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	tagType := req.URL.Query().Get("type")
	if tagType == "" {
		tagType = "tag"
	}
	sortType := comic.SortTypeByName
	if s := req.URL.Query().Get("sort"); s != "" {
		if s == "popular" {
			sortType = comic.SortTypeByPopular
		} else {
			sortType = comic.SortTypeByName
		}
	} else if s := req.URL.Query().Get("sortType"); s != "" {
		if s == "popular" {
			sortType = comic.SortTypeByPopular
		}
	}
	limit := int64(20)
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
	likedOnly := false
	if lo := req.URL.Query().Get("likedOnly"); lo != "" {
		if lo == "true" || lo == "1" {
			likedOnly = true
		}
	}
	if l := req.URL.Query().Get("liked"); l != "" && !likedOnly {
		if l == "true" || l == "1" {
			likedOnly = true
		}
	}
	tags, total, err := tag.AggregateTagList(ctx, tagType, sortType, skip, limit, likedOnly)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.ErrorContext(ctx, "get tags failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, "get tags failed")
		return
	}
	httpwrap.ResponseSucc(ctx, w, map[string]any{
		"data":  tags,
		"total": total,
	})
}

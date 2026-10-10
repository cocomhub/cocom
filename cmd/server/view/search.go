// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package view

import (
	"log/slog"
	"net/http"
	"regexp"
	"strconv"

	"github.com/cocomhub/cocom/pkg/errwrap"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// maxSearchQueryLen 限制搜索关键词长度，避免超长输入放大查询开销。
const maxSearchQueryLen = 200

func parseSearchResultPageArgs(c *gin.Context) (page int, query string, err error) {
	if len(c.Query("page")) != 0 {
		page, err = strconv.Atoi(c.Query("page"))
		if err != nil {
			err = errwrap.ErrInvalidArgs.SetIErrF("parse page failed: %s", err)
			return
		}
	}

	if page < 1 {
		page = 1
	}

	query = c.Query("q")
	if len(query) == 0 {
		err = errwrap.ErrInvalidArgs.SetIErrF("query not found")
		return
	}
	if len(query) > maxSearchQueryLen {
		err = errwrap.ErrInvalidArgs.SetIErrF("query too long: %d > %d", len(query), maxSearchQueryLen)
		return
	}
	slog.InfoContext(c, "search query", slog.String("query", query))
	return
}

// buildSearchTitleFilter 构造标题搜索（english/japanese/pretty）的 $or 过滤条件。
// 关键词经 regexp.QuoteMeta 转义后仅作为**字面量**匹配：既防正则注入，也防 ReDoS。
func buildSearchTitleFilter(query string) []bson.M {
	escaped := regexp.QuoteMeta(query)
	re := primitive.Regex{Pattern: escaped, Options: "i"}
	return []bson.M{
		{"title.english": bson.M{"$regex": re}},
		{"title.japanese": bson.M{"$regex": re}},
		{"title.pretty": bson.M{"$regex": re}},
	}
}

func SearchResultPage(c *gin.Context) {
	page, query, err := parseSearchResultPageArgs(c)
	if err != nil {
		slog.ErrorContext(c, "parseSearchResultPageArgs failed", slog.String("errmsg", err.Error()))
		httpwrap.GinRespondError(c, http.StatusBadRequest, httpwrap.ErrCodeInvalid, "invalid search request")
		c.Abort()
		return
	}

	// 关键词一律 QuoteMeta：用户输入不得作为正则语法参与匹配，
	// 否则既是正则注入（如 .* 匹配全部）也是 ReDoS 入口。与 handler/search.go 保持一致。
	indexInfo, err := NewGalleryIndexPage(c, c.Request.URL.Path, page, "$or", buildSearchTitleFilter(query))
	if err != nil {
		slog.ErrorContext(c, "NewGalleryIndexPage failed", slog.String("errmsg", err.Error()))
		httpwrap.GinRespondError(c, http.StatusBadRequest, httpwrap.ErrCodeInvalid, "invalid search request")
		c.Abort()
		return
	}
	indexInfo.SearchQuery = query

	c.HTML(http.StatusOK, "index.tpl", indexInfo)
}

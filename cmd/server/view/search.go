// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package view

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cocomhub/cocom/pkg/errwrap"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

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
	slog.InfoContext(c, "search query", slog.String("query", query))
	return
}

func SearchResultPage(c *gin.Context) {
	page, query, err := parseSearchResultPageArgs(c)
	if err != nil {
		slog.ErrorContext(c, "parseSearchResultPageArgs failed", slog.String("errmsg", err.Error()))
		httpwrap.GinRespondError(c, http.StatusBadRequest, httpwrap.ErrCodeInvalid, "invalid search request")
		c.Abort()
		return
	}

	indexInfo, err := NewGalleryIndexPage(c, c.Request.URL.Path, page, "$or", []bson.M{
		{"title.english": bson.M{"$regex": primitive.Regex{Pattern: query, Options: "i"}}},
		{"title.japanese": bson.M{"$regex": primitive.Regex{Pattern: query, Options: "i"}}},
		{"title.pretty": bson.M{"$regex": primitive.Regex{Pattern: query, Options: "i"}}},
	})
	if err != nil {
		slog.ErrorContext(c, "NewGalleryIndexPage failed", slog.String("errmsg", err.Error()))
		httpwrap.GinRespondError(c, http.StatusBadRequest, httpwrap.ErrCodeInvalid, "invalid search request")
		c.Abort()
		return
	}
	indexInfo.SearchQuery = query

	c.HTML(http.StatusOK, "index.tpl", indexInfo)
}

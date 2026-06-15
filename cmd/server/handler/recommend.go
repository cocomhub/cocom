// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	"github.com/gin-gonic/gin"
)

var validTypes = map[string]bool{"artist": true, "group": true, "parody": true, "character": true, "tag": true}

// GetRecommendations 返回指定维度的推荐漫画
// GET /api/comic/recommendations?cid=12345&type=artist
func GetRecommendations(c *gin.Context) {
	ctx := c.Request.Context()

	cidStr := c.Query("cid")
	cid, err := strconv.Atoi(cidStr)
	if err != nil || cid <= 0 {
		httpwrap.GinRespondError(c, http.StatusBadRequest, httpwrap.ErrCodeInvalid, "invalid cid")
		return
	}

	tagType := c.Query("type")
	if !validTypes[tagType] {
		httpwrap.GinRespondError(c, http.StatusBadRequest, httpwrap.ErrCodeInvalid, "invalid type, must be one of: artist, group, parody, character, tag")
		return
	}

	info := api.ComicInfo{}
	if getErr := comic.GetComicInfo(ctx, cid, &info); getErr != nil {
		slog.ErrorContext(ctx, "GetRecommendations: GetComicInfo failed",
			slog.Int("cid", cid),
			slog.String("errmsg", getErr.Error()))
		httpwrap.GinRespondError(c, http.StatusInternalServerError, httpwrap.ErrCodeInternal, "get comic info failed")
		return
	}

	limit := config.GetRecommendLimit()
	if limit <= 0 {
		limit = 5
	}

	infos, err := comic.GetByTagType(ctx, cid, info.Tags, tagType, limit)
	if err != nil {
		slog.ErrorContext(ctx, "GetRecommendations: GetByTagType failed",
			slog.Int("cid", cid),
			slog.String("type", tagType),
			slog.String("errmsg", err.Error()))
		httpwrap.GinRespondError(c, http.StatusInternalServerError, httpwrap.ErrCodeInternal, "get recommendations failed")
		return
	}

	type recommResult struct {
		CID          int    `json:"cid"`
		TitleEnglish string `json:"title_english"`
		MediaID      string `json:"media_id"`
		CoverName    string `json:"cover_name"`
		TagsIDString string `json:"tags_id_string,omitempty"`
		NumPages     int    `json:"num_pages,omitempty"`
	}

	results := make([]recommResult, 0, len(infos))
	for _, item := range infos {
		results = append(results, recommResult{
			CID:          item.CID,
			TitleEnglish: item.Title.English,
			MediaID:      fmt.Sprint(item.CID),
			CoverName:    item.Images.CoverName(),
			TagsIDString: item.Tags.IdString(),
			NumPages:     item.NumPages,
		})
	}

	httpwrap.GinRespondOK(c, gin.H{
		"cid":     cid,
		"type":    tagType,
		"limit":   limit,
		"results": results,
	})
}

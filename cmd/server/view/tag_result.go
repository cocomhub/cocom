// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package view

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cocomhub/cocom/cmd/server/internal/tag"
	"github.com/cocomhub/cocom/pkg/errwrap"
	"github.com/cocomhub/cocom/pkg/httpwrap"

	"github.com/gin-gonic/gin"
)

func parseTagResultPageArgs(c *gin.Context) (page int, tag string, name string, url string, err error) {
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

	tag = c.Param("tag")
	if len(tag) == 0 {
		err = errwrap.ErrInvalidArgs.SetIErrF("tag type not found")
		return
	}

	name = c.Param("name")
	if len(name) == 0 {
		err = errwrap.ErrInvalidArgs.SetIErrF("tag name not found")
		return
	}
	url = fmt.Sprintf("/%s/%s/", tag, name)
	return
}

func TagResultPage(c *gin.Context) {
	page, tagType, tagName, url, err := parseTagResultPageArgs(c)
	if err != nil {
		slog.ErrorContext(c, "parseTagResultPageArgs failed",
			slog.String("errmsg", err.Error()))
		httpwrap.GinRespondError(c, http.StatusBadRequest, httpwrap.ErrCodeInvalid, "invalid request")
		c.Abort()
		return
	}

	indexInfo, err := NewGalleryIndexPage(c, c.Request.URL.Path, page, "tags.type", tagType, "tags.url", url)
	if err != nil {
		slog.ErrorContext(c, "NewGalleryIndexPage failed",
			slog.String("url", url),
			slog.Int("page", page),
			slog.String("tagType", tagType),
			slog.String("tagName", tagName),
			slog.String("errmsg", err.Error()))
		httpwrap.GinRespondError(c, http.StatusBadRequest, httpwrap.ErrCodeInvalid, "invalid request")
		c.Abort()
		return
	}

	doc, err := tag.GetTagByTypeURL(c, tagType, url)
	if err != nil {
		// 主文档查询故障 → 500（区别于 not-found：not-found 由 doc==nil 分支渲染 ID0）
		slog.ErrorContext(c, "GetTagByTypeURL failed",
			slog.String("tagType", tagType),
			slog.String("url", url),
			slog.String("errmsg", err.Error()))
		httpwrap.GinRespondError(c, http.StatusInternalServerError, httpwrap.ErrCodeInternal, "query tag failed")
		c.Abort()
		return
	}
	if doc != nil {
		indexInfo.CurTag = &TagMeta{
			Type: tagType,
			ID:   doc.ID,
			Name: tagName,
			URL:  url,
			Like: doc.Like,
		}
	} else {
		indexInfo.CurTag = &TagMeta{
			Type: tagType,
			ID:   0,
			Name: tagName,
			URL:  url,
			Like: false,
		}
	}

	// Fetch related tags (computed + explicit)
	// 附属数据故障可降级：不阻塞主页面渲染，仅记录日志（区别于主干 GetTagByTypeURL 的 500）。
	relatedTags, err := tag.GetRelatedTags(c, tagType, tagName, 30)
	if err != nil {
		slog.ErrorContext(c, "get related tags failed (degraded)",
			slog.String("tagType", tagType),
			slog.String("tagName", tagName),
			slog.String("errmsg", err.Error()))
	} else {
		indexInfo.RelatedTags = relatedTags
	}

	// Fetch explicit relation groups for management
	// 附属数据故障可降级：不阻塞主页面渲染，仅记录日志（区别于主干 GetTagByTypeURL 的 500）。
	if indexInfo.CurTag != nil && indexInfo.CurTag.ID > 0 {
		groups, err := tag.GetRelationsGroupList(c, indexInfo.CurTag.Type, indexInfo.CurTag.ID)
		if err != nil {
			slog.ErrorContext(c, "get relations group list failed (degraded)",
				slog.String("tagType", tagType),
				slog.String("tagName", tagName),
				slog.String("errmsg", err.Error()))
		} else {
			indexInfo.TagRelations = groups
		}
	}

	c.HTML(http.StatusOK, "index.tpl", indexInfo)
}

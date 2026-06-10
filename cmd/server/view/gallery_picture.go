// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package view

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/pkg/errwrap"
	"github.com/cocomhub/cocom/pkg/httpwrap"

	"github.com/gin-gonic/gin"
)

func parseGalleryPicturePageArgs(c *gin.Context) (cid int, no int, err error) {
	cid, err = strconv.Atoi(c.Param("cid"))
	if err != nil {
		err = errwrap.ErrInvalidArgs.SetIErrF("parse cid failed: %s", err)
		return
	}

	no, err = strconv.Atoi(c.Param("no"))
	if err != nil {
		err = errwrap.ErrInvalidArgs.SetIErrF("parse pic no failed: %s", err)
		return
	}

	return
}

func GalleryPicturePage(c *gin.Context) {
	cid, no, err := parseGalleryPicturePageArgs(c)
	if err != nil {
		slog.ErrorContext(c, "parseGalleryPicturePageArgs failed",
			slog.String("errmsg", err.Error()))
		httpwrap.GinRespondError(c, http.StatusBadRequest, httpwrap.ErrCodeInvalid, "invalid request")
		c.Abort()
		return
	}

	info := api.ComicInfo{}
	err = comic.GetComicInfo(c, cid, &info)
	if err != nil {
		slog.ErrorContext(c, "comic.GetComicInfo failed",
			slog.String("errmsg", err.Error()))
		httpwrap.GinRespondError(c, http.StatusBadRequest, httpwrap.ErrCodeInvalid, "resource not found")
		c.Abort()
		return
	}

	// 检查是否有重定向（从属漫画）
	if info.RedirectTo != nil && *info.RedirectTo > 0 {
		c.Redirect(http.StatusFound, fmt.Sprintf("/g/%d/%d/", *info.RedirectTo, no))
		return
	}

	// 检查漫画是否已被删除
	if info.Deleted {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.File(info.PageSavePath(no))
}

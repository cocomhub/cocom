// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package view

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/pkg/errwrap"

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
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	info := api.ComicInfo{}
	err = comic.GetComicInfo(c, cid, &info)
	if err != nil {
		slog.ErrorContext(c, "comic.GetComicInfo failed",
			slog.String("errmsg", err.Error()))
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	c.File(info.PageSavePath(no))
}

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

func parsePictureArgs(c *gin.Context) (cid int, name string, err error) {
	cid, err = strconv.Atoi(c.Param("cid"))
	if err != nil {
		err = errwrap.ErrInvalidArgs.SetIErrF("request parse cid failed: %s", err)
		return
	}

	name = c.Param("name")
	if len(name) == 0 {
		err = errwrap.ErrInvalidArgs.SetIErrF("picture name not found")
		return
	}

	return
}

func Picture(c *gin.Context) {
	cid, name, err := parsePictureArgs(c)
	if err != nil {
		slog.ErrorContext(c, "parsePictureArgs failed",
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

	c.File(info.PageSavePathByName(name))
}

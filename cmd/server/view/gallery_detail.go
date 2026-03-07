// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package view

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/pkg/clog"
	"github.com/cocomhub/cocom/pkg/conv"
	"github.com/cocomhub/cocom/pkg/errwrap"

	"github.com/gin-gonic/gin"
)

func parseGalleryDetailPage(c *gin.Context) (cid int, large bool, err error) {
	cid, err = strconv.Atoi(c.Param("cid"))
	if err != nil {
		err = errwrap.ErrInvalidArgs.SetIErrF("parse cid failed: %s", err)
		return
	}

	large, _ = strconv.ParseBool(c.Query("large"))
	return
}

func GalleryDetailPage(c *gin.Context) {
	cid, large, err := parseGalleryDetailPage(c)
	if err != nil {
		clog.Errorf(c, "parseGalleryDetailPage failed: %#v", err)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	info := api.ComicInfo{}
	err = comic.GetComicInfo(c, cid, &info)
	if err != nil {
		clog.Errorf(c, "comic.GetComicInfo failed: %#v", err)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	page := &GalleryDetail{ComicInfo: info, URL: c.Request.URL.Path, EnableLarge: large}
	c.HTML(http.StatusOK, "gallery_detail.tpl", page)
}

type GalleryDetail struct {
	api.ComicInfo
	EnableLarge bool
	URL         string
	CSRFToken   string
}

func (g *GalleryDetail) IsNavigationActive(name string) bool {
	return false
}

func (g *GalleryDetail) SubTypeTagsIdString(subType string) string {
	return g.Tags.SubTypeTags(subType).IdString()
}

func (g *GalleryDetail) CoverName() string {
	return g.Images.CoverName()
}

func (g *GalleryDetail) TagsTypeShowName() []string {
	return []string{"Parodies", "Characters", "Tags", "Artists", "Groups", "Languages", "Categories"}
}

func (g *GalleryDetail) UploadDate() string {
	return time.Unix(g.ComicInfo.UploadDate, 0).Format(time.RFC3339)
}

func (g *GalleryDetail) MoreLikeThis() []*GalleryDetail {
	return []*GalleryDetail{g, g, g, g, g}
}

func (g *GalleryDetail) ShowMediaId() string {
	return fmt.Sprint(g.CID)
}

func (g *GalleryDetail) GalleryRawStr() string {
	return conv.JSON(g.ComicInfo)
}

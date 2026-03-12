// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package view

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/cmd/server/internal/tag"
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
		slog.ErrorContext(c, "parseGalleryDetailPage failed",
			slog.String("errmsg", err.Error()))
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	info := api.ComicInfo{}
	err = comic.GetComicInfo(c, cid, &info)
	if err != nil {
		slog.ErrorContext(c, "comic.GetComicInfo failed",
			slog.Int64("cid", int64(cid)),
			slog.String("errmsg", err.Error()))
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	// 覆盖标签的 count，并收集 liked 状态
	liked := map[int]bool{}
	for i := range info.Tags {
		tg := info.Tags[i]
		doc, err := tag.GetTagByID(c, tg.Type, tg.ID)
		if err != nil {
			slog.WarnContext(c, "GetTagByID failed",
				slog.String("type", tg.Type),
				slog.Int64("id", int64(tg.ID)),
				slog.String("errmsg", err.Error()))
			continue
		}
		if doc == nil {
			continue
		}
		info.Tags[i].Count = doc.Count
		if doc.Like {
			liked[doc.ID] = true
		}
	}

	page := &GalleryDetail{ComicInfo: info, URL: c.Request.URL.Path, EnableLarge: large}
	page.likedTagIDs = liked
	c.HTML(http.StatusOK, "gallery_detail.tpl", page)
}

type GalleryDetail struct {
	api.ComicInfo
	EnableLarge bool
	URL         string
	CSRFToken   string
	likedTagIDs map[int]bool
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
	return []string{"Parodies", "Characters", "Tags", "Artists", "Groups", "Languages", "Categories", "Customs"}
}

func (g *GalleryDetail) UploadDate() string {
	return time.Unix(g.ComicInfo.UploadDate, 0).Format(time.RFC3339)
}

func (g *GalleryDetail) MoreLikeThis() []*GalleryDetail {
	if len(g.Tags) == 0 {
		return []*GalleryDetail{g, g, g, g, g}
	}
	list := make([]*GalleryDetail, 0, 5)
	infos, err := comic.GetMoreLikeThis(context.Background(), g.CID, g.Tags, 5)
	if err == nil && len(infos) != 0 {
		for _, info := range infos {
			list = append(list, &GalleryDetail{
				ComicInfo:   *info,
				EnableLarge: false,
				URL:         fmt.Sprintf("/g/%d/", info.CID),
			})
		}
	} else {
		slog.ErrorContext(context.Background(), "GetMoreLikeThis failed",
			slog.Int64("cid", int64(g.CID)),
			slog.String("errmsg", err.Error()),
			slog.String("tags", conv.JSON(g.Tags)))
	}
	for len(list) < 5 {
		list = append(list, g)
	}
	return list
}

func (g *GalleryDetail) ShowMediaId() string {
	return fmt.Sprint(g.CID)
}

func (g *GalleryDetail) GalleryRawStr() string {
	return conv.JSON(g.ComicInfo)
}

func (g *GalleryDetail) HasLike() bool {
	for _, t := range g.Tags {
		if t.Type == "custom" && t.Name == "like" {
			return true
		}
	}
	return false
}

func (g *GalleryDetail) IsTagLiked(t api.Tag) bool {
	return g.likedTagIDs != nil && g.likedTagIDs[t.ID]
}

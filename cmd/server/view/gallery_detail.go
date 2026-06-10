// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package view

import (
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
	"github.com/cocomhub/cocom/pkg/httpwrap"

	"github.com/gin-gonic/gin"
)

func parseGalleryDetailPage(c *gin.Context) (cid int, err error) {
	cid, err = strconv.Atoi(c.Param("cid"))
	if err != nil {
		err = errwrap.ErrInvalidArgs.SetIErrF("parse cid failed: %s", err)
		return
	}

	return
}

func GalleryDetailPage(c *gin.Context) {
	cid, err := parseGalleryDetailPage(c)
	if err != nil {
		slog.ErrorContext(c, "parseGalleryDetailPage failed",
			slog.String("errmsg", err.Error()))
		httpwrap.GinRespondError(c, http.StatusBadRequest, httpwrap.ErrCodeInvalid, "invalid request")
		c.Abort()
		return
	}

	info := api.ComicInfo{}
	err = comic.GetComicInfo(c, cid, &info)
	if err != nil {
		slog.ErrorContext(c, "comic.GetComicInfo failed",
			slog.Int64("cid", int64(cid)),
			slog.String("errmsg", err.Error()))
		httpwrap.GinRespondError(c, http.StatusBadRequest, httpwrap.ErrCodeInvalid, "invalid request")
		c.Abort()
		return
	}

	// 检查漫画是否已被删除
	if info.Deleted {
		c.HTML(http.StatusOK, "page_deleted.tpl", gin.H{
			"CID": cid,
		})
		return
	}

	// 检查是否有重定向（从属漫画）
	if info.RedirectTo != nil && *info.RedirectTo > 0 {
		c.Redirect(http.StatusFound, fmt.Sprintf("/g/%d/", *info.RedirectTo))
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

	page := &GalleryDetail{ComicInfo: info, URL: c.Request.URL.Path}
	page.ArchiveStale = info.Archive != nil && info.Archive.Status == "stale"
	page.likedTagIDs = liked
	c.HTML(http.StatusOK, "gallery_detail.tpl", page)
}

type GalleryDetail struct {
	api.ComicInfo
	URL         string
	CSRFToken   string
	likedTagIDs map[int]bool
	ArchiveStale bool
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

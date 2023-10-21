/*
Copyright © 2023 suixibing <suixibing@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package view

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/suixibing/cocom/cmd/server/api"
	"github.com/suixibing/cocom/cmd/server/internal/comic"
	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/conv"

	"github.com/gin-gonic/gin"
)

func GalleryDetailPage(c *gin.Context) {
	cid, err := strconv.Atoi(c.Param("cid"))
	if err != nil {
		clog.Errorf(c, "request parse cid failed. errmsg: %s", err)
		c.AbortWithError(-1, err)
		return
	}

	info := api.ComicInfo{}
	err = comic.GetComicInfo(c, cid, &info)
	if err != nil {
		clog.Errorf(c, "get comic info failed. errmsg: %s", err)
		c.AbortWithError(-1, fmt.Errorf("get comic info failed. errmsg: %s", err))
		return
	}

	page := &GalleryDetail{ComicInfo: info}
	c.HTML(200, "gallery_detail.tpl", page)
}

type GalleryDetail struct {
	api.ComicInfo
	CSRFToken string
}

func (g *GalleryDetail) SubTypeTagsIdString(subType string) string {
	return g.Tags.SubTypeTags(subType).IdString()
}

func (g *GalleryDetail) CoverName() string {
	return g.Images.CoverName()
}

func (g *GalleryDetail) TitleBefore() string {
	return strings.TrimSpace(g.Title.English[:strings.Index(g.Title.English, "] ")+2])
}

func (g *GalleryDetail) TitlePretty() string {
	pretty := g.Title.English[strings.Index(g.Title.English, "] ")+2:]
	return strings.TrimSpace(pretty[:strings.IndexAny(pretty, "([")])
}

func (g *GalleryDetail) TitleAfter() string {
	pretty := g.Title.English[strings.Index(g.Title.English, "] ")+2:]
	return strings.TrimSpace(pretty[strings.IndexAny(pretty, "(["):])
}

func (g *GalleryDetail) TitleBefore2() string {
	return strings.TrimSpace(g.Title.Japanese[:strings.Index(g.Title.Japanese, "] ")+2])
}

func (g *GalleryDetail) TitlePretty2() string {
	pretty := g.Title.Japanese[strings.Index(g.Title.Japanese, "] ")+2:]
	return strings.TrimSpace(pretty[:strings.IndexAny(pretty, "([")])
}

func (g *GalleryDetail) TitleAfter2() string {
	pretty := g.Title.Japanese[strings.Index(g.Title.Japanese, "] ")+2:]
	return strings.TrimSpace(pretty[strings.IndexAny(pretty, "(["):])
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

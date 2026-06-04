// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package view

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/cmd/server/internal/tag"
	"github.com/cocomhub/cocom/pkg/httpwrap"

	"github.com/gin-gonic/gin"
)

func parseTagListResultPageArgs(c *gin.Context) (page int, tagType string, sortType int, likedOnly bool) {
	page, _ = strconv.Atoi(c.Query("page"))
	if page <= 0 {
		page = 1
	}

	switch c.Param("tagType") {
	case "tags":
		tagType = "tag"
	case "artists":
		tagType = "artist"
	case "characters":
		tagType = "character"
	case "parodies":
		tagType = "parody"
	case "groups":
		tagType = "group"
	default:
		tagType = "tag"
	}

	switch c.Query("sortType") {
	case "popular":
		sortType = comic.SortTypeByPopular
	default:
		sortType = comic.SortTypeByName
	}
	likedOnly = false
	if lo := c.Query("likedOnly"); lo == "true" || lo == "1" {
		likedOnly = true
	}
	return
}

func TagListResultPage(c *gin.Context) {
	page, tagType, sortType, likedOnly := parseTagListResultPageArgs(c)

	p, err := NewTagListPage(c, c.Request.URL.Path, tagType, page, sortType, likedOnly)
	if err != nil {
		slog.ErrorContext(c, "TagListResultPage failed",
			slog.String("errmsg", err.Error()))
		httpwrap.GinRespondError(c, http.StatusBadRequest, httpwrap.ErrCodeInvalid, "resource not found")
		c.Abort()
		return
	}

	c.HTML(http.StatusOK, "tag_list_result.tpl", p)
}

func NewTagListPage(ctx context.Context, url string, tagType string, curPage int, sortType int, likedOnly bool) (*TagListPage, error) {
	p := &TagListPage{
		URL:        url,
		TagType:    tagType,
		CurPage:    curPage,
		PageTagNum: DefaultPageTagNum,
		SortType:   sortType,
		LikedOnly:  likedOnly,
	}

	err := p.initTagList(ctx)
	if err != nil {
		return nil, err
	}

	switch sortType {
	case comic.SortTypeByName:
		tagIndices, err := tag.AggregateTagSectionIndices(ctx, p.TagType, p.PageTagNum, p.LikedOnly)
		if err != nil {
			return nil, err
		}
		p.TagIndices = tagIndices
	case comic.SortTypeByPopular:
	}

	return p, nil
}

var DefaultPageTagNum = 100

type TagListPage struct {
	URL        string
	CSRFToken  string
	TagType    string
	TagIndices []*api.TagSectionIndex
	CurPage    int
	LastPage   int
	PageTagNum int
	SortType   int
	Tags       []*api.TagInfo
	Total      int64
	LikedOnly  bool
}

func (p *TagListPage) IsNavigationActive(name string) bool {
	return p.TagType == name
}

func (p *TagListPage) initTagList(ctx context.Context) error {
	tags, total, err := tag.AggregateTagList(ctx, p.TagType, p.SortType, int64(p.PageTagNum*(p.CurPage-1)), int64(p.PageTagNum), p.LikedOnly)
	if err != nil {
		return err
	}
	p.Tags = tags
	p.Total = total
	p.LastPage = int(total/int64(p.PageTagNum)) + 1
	return nil
}

type TagsSection struct {
	Name string
	Tags []*api.TagInfo
}

func (p *TagListPage) TagsSections() []TagsSection {
	if len(p.Tags) == 0 {
		return nil
	}

	sections := make([]TagsSection, 1)
	sections[0] = TagsSection{
		Name: p.Tags[0].SectionName(),
		Tags: p.Tags[0:],
	}

	if p.Tags[len(p.Tags)-1].SectionName() == sections[0].Name {
		return sections
	}

	idx, left := 0, 0
	for i := 1; i < len(p.Tags); i++ {
		if sections[idx].Name == p.Tags[i].SectionName() {
			continue
		}
		sections[idx].Tags = p.Tags[left:i]

		idx++
		left = i
		sections = append(sections, TagsSection{
			Name: p.Tags[i].SectionName(),
			Tags: p.Tags[i:],
		})
	}
	return sections
}

func (p *TagListPage) PageNumList() (list []int) {
	return PageNumList(p.LastPage, p.CurPage)
}


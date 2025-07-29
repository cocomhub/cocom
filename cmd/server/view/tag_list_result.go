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
	"context"
	"net/http"
	"strconv"

	"github.com/suixibing/cocom/cmd/server/api"
	"github.com/suixibing/cocom/cmd/server/internal/comic"
	"github.com/suixibing/cocom/pkg/clog"

	"github.com/gin-gonic/gin"
)

func parseTagListResultPageArgs(c *gin.Context) (page int, tagType string, sortType int) {
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
	return
}

func TagListResultPage(c *gin.Context) {
	page, tagType, sortType := parseTagListResultPageArgs(c)

	p, err := NewTagListPage(c, c.Request.URL.Path, tagType, page, sortType)
	if err != nil {
		clog.Errorf(c, "TagListResultPage failed: %#v", err)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	c.HTML(http.StatusOK, "tag_list_result.tpl", p)
}

func NewTagListPage(ctx context.Context, url string, tagType string, curPage int, sortType int) (*TagListPage, error) {
	p := &TagListPage{
		URL:        url,
		TagType:    tagType,
		CurPage:    curPage,
		PageTagNum: DefaultPageTagNum,
		SortType:   sortType,
	}

	err := p.initTagList(ctx)
	if err != nil {
		return nil, err
	}

	switch sortType {
	case comic.SortTypeByName:
		tagIndices, err := comic.AggregateTagSectionIndices(ctx, p.TagType, p.PageTagNum)
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
}

func (p *TagListPage) IsNavigationActive(name string) bool {
	return p.TagType == name
}

func (p *TagListPage) initTagList(ctx context.Context) error {
	tags, total, err := comic.AggregateTagList(ctx, p.TagType, p.SortType,
		int64(p.CurPage-1)*int64(p.PageTagNum), int64(p.PageTagNum))
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

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

	"github.com/suixibing/cocom/cmd/server/internal/comic"
	"github.com/suixibing/cocom/cmd/server/internal/setting"
	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/errwrap"
	"github.com/suixibing/cocom/pkg/util"

	"github.com/gin-gonic/gin"
)

func parseIndexPageArgs(c *gin.Context) (page int, err error) {
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
	return
}

func IndexPage(c *gin.Context) {
	page, err := parseIndexPageArgs(c)
	if err != nil {
		clog.Errorf(c, "parseIndexPageArgs failed: %#v", err)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	indexInfo, err := NewGalleryIndexPage(c, c.Request.URL.Path, page)
	if err != nil {
		clog.Errorf(c, "NewGalleryIndexPage failed: %#v", err)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	c.HTML(http.StatusOK, "index.tpl", indexInfo)
}

var (
	DefaultPageComicNum    = 20
	DefaultPopulorComicNum = 5
)

func NewGalleryIndexPage(ctx context.Context, url string, page int, filters ...interface{}) (*GalleryIndexPage, error) {
	p := &GalleryIndexPage{URL: url}

	p.initConfig(ctx)

	if !p.cfg.ShowStatusNotTrue {
		filters = append(filters, "status", true)
	}

	err := p.initPageNum(ctx, page, filters...)
	if err != nil {
		return nil, err
	}

	err = p.initComicInfos(ctx, filters...)
	if err != nil {
		return nil, err
	}
	return p, nil
}

type GalleryIndexPageConfig struct {
	ShowStatusNotTrue bool
}

type GalleryIndexPage struct {
	PopularNow  []*GalleryDetail
	NewUpdates  []*GalleryDetail
	URL         string
	SearchQuery string
	cfg         *GalleryIndexPageConfig
	Total       int
	CurPage     int
	LastPage    int
}

func (p *GalleryIndexPage) initConfig(ctx context.Context) {
	p.cfg = &GalleryIndexPageConfig{
		ShowStatusNotTrue: false,
	}

	settings, err := setting.GetSettings(ctx, "view", "show_status_not_true")
	if err != nil {
		clog.Warnf(ctx, "GalleryIndexPage.initConfig get settings failed. errmsg:%s", err.Error())
		return
	}

	if len(settings) == 0 {
		return
	}

	showStatusNotTrue, ok := settings["show_status_not_true"].(bool)
	if ok {
		p.cfg.ShowStatusNotTrue = showStatusNotTrue
	}
}

func (p *GalleryIndexPage) initPageNum(ctx context.Context, page int, filters ...interface{}) error {
	total, err := comic.CountTotalComicInfos(ctx, filters...)
	if err != nil {
		return err
	}
	p.Total = int(total)

	p.LastPage = p.Total / DefaultPageComicNum
	if p.Total%DefaultPageComicNum > 0 {
		p.LastPage++
	}

	if page < 0 || page > p.LastPage {
		return errwrap.New(-1, "invalid page").SetIErrF("page[%d] lastPage[%d]", page, p.LastPage)
	}
	p.CurPage = page
	return nil
}

func (p *GalleryIndexPage) initComicInfos(ctx context.Context, filters ...interface{}) error {
	pageNum := DefaultPageComicNum
	infos, err := comic.GetRangeComicInfos(ctx, int64(pageNum), int64(pageNum*(p.CurPage-1)), filters...)
	if err != nil {
		return err
	}
	clog.Debugf(ctx, "comic.GetRangeComicInfos length[%d]", len(infos))

	if p.CurPage == 1 {
		popularNum := DefaultPopulorComicNum
		if len(infos) < popularNum {
			popularNum = len(infos)
		}
		for _, info := range infos[:popularNum] {
			p.PopularNow = append(p.PopularNow, &GalleryDetail{ComicInfo: *info})
		}
	}

	if len(infos) < pageNum {
		pageNum = len(infos)
	}
	for _, info := range infos[:pageNum] {
		p.NewUpdates = append(p.NewUpdates, &GalleryDetail{ComicInfo: *info})
	}

	return nil
}

func (p *GalleryIndexPage) PageNumList() (list []int) {
	if p.CurPage < 1 || p.LastPage < 1 || p.CurPage > p.LastPage {
		return nil
	}

	left := p.CurPage - 5
	if left < 1 {
		left = 1
	}

	right := p.CurPage + 5
	if right > p.LastPage {
		right = p.LastPage
	}

	list = make([]int, right-left+1)
	util.FillIncrNum(list, left, 1)
	return
}

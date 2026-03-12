// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package view

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/cmd/server/internal/setting"
	"github.com/cocomhub/cocom/pkg/errwrap"
	"github.com/cocomhub/cocom/pkg/util"

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
		slog.ErrorContext(c, "parseIndexPageArgs failed",
			slog.String("errmsg", err.Error()))
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	indexInfo, err := NewGalleryIndexPage(c, c.Request.URL.Path, page)
	if err != nil {
		slog.ErrorContext(c, "NewGalleryIndexPage failed",
			slog.String("errmsg", err.Error()))
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	c.HTML(http.StatusOK, "index.tpl", indexInfo)
}

var (
	DefaultPageComicNum    = 20
	DefaultPopulorComicNum = 5
)

func NewGalleryIndexPage(ctx context.Context, url string, page int, filters ...any) (*GalleryIndexPage, error) {
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
	CurTag      *TagMeta
}

func (p *GalleryIndexPage) initConfig(ctx context.Context) {
	p.cfg = &GalleryIndexPageConfig{
		ShowStatusNotTrue: false,
	}

	settings, err := setting.GetSettings(ctx, "view", "show_status_not_true")
	if err != nil {
		slog.WarnContext(ctx, "GalleryIndexPage.initConfig get settings failed",
			slog.String("errmsg", err.Error()))
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

func (p *GalleryIndexPage) initPageNum(ctx context.Context, page int, filters ...any) error {
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

func (p *GalleryIndexPage) initComicInfos(ctx context.Context, filters ...any) error {
	pageNum := DefaultPageComicNum
	infos, err := comic.GetRangeComicInfos(ctx, int64(pageNum), int64(pageNum*(p.CurPage-1)), filters...)
	if err != nil {
		return err
	}
	slog.DebugContext(ctx, "comic.GetRangeComicInfos", slog.Int("length", len(infos)))

	if p.CurPage == 1 {
		popularNum := min(len(infos), DefaultPopulorComicNum)
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

func (p *GalleryIndexPage) IsNavigationActive(name string) bool {
	return false
}

func (p *GalleryIndexPage) PageNumList() (list []int) {
	return PageNumList(p.LastPage, p.CurPage)
}

func PageNumList(total, curPage int) (list []int) {
	if curPage < 1 || total < 1 || curPage > total {
		return nil
	}

	left := max(curPage-5, 1)

	right := min(curPage+5, total)

	list = make([]int, right-left+1)
	util.FillIncrNum(list, left, 1)
	return
}

type TagMeta struct {
	Type string
	ID   int
	Name string
	URL  string
	Like bool
}

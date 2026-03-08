// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package view

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"github.com/cocomhub/cocom/pkg/middlewares"
	"github.com/cocomhub/cocom/pkg/util"

	"github.com/gin-gonic/gin"
)

//go:embed static/*
var embedFS embed.FS

var staticFS fs.FS

func init() {
	var err error
	staticFS, err = fs.Sub(embedFS, "static")
	if err != nil {
		panic(any(err))
	}
}

func Register(r *gin.Engine) {
	r.SetHTMLTemplate(Template())

	r.StaticFS("/static", http.FS(staticFS))
	// r.StaticFS("/galleries/", gin.Dir("./galleries", false))
	r.GET("/galleries/:cid/:name", Picture)
	r.HEAD("/galleries/:cid/:name", Picture)
	r.GET("/", IndexPage)
	r.HEAD("/", IndexPage)
	r.GET("/tag/:tag/:name", TagResultPage)
	r.HEAD("/tag/:tag/:name", TagResultPage)
	r.GET("/g/:cid", GalleryDetailPage)
	r.HEAD("/g/:cid", GalleryDetailPage)
	r.GET("/g/:cid/:no", GalleryPicturePage)
	r.HEAD("/g/:cid/:no", GalleryPicturePage)
	r.GET("/search", SearchResultPage)
	r.HEAD("/search", SearchResultPage)
	r.GET("/list/:tagType", TagListResultPage)
	r.HEAD("/list/:tagType", TagListResultPage)
	// 管理界面入口
	r.GET("/admin", middlewares.LocalGuard("admin.allow_remote"), AdminPage)
}

var funcMap = template.FuncMap{
	"Add":         Add,
	"Tag":         RawStr("tag"),
	"Artist":      RawStr("artist"),
	"TitleBefore": TitleBefore,
	"TitlePretty": TitlePretty,
	"TitleAfter":  TitleAfter,
	"TagTypeList": TagTypeList,
}

func Add(a, b int) int {
	return a + b
}

func RawStr(raw string) func() string {
	return func() string {
		return raw
	}
}

func Template() *template.Template {
	tpl, err := template.New("tpl").Funcs(funcMap).ParseFS(staticFS, "tpl/*.tpl")
	if err != nil {
		panic(any(err))
	}
	return tpl
}

func TitleBefore(title string) string {
	return strings.TrimSpace(title[:strings.Index(title, "]")+1])
}

func TitlePretty(title string) string {
	title = util.SafeSubStr(title, strings.Index(title, "]")+1, 0)
	return strings.TrimSpace(util.SafeSubStr(title, 0, strings.IndexAny(title, "([")))
}

func TitleAfter(title string) string {
	title = util.SafeSubStr(title, strings.Index(title, "]")+1, 0)
	return strings.TrimSpace(util.SafeSubStr(title, strings.IndexAny(title, "(["), 0))
}

type tagType struct {
	Name      string
	FieldName string
}

var tagTypes = []tagType{
	{"parody", "Parodies"},
	{"character", "Characters"},
	{"tag", "Tags"},
	{"artist", "Artists"},
	{"group", "Groups"},
	{"language", "Languages"},
	{"category", "Categories"},
	{"custom", "Customs"},
}

func TagTypeList() []tagType {
	return tagTypes
}

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/cocomhub/cocom/cmd/server/handler"
	"github.com/cocomhub/cocom/cmd/server/view"
	"github.com/cocomhub/cocom/tests/e2e/fixtures"
	"github.com/cocomhub/cocom/tests/e2e/helpers"
	comicpkg "github.com/cocomhub/cocom/pkg/comic"
	"github.com/gin-gonic/gin"
	"github.com/playwright-community/playwright-go"
)

var (
	testServer   *httptest.Server
	pw           *playwright.Playwright
	browser      playwright.Browser
	testMemStore *comicpkg.MemoryStorage
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// 初始化内存存储（通过 handler 包桥接，避免导入 internal 包）
	testMemStore = handler.InitE2EStorage()

	// 创建临时 gallery 目录
	tmpDir, err := os.MkdirTemp("", "cocom-e2e-gallery-*")
	if err != nil {
		slog.Error("failed to create temp dir", "err", err)
		os.Exit(1)
	}

	// 设置环境变量使 config.GetSaveRoot() 返回临时目录
	os.Setenv("COCOM_STORAGE_GALLERY", tmpDir)
	os.Setenv("COCOM_STORAGE_ARCHIVE", tmpDir)
	os.Setenv("COCOM_STORAGE_ARCHIVE_TEMP", tmpDir)

	// 播种种子数据到 MemoryStorage
	handler.SeedTestData(ctx, testMemStore)

	// 生成 mock 图片文件
	if err := fixtures.SeedE2EData(ctx, testMemStore, tmpDir); err != nil {
		slog.Error("seed E2E images failed", "err", err)
		os.Exit(1)
	}

	// 构建 Gin Engine（不触发 handler.Init 中的 mongowrap）
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// 手动写入 RequestID 中间件（简化版，不依赖 pkg/middlewares import）
	r.Use(func(c *gin.Context) {
		c.Next()
	})

	// 注册 view 路由（包含 /admin 的 LocalGuard）
	view.Register(r)

	// 手动注册 handler（避免 mongowrap 依赖）
	r.POST("/api/admin/comic/compare", gin.WrapF(handler.CompareComics))
	r.POST("/api/admin/comic/link", gin.WrapF(handler.LinkComics))
	r.POST("/api/admin/comic/unlink", gin.WrapF(handler.UnlinkComics))
	r.GET("/api/admin/comic/links", gin.WrapF(handler.GetLinks))
	r.POST("/api/admin/comic/delete", gin.WrapF(handler.DeleteComic))

	r.POST("/api/comic/getComicInfo", gin.WrapF(handler.GetComicInfo))
	r.GET("/api/comic/getComicInfo", gin.WrapF(handler.GetComicInfo))
	r.POST("/api/comic/addLikeGroup", gin.WrapF(handler.AddLikeGroup))
	r.GET("/api/search/autocomplete", gin.WrapF(handler.SearchAutocomplete))
	r.GET("/api/comic/tags/search", gin.WrapF(handler.SearchTags))
	r.GET("/api/comic/recommendations", handler.GetRecommendations)
	r.POST("/api/comic/tags/like", gin.WrapF(handler.AddLikeTag))
	r.DELETE("/api/comic/tags/like", gin.WrapF(handler.RemoveLikeTag))
	r.POST("/api/comic/download", gin.WrapF(handler.DownloadComic))
	r.POST("/api/comic/restore", gin.WrapF(handler.RestoreComic))

	testServer = httptest.NewServer(r)

	// 启动 playwright
	pw, err = playwright.Run()
	if err != nil {
		slog.Error("playwright run failed", "err", err)
		testServer.Close()
		os.Exit(1)
	}
	browser, err = pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		slog.Error("chromium launch failed", "err", err)
		pw.Stop()
		testServer.Close()
		os.Exit(1)
	}

	exitCode := m.Run()

	// 清理
	browser.Close()
	pw.Stop()
	testServer.Close()
	os.RemoveAll(tmpDir)

	os.Exit(exitCode)
}

// newPage 为每个测试创建一个新页面，返回 page 和 cleanup 函数
func newPage(t *testing.T) (playwright.Page, func()) {
	t.Helper()
	context, err := browser.NewContext()
	if err != nil {
		t.Fatalf("create browser context: %v", err)
	}
	page, err := context.NewPage()
	if err != nil {
		t.Fatalf("create page: %v", err)
	}
	page.SetDefaultTimeout(10000)
	return page, func() {
		if t.Failed() {
			helpers.TakeScreenshot(t, page, "fail")
		}
		context.Close()
	}
}

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/cocomhub/cocom/cmd/server/handler"
	"github.com/cocomhub/cocom/cmd/server/view"
	comicpkg "github.com/cocomhub/cocom/pkg/comic"
	_ "github.com/cocomhub/cocom/pkg/comic/storage"
	"github.com/cocomhub/cocom/tests/e2e/fixtures"
	"github.com/cocomhub/cocom/tests/e2e/helpers"
	"github.com/gin-gonic/gin"
	"github.com/playwright-community/playwright-go"
)

var (
	testServer   *httptest.Server
	testMemStore *comicpkg.MemoryStorage
	pw           *playwright.Playwright
	browser      playwright.Browser
)

// TestMain 启动 Gin TestServer + Playwright Chromium + 种子数据
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	// 确保信号处理不受测试影响
	signal.Ignore(syscall.SIGHUP)

	ctx := context.Background()

	// 创建内存存储并注入
	testMemStore = handler.InitE2EStorage()

	// 创建临时 gallery 目录
	tmpDir, err := os.MkdirTemp("", "cocom-e2e-gallery-*")
	if err != nil {
		slog.Error("failed to create temp dir", "err", err)
		os.Exit(1)
	}

	// 保存原环境变量，确保退出时恢复（避免泄漏到其他测试）
	origGallery := os.Getenv("COCOM_STORAGE_GALLERY")
	origArchive := os.Getenv("COCOM_STORAGE_ARCHIVE")
	origArchiveTemp := os.Getenv("COCOM_STORAGE_ARCHIVE_TEMP")

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

	// 创建截图目录
	if err := os.MkdirAll(helpers.ScreenshotDir, 0o755); err != nil {
		slog.Error("create screenshot dir failed", "err", err)
		os.Exit(1)
	}

	// 验证种子数据加载成功：至少能查出关键 CID
	for _, cid := range []string{"1001", "2001", "3001"} {
		if _, err := testMemStore.Get(ctx, cid); err != nil {
			slog.Error("seed data verification failed", "cid", cid, "err", err)
			os.Exit(1)
		}
	}

	// 注册 Gin 路由
	r := gin.New()
	r.Use(gin.Recovery())
	view.Register(r)

	// 手动注册 handler 函数（避免调用 handler.Init 触发 mongowrap.Init）
	//
	// 注意：前端 JS 调用的 gallery-actions.js 中 like/archive/restore 走的是
	// `/v2/api/nhcomic/:cid/(archive|restore)` 路由，由 pkg/comic.Handler 注册。
	// 当前 TestMain 不注册 v2 路由（需要 Service 实例），因此 gallery_detail_test.go
	// 中 Like/Archive/Restore 按钮交互仅验证 DOM 可见性，不点击触发 XHR。
	apiGroup := r.Group("/api")
	{
		apiGroup.GET("/search", gin.WrapF(handler.SearchAutocomplete))
		apiGroup.GET("/search/tags", gin.WrapF(handler.SearchTags))
		apiGroup.GET("/search/autocomplete", gin.WrapF(handler.SearchAutocomplete))
	}
	adminGroup := r.Group("/admin")
	{
		adminGroup.POST("/compare", gin.WrapF(handler.CompareComics))
		adminGroup.POST("/swap", gin.WrapF(handler.LinkComics))
	}
	galleryGroup := r.Group("/g")
	{
		galleryGroup.POST("/api/like", gin.WrapF(handler.LikeTag))
		galleryGroup.POST("/api/archive", gin.WrapF(handler.AddLikeGroup))
		galleryGroup.POST("/api/restore", gin.WrapF(handler.RestoreComic))
	}

	// 启动 TestServer（随机端口）
	testServer = httptest.NewServer(r)
	defer testServer.Close()

	// 启动 Playwright
	pw, err = playwright.Run()
	if err != nil {
		slog.Error("could not start playwright", "err", err)
		os.Exit(1)
	}
	defer pw.Stop()

	browser, err = pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		slog.Error("could not launch Chromium", "err", err)
		os.Exit(1)
	}
	defer browser.Close()

	// 运行测试
	exitCode := m.Run()

	// 恢复环境变量
	if origGallery != "" {
		os.Setenv("COCOM_STORAGE_GALLERY", origGallery)
	} else {
		os.Unsetenv("COCOM_STORAGE_GALLERY")
	}
	if origArchive != "" {
		os.Setenv("COCOM_STORAGE_ARCHIVE", origArchive)
	} else {
		os.Unsetenv("COCOM_STORAGE_ARCHIVE")
	}
	if origArchiveTemp != "" {
		os.Setenv("COCOM_STORAGE_ARCHIVE_TEMP", origArchiveTemp)
	} else {
		os.Unsetenv("COCOM_STORAGE_ARCHIVE_TEMP")
	}

	// 清理临时目录
	os.RemoveAll(tmpDir)

	os.Exit(exitCode)
}

// newPage 为每个测试创建新的 BrowserContext + Page 以提供隔离
func newPage(t *testing.T) (playwright.Page, func()) {
	t.Helper()
	context, err := browser.NewContext()
	if err != nil {
		t.Fatalf("could not create browser context: %v", err)
	}
	page, err := context.NewPage()
	if err != nil {
		context.Close()
		t.Fatalf("could not create page: %v", err)
	}
	return page, func() {
		page.Close()
		context.Close()
	}
}

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

	// 注册 Gin 路由 — 复用生产代码路由注册
	r := gin.New()
	r.Use(gin.Recovery())
	view.Register(r)

	handler.RegisterE2ERoutesWithStore(ctx, r, testMemStore)

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
	page.SetDefaultTimeout(10000)
	helpers.InjectTestMode(t, page)
	return page, func() {
		page.Close()
		context.Close()
	}
}

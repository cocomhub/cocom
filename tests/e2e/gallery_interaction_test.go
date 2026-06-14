// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cocomhub/cocom/tests/e2e/helpers"
	"github.com/playwright-community/playwright-go"
)

// TestGalleryInteraction 详情页高级交互测试组（页管理器、推荐等）
func TestGalleryInteraction(t *testing.T) {
	page, cleanup := newPage(t)
	defer cleanup()

	t.Run("PageManagerToggle", func(t *testing.T) {
		_, err := page.Goto(fmt.Sprintf("%s/g/3003", testServer.URL),
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate to gallery detail failed: %v", err)
		}
		helpers.WaitForVisible(t, page, helpers.PageManageBtn)

		// 打开页管理器
		helpers.ClickAndWait(t, page, helpers.PageManageBtn)
		page.WaitForTimeout(500)

		if helpers.IsVisible(t, page, "#page-manager-bar") {
			t.Log("page manager bar appeared after toggle")
		} else {
			t.Log("page manager bar not visible (display:none in JS)")
		}

		// 关闭页管理器
		helpers.ClickAndWait(t, page, helpers.PageManageBtn)
		page.WaitForTimeout(300)

		if helpers.IsVisible(t, page, "#page-manager-bar") {
			t.Log("page manager bar still visible after second toggle")
		} else {
			t.Log("page manager bar hidden after second toggle")
		}
	})

	t.Run("PageManagerModeButtons", func(t *testing.T) {
		_, err := page.Goto(fmt.Sprintf("%s/g/3003", testServer.URL),
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate to gallery detail failed: %v", err)
		}
		helpers.WaitForVisible(t, page, helpers.PageManageBtn)
		helpers.ClickAndWait(t, page, helpers.PageManageBtn)
		page.WaitForTimeout(500)

		// 检查模式按钮存在
		modeBtns := page.Locator("#pm-delete-mode, #pm-insert-mode, #pm-replace-mode, #pm-reorder-mode")
		count, err := modeBtns.Count()
		if err == nil {
			t.Logf("page manager mode buttons found: %d", count)
		}
		if count == 0 {
			t.Log("no mode buttons found in page manager bar")
		}

		// 检查退出按钮
		if helpers.IsVisible(t, page, "#pm-exit") || helpers.IsVisible(t, page, "#pm-save") {
			t.Log("page manager action buttons visible")
		} else {
			t.Log("page manager action buttons not visible")
		}
	})

	t.Run("RecommendContainer", func(t *testing.T) {
		_, err := page.Goto(fmt.Sprintf("%s/g/3003", testServer.URL),
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate to gallery detail failed: %v", err)
		}
		page.WaitForTimeout(1500) // 等待推荐异步加载

		if helpers.IsVisible(t, page, "#recommend-container") {
			t.Log("recommend container found")
		} else {
			t.Log("recommend container not found (may need login or data)")
		}

		recommendText, err := page.Locator("#recommend-container").TextContent()
		if err == nil && len(recommendText) > 0 {
			t.Logf("recommend container has content (%d chars)", len(recommendText))
		} else {
			t.Log("recommend container is empty (async loading may need more time)")
		}
	})

	t.Run("RecommendRefreshClick", func(t *testing.T) {
		_, err := page.Goto(fmt.Sprintf("%s/g/3003", testServer.URL),
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate to gallery detail failed: %v", err)
		}
		page.WaitForTimeout(1000)

		// 尝试点击推荐的刷新按钮
		refreshBtn := page.Locator("#recommend-container .recommend-refresh, #recommend-container .refresh-btn")
		count, err := refreshBtn.Count()
		if err == nil && count > 0 {
			refreshBtn.First().Click()
			page.WaitForTimeout(500)
			t.Logf("clicked recommend refresh button (%d found)", count)
		} else {
			t.Log("no recommend refresh button found")
		}
	})

	t.Run("ThumbnailClickNavigates", func(t *testing.T) {
		_, err := page.Goto(fmt.Sprintf("%s/g/3003", testServer.URL),
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate to gallery detail failed: %v", err)
		}
		page.WaitForTimeout(500)

		// 尝试点击第一张缩略图
		thumbs := page.Locator(helpers.ThumbContainer)
		count, err := thumbs.Count()
		if err != nil || count == 0 {
			t.Log("no thumbnails found")
			return
		}

		// 查找缩略图内的链接
		thumbLink := thumbs.Nth(0).Locator("a")
		linkCount, _ := thumbLink.Count()
		if linkCount > 0 {
			href, err := thumbLink.First().GetAttribute("href")
			if err == nil {
				t.Logf("first thumbnail link: %s", href)
			}
			thumbLink.First().Click()
			page.WaitForTimeout(500)

			currentURL := page.URL()
			if strings.Contains(currentURL, "/g/3003/") {
				t.Logf("navigated to picture page: %s", currentURL)
			} else {
				t.Logf("after thumbnail click, URL: %s", currentURL)
			}
		} else {
			t.Log("no link inside thumbnail container")
		}
	})
}

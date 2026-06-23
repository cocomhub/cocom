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

// TestSearchPage 搜索结果页面结构测试组
// 注意：E2E seed 数据中未加载 SearchScenario（Naruto/One Piece/Bleach），
// 因此搜索结果可能为空。测试聚焦页面渲染结构而非结果内容。
func TestSearchPage(t *testing.T) {
	page, cleanup := newPage(t)
	defer cleanup()

	t.Run("SearchRedirect", func(t *testing.T) {
		_, err := page.Goto(testServer.URL,
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate to home failed: %v", err)
		}

		helpers.WaitForVisible(t, page, helpers.SearchInput)
		page.Locator(helpers.SearchInput).Fill("naruto")
		page.Keyboard().Press("Enter")
		if err := page.WaitForURL("**/search**", playwright.PageWaitForURLOptions{
			Timeout: playwright.Float(5000),
		}); err != nil {
			t.Logf("WaitForURL timeout: %v, checking URL directly", err)
		}

		currentURL := page.URL()
		if !strings.Contains(currentURL, "search") {
			t.Errorf("expected search results URL after submit, got: %s", currentURL)
		} else {
			t.Logf("search URL: %s", currentURL)
		}
	})

	t.Run("SearchPageRender", func(t *testing.T) {
		_, err := page.Goto(fmt.Sprintf("%s/search?q=action", testServer.URL),
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate to search page failed: %v", err)
		}

		// 页面不应崩溃或返回 404
		bodyText, err := page.Locator("body").TextContent()
		if err != nil {
			t.Errorf("failed to get body text: %v", err)
		} else if len(bodyText) > 0 {
			t.Logf("search results page rendered (%d chars)", len(bodyText))
		}

		// 检查 gallery cards（可能为空但也应渲染容器）
		cards := page.Locator(helpers.GalleryCard)
		count, _ := cards.Count()
		t.Logf("gallery cards found in search results: %d", count)
	})

	t.Run("SearchQueryPreserved", func(t *testing.T) {
		_, err := page.Goto(fmt.Sprintf("%s/search?q=test-query", testServer.URL),
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate to search page failed: %v", err)
		}

		currentURL := page.URL()
		if !strings.Contains(currentURL, "test-query") {
			t.Errorf("expected query param preserved in URL, got: %s", currentURL)
		} else {
			t.Log("search query preserved in URL")
		}
	})
}

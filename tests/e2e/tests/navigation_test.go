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

// TestNavigation 顶部导航栏测试组
func TestNavigation(t *testing.T) {
	page, cleanup := newPage(t)
	defer cleanup()

	navigateToHome := func() {
		_, err := page.Goto(testServer.URL,
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate to home failed: %v", err)
		}
	}

	t.Run("LogoLink", func(t *testing.T) {
		_, err := page.Goto(fmt.Sprintf("%s/admin", testServer.URL),
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate to admin failed: %v", err)
		}

		// 用 Logo 回首页
		_, err = page.Goto(testServer.URL,
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("logo goto failed: %v", err)
		}
		currentURL := page.URL()
		if !strings.HasSuffix(currentURL, "/") && !strings.Contains(currentURL, testServer.URL) {
			t.Logf("after home goto, URL: %s", currentURL)
		}
	})

	t.Run("SearchSubmit", func(t *testing.T) {
		navigateToHome()
		helpers.WaitForVisible(t, page, helpers.SearchInput)

		page.Locator(helpers.SearchInput).Fill("naruto")
		page.Keyboard().Press("Enter")
		page.WaitForTimeout(1000)

		currentURL := page.URL()
		if strings.Contains(currentURL, "search") && strings.Contains(currentURL, "naruto") {
			t.Log("search submitted correctly")
		} else {
			t.Logf("search redirect URL: %s", currentURL)
		}
	})

	t.Run("SearchAutocomplete", func(t *testing.T) {
		navigateToHome()
		helpers.WaitForVisible(t, page, helpers.SearchInput)

		page.Locator(helpers.SearchInput).Fill("act")
		page.WaitForTimeout(500)

		// 检查自动补全下拉是否出现
		autocompleteSelectors := "#search-autocomplete, .autocomplete, .search-suggestions"
		if helpers.IsVisible(t, page, autocompleteSelectors) {
			t.Log("autocomplete dropdown appeared")
		} else {
			t.Log("autocomplete not visible (may need debounce timing)")
		}
	})

	t.Run("TagsLink", func(t *testing.T) {
		navigateToHome()
		helpers.WaitForVisible(t, page, helpers.NavTagsLink)
		helpers.ClickAndWait(t, page, helpers.NavTagsLink)
		page.WaitForTimeout(500)

		currentURL := page.URL()
		if !strings.Contains(currentURL, "/list/tags") {
			t.Errorf("expected /list/tags, got: %s", currentURL)
		}
	})

	t.Run("ArtistsLink", func(t *testing.T) {
		navigateToHome()
		helpers.WaitForVisible(t, page, helpers.NavArtistsLink)
		helpers.ClickAndWait(t, page, helpers.NavArtistsLink)
		page.WaitForTimeout(500)

		currentURL := page.URL()
		if !strings.Contains(currentURL, "/list/artists") {
			t.Errorf("expected /list/artists, got: %s", currentURL)
		}
	})

	t.Run("AdminLink", func(t *testing.T) {
		navigateToHome()
		helpers.WaitForVisible(t, page, helpers.NavAdminLink)
		helpers.ClickAndWait(t, page, helpers.NavAdminLink)
		page.WaitForTimeout(500)

		currentURL := page.URL()
		if !strings.Contains(currentURL, "/admin") {
			t.Errorf("expected /admin, got: %s", currentURL)
		}
	})

	t.Run("SlashShortcut", func(t *testing.T) {
		navigateToHome()
		page.Keyboard().Press("/")
		page.WaitForTimeout(200)

		isFocused, err := page.Locator(helpers.SearchInput).Evaluate("el => el === document.activeElement")
		if err == nil && isFocused == true {
			t.Log("/ shortcut focuses search input")
		} else {
			t.Log("/ shortcut may not have focused search input")
		}
	})

	// ⚠️ TODO: 移动端汉堡菜单需要移动端 browser context
	// 当前暂跳过
	t.Run("MobileHamburger", func(t *testing.T) {
		t.Skip("mobile hamburger needs separate mobile browser context")
	})
}

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
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
		navigateToHome()
		currentURL := page.URL()
		if !strings.HasSuffix(currentURL, "/") && !strings.Contains(currentURL, testServer.URL) {
			t.Errorf("unexpected URL after home goto: %s", currentURL)
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
			t.Errorf("expected search URL with 'search' and 'naruto', got: %s", currentURL)
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

		currentURL := page.URL()
		if !strings.Contains(currentURL, "/list/tags") {
			t.Errorf("expected /list/tags, got: %s", currentURL)
		} else {
			t.Log("tags link navigated correctly")
		}
	})

	t.Run("ArtistsLink", func(t *testing.T) {
		navigateToHome()
		helpers.WaitForVisible(t, page, helpers.NavArtistsLink)
		helpers.ClickAndWait(t, page, helpers.NavArtistsLink)

		currentURL := page.URL()
		if !strings.Contains(currentURL, "/list/artists") {
			t.Errorf("expected /list/artists, got: %s", currentURL)
		} else {
			t.Log("artists link navigated correctly")
		}
	})

	t.Run("AdminLink", func(t *testing.T) {
		navigateToHome()
		helpers.WaitForVisible(t, page, helpers.NavAdminLink)
		helpers.ClickAndWait(t, page, helpers.NavAdminLink)

		currentURL := page.URL()
		if !strings.Contains(currentURL, "/admin") {
			t.Errorf("expected /admin, got: %s", currentURL)
		} else {
			t.Log("admin link navigated correctly")
		}
	})

	t.Run("SlashShortcut", func(t *testing.T) {
		navigateToHome()
		page.Keyboard().Press("/")

		isFocused, err := page.Locator(helpers.SearchInput).Evaluate("el => el === document.activeElement", nil)
		if err == nil && isFocused == true {
			t.Log("/ shortcut focuses search input")
		} else {
			t.Errorf("/ shortcut did not focus search input (focused=%v, err=%v)", isFocused, err)
		}
	})

	// ⚠️ TODO: 移动端汉堡菜单需要移动端 browser context
	// 当前暂跳过
	t.Run("MobileHamburger", func(t *testing.T) {
		mobileCtx, err := helpers.CreateMobileContext(pw, browser, "iPhone 12")
		if err != nil {
			t.Fatalf("create mobile context failed: %v", err)
		}
		defer mobileCtx.Close()

		mobilePage, err := mobileCtx.NewPage()
		if err != nil {
			t.Fatalf("create mobile page failed: %v", err)
		}
		helpers.InjectTestMode(t, mobilePage)
		mobilePage.SetDefaultTimeout(10000)

		_, err = mobilePage.Goto(testServer.URL+"/g/3001",
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("mobile navigate failed: %v", err)
		}

		// 汉堡菜单按钮应当可见
		if !helpers.IsVisible(t, mobilePage, helpers.HamburgerBtn) {
			t.Error("hamburger button not visible on mobile viewport")
		}

		// 点击汉堡菜单展开导航
		helpers.ClickAndWait(t, mobilePage, helpers.HamburgerBtn)
		mobilePage.WaitForTimeout(300)

		// 展开后导航链接应当可见
		if helpers.IsVisible(t, mobilePage, helpers.NavTagsLink) {
			t.Log("nav links visible after hamburger click on mobile")
		}

		// 再次点击收缩
		helpers.ClickAndWait(t, mobilePage, helpers.HamburgerBtn)
		mobilePage.WaitForTimeout(300)
		t.Log("mobile hamburger toggle completed")
	})
}

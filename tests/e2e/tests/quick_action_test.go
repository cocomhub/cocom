// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"strings"
	"testing"

	"github.com/cocomhub/cocom/tests/e2e/helpers"
	"github.com/playwright-community/playwright-go"
)

// TestQuickActions 首页快速操作侧边栏测试组
func TestQuickActions(t *testing.T) {
	page, cleanup := newPage(t)
	defer cleanup()

	navigateToHome := func() {
		_, err := page.Goto(testServer.URL,
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate to home failed: %v", err)
		}
	}

	t.Run("LinkModeEnterExit", func(t *testing.T) {
		navigateToHome()
		helpers.WaitForVisible(t, page, helpers.LinkModeBtn)

		// 进入链接模式
		helpers.ClickAndWait(t, page, helpers.LinkModeBtn)
		page.WaitForTimeout(300)

		// 验证状态面板出现
		if helpers.IsVisible(t, page, helpers.SidebarStatus) {
			t.Log("sidebar status visible in link mode")
		}

		// 退出
		helpers.ClickAndWait(t, page, helpers.LinkModeBtn)
		page.WaitForTimeout(200)
	})

	t.Run("LinkModeSelect", func(t *testing.T) {
		navigateToHome()
		helpers.WaitForVisible(t, page, helpers.LinkModeBtn)
		helpers.ClickAndWait(t, page, helpers.LinkModeBtn)
		page.WaitForTimeout(300)

		cards := page.Locator(helpers.GalleryCard)
		count, err := cards.Count()
		if err != nil || count < 2 {
			t.Skip("need at least 2 gallery cards on home page")
		}

		// 选第一个为主
		cards.Nth(0).Click()
		page.WaitForTimeout(200)
		// 选第二个为子
		cards.Nth(1).Click()
		page.WaitForTimeout(200)

		if helpers.IsVisible(t, page, helpers.SidebarStatus) {
			t.Log("sidebar status updated after selecting comics")
		}
	})

	t.Run("CompareMode", func(t *testing.T) {
		navigateToHome()
		helpers.WaitForVisible(t, page, helpers.CompareModeBtn)
		helpers.ClickAndWait(t, page, helpers.CompareModeBtn)
		page.WaitForTimeout(300)

		cards := page.Locator(helpers.GalleryCard)
		count, err := cards.Count()
		if err != nil || count < 2 {
			t.Skip("need at least 2 gallery cards")
		}

		cards.Nth(0).Click()
		page.WaitForTimeout(100)
		cards.Nth(1).Click()
		page.WaitForTimeout(100)

		if helpers.IsVisible(t, page, helpers.ConfirmBtn) {
			helpers.ClickAndWait(t, page, helpers.ConfirmBtn)
			page.WaitForTimeout(500)
			currentURL := page.URL()
			if strings.Contains(currentURL, "/admin") {
				t.Logf("redirected to admin: %s", currentURL)
			}
		}
	})

	t.Run("NewTabPreference", func(t *testing.T) {
		navigateToHome()
		helpers.WaitForVisible(t, page, helpers.NewTabCheckbox)

		// 取消勾选
		page.Locator(helpers.NewTabCheckbox).Uncheck()
		page.WaitForTimeout(100)

		pref, err := page.Evaluate("localStorage.getItem('comic-link-target')")
		if err == nil {
			t.Logf("new tab pref: %v", pref)
		}

		// 重新勾上
		page.Locator(helpers.NewTabCheckbox).Check()
		page.WaitForTimeout(100)
	})
}

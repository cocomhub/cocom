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

		// 验证状态面板出现
		if !helpers.IsVisible(t, page, helpers.SidebarStatus) {
			t.Error("sidebar status not visible after entering link mode")
		} else {
			t.Log("link mode entered, sidebar status appeared")
		}

		// 退出
		helpers.ClickAndWait(t, page, helpers.LinkModeBtn)
	})

	t.Run("LinkModeSelect", func(t *testing.T) {
		navigateToHome()
		helpers.WaitForVisible(t, page, helpers.LinkModeBtn)
		helpers.ClickAndWait(t, page, helpers.LinkModeBtn)

		cards := page.Locator(helpers.GalleryCard)
		count, err := cards.Count()
		if err != nil || count < 2 {
			t.Skip("need at least 2 gallery cards on home page")
		}

		// 选第一个为主，第二个为子
		cards.Nth(0).Click()
		cards.Nth(1).Click()

		if !helpers.IsVisible(t, page, helpers.SidebarStatus) {
			t.Error("sidebar status disappeared after selecting comics in link mode")
		} else {
			t.Log("sidebar status updated after selecting comics")
		}
	})

	t.Run("CompareMode", func(t *testing.T) {
		navigateToHome()
		helpers.WaitForVisible(t, page, helpers.CompareModeBtn)
		helpers.ClickAndWait(t, page, helpers.CompareModeBtn)

		cards := page.Locator(helpers.GalleryCard)
		count, err := cards.Count()
		if err != nil || count < 2 {
			t.Skip("need at least 2 gallery cards")
		}

		cards.Nth(0).Click()
		cards.Nth(1).Click()

		if helpers.IsVisible(t, page, helpers.ConfirmBtn) {
			helpers.ClickAndWait(t, page, helpers.ConfirmBtn)
			currentURL := page.URL()
			if strings.Contains(currentURL, "/admin") {
				t.Log("redirected to admin compare page")
			} else {
				t.Errorf("expected redirect to /admin, got: %s", currentURL)
			}
		} else {
			t.Error("confirm button not visible after selecting 2 comics")
		}
	})

	t.Run("NewTabPreference", func(t *testing.T) {
		navigateToHome()
		helpers.WaitForVisible(t, page, helpers.NewTabCheckbox)

		// 取消勾选
		page.Locator(helpers.NewTabCheckbox).Uncheck()
		pref, err := page.Evaluate("localStorage.getItem('comic-link-target')", nil)
		if err != nil {
			t.Errorf("failed to read localStorage pref: %v", err)
		} else if pref != "_blank" && pref != nil && pref != "" {
			t.Errorf("expected new tab pref to be _blank, empty, or nil after uncheck, got: %v", pref)
		}

		// 重新勾上
		page.Locator(helpers.NewTabCheckbox).Check()
	})
}

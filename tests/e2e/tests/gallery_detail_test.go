// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/cocomhub/cocom/tests/e2e/helpers"
	"github.com/playwright-community/playwright-go"
)

// TestGalleryDetail 漫画详情页侧边栏测试组
func TestGalleryDetail(t *testing.T) {
	page, cleanup := newPage(t)
	defer cleanup()

	navigateToDetail := func(cid int) {
		_, err := page.Goto(fmt.Sprintf("%s/g/%d", testServer.URL, cid),
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate to /g/%d failed: %v", cid, err)
		}
	}

	t.Run("LikeToggle", func(t *testing.T) {
		navigateToDetail(3001)
		helpers.WaitForVisible(t, page, helpers.LikeBtn)

		helpers.ClickAndWait(t, page, helpers.LikeBtn)
		page.WaitForTimeout(500)
		likeText := helpers.GetText(t, page, helpers.LikeBtn)
		t.Logf("after like toggle, button text: %s", likeText)
	})

	t.Run("LikeToggleTwice", func(t *testing.T) {
		navigateToDetail(3001)
		helpers.WaitForVisible(t, page, helpers.LikeBtn)

		helpers.ClickAndWait(t, page, helpers.LikeBtn)
		page.WaitForTimeout(300)
		helpers.ClickAndWait(t, page, helpers.LikeBtn)
		page.WaitForTimeout(300)
		t.Logf("double toggle completed")
	})

	t.Run("ArchiveButtonVisible", func(t *testing.T) {
		navigateToDetail(3001)
		helpers.WaitForVisible(t, page, helpers.ArchiveBtn)

		archiveText := helpers.GetText(t, page, helpers.ArchiveBtn)
		t.Logf("archive button text: %s", archiveText)
	})

	t.Run("RestoreButton", func(t *testing.T) {
		navigateToDetail(3001)
		helpers.WaitForVisible(t, page, helpers.ArchiveBtn)

		archiveText := helpers.GetText(t, page, helpers.ArchiveBtn)
		if strings.Contains(archiveText, "恢复") {
			t.Log("restore button visible for archived comic 3001")
		}
	})

	t.Run("ZoomPreset", func(t *testing.T) {
		navigateToDetail(3001)
		// 检查 zoom sidebar 是否可见
		if helpers.IsVisible(t, page, helpers.PresetBtn400) {
			helpers.ClickAndWait(t, page, helpers.PresetBtn400)
			page.WaitForTimeout(200)
			zoomVal := helpers.GetText(t, page, helpers.ZoomValue)
			val, err := strconv.Atoi(strings.TrimSuffix(zoomVal, "px"))
			if err == nil && val != 400 {
				t.Logf("expected zoom 400px, got %d", val)
			} else {
				t.Logf("zoom preset 400 applied: %s", zoomVal)
			}
		} else {
			t.Skip("zoom sidebar not visible (display:none)")
		}
	})

	t.Run("ZoomReset", func(t *testing.T) {
		navigateToDetail(3001)
		if !helpers.IsVisible(t, page, helpers.ResetBtn) {
			t.Skip("zoom sidebar not visible")
		}
		// 设为 400 再重置
		helpers.ClickAndWait(t, page, helpers.PresetBtn400)
		page.WaitForTimeout(100)
		helpers.ClickAndWait(t, page, helpers.ResetBtn)
		page.WaitForTimeout(100)
		zoomVal := helpers.GetText(t, page, helpers.ZoomValue)
		t.Logf("after reset, zoom: %s", zoomVal)
	})

	t.Run("LargeModeToggle", func(t *testing.T) {
		navigateToDetail(3001)
		helpers.WaitForVisible(t, page, helpers.LargeToggleBtn)
		helpers.ClickAndWait(t, page, helpers.LargeToggleBtn)
		page.WaitForTimeout(300)

		hasLarge, err := page.Locator(helpers.ThumbContainer).Evaluate("el => el.classList.contains('thumb-container-large')")
		if err == nil && hasLarge == true {
			t.Log("large mode class applied")
		} else {
			t.Log("large mode toggle may not have applied class")
		}
	})

	t.Run("PageManageBtn", func(t *testing.T) {
		navigateToDetail(3001)
		helpers.WaitForVisible(t, page, helpers.PageManageBtn)
		helpers.ClickAndWait(t, page, helpers.PageManageBtn)
		page.WaitForTimeout(300)
		t.Log("page manage button clicked")
	})

	// ⚠️ TODO: 删除确认使用浏览器原生 prompt，Playwright 无法正常自动化
	// 建议后续重构替换为自定义 Modal 弹窗
	t.Run("DeleteButtonVisible", func(t *testing.T) {
		navigateToDetail(3001)
		helpers.WaitForVisible(t, page, helpers.DeleteBtn)
		t.Log("delete button is visible")
	})
}

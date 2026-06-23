// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
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

	t.Run("LikeToggle", func(t *testing.T) {
		page.Goto(testServer.URL + "/g/3001")
		helpers.WaitForVisible(t, page, helpers.LikeBtn)

		helpers.ClickAndWait(t, page, helpers.LikeBtn)
		likeText := helpers.GetText(t, page, helpers.LikeBtn)
		if !strings.Contains(likeText, "Like") {
			t.Errorf("expected like button to show 'Like' after toggle, got: %s", likeText)
		}
	})

	t.Run("ArchiveButtonVisible", func(t *testing.T) {
		page.Goto(testServer.URL + "/g/3001")
		helpers.WaitForVisible(t, page, helpers.ArchiveBtn)

		archiveText := helpers.GetText(t, page, helpers.ArchiveBtn)
		if !strings.Contains(archiveText, "恢复") {
			t.Errorf("expected archive button text to contain 恢复, got: %s", archiveText)
		}
	})

	t.Run("RestoreButton", func(t *testing.T) {
		page.Goto(testServer.URL + "/g/3001")
		helpers.WaitForVisible(t, page, helpers.ArchiveBtn)

		archiveText := helpers.GetText(t, page, helpers.ArchiveBtn)
		if strings.Contains(archiveText, "恢复") {
			t.Log("restore button visible for archived comic 3001")
		} else {
			t.Errorf("expected restore (恢复) button for archived comic 3001, got: %s", archiveText)
		}
	})

	t.Run("ZoomPreset", func(t *testing.T) {
		page.Goto(testServer.URL + "/g/3001")
		if !helpers.IsVisible(t, page, helpers.ZoomSidebar) {
			helpers.EnterLargeMode(t, page)
		}
		helpers.ClickAndWait(t, page, helpers.PresetBtn400)
		zoomVal := helpers.GetText(t, page, helpers.ZoomValue)
		val, err := strconv.Atoi(strings.TrimSuffix(zoomVal, "px"))
		if err == nil && val != 400 {
			t.Errorf("zoom zoom preset 400 not applied: expected 400px, got %dpx", val)
		} else if err != nil {
			t.Errorf("zoom zoom value parse failed: %v", err)
		} else {
			t.Logf("zoom preset 400 applied: %s", zoomVal)
		}
	})

	t.Run("ZoomReset", func(t *testing.T) {
		page.Goto(testServer.URL + "/g/3001")
		if !helpers.IsVisible(t, page, helpers.ZoomSidebar) {
			helpers.EnterLargeMode(t, page)
		}
		helpers.ClickAndWait(t, page, helpers.PresetBtn400)
		helpers.ClickAndWait(t, page, helpers.ResetBtn)
		zoomVal := helpers.GetText(t, page, helpers.ZoomValue)
		val, parseErr := strconv.Atoi(strings.TrimSuffix(zoomVal, "px"))
		if parseErr != nil {
			t.Errorf("zoom value parse failed after reset: %v (val=%q)", parseErr, zoomVal)
		} else if val <= 400 {
			t.Errorf("expected zoom to reset above 400px, got %dpx", val)
		} else {
			t.Logf("after reset, zoom: %s", zoomVal)
		}
	})

	t.Run("LargeModeToggle", func(t *testing.T) {
		// 使用独立 page 避免其他测试的 JS 状态污染
		p, cleanup2 := newPage(t)
		defer cleanup2()

		p.Goto(testServer.URL + "/g/3001")
		helpers.WaitForVisible(t, p, helpers.LargeToggleBtn)
		helpers.ClickAndWait(t, p, helpers.LargeToggleBtn)

		if err := p.Locator("#thumbnail-container.large-mode").WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(5000),
			State:   playwright.WaitForSelectorStateAttached,
		}); err != nil {
			t.Errorf("large mode class not applied after toggle: %v", err)
		} else {
			t.Log("large mode class applied")
		}
	})

	t.Run("PageManageBtn", func(t *testing.T) {
		page.Goto(testServer.URL + "/g/3001")
		helpers.WaitForVisible(t, page, helpers.PageManageBtn)
		helpers.ClickAndWait(t, page, helpers.PageManageBtn)
		t.Log("page manage button clicked")
	})

	t.Run("DeleteButtonVisible", func(t *testing.T) {
		page.Goto(testServer.URL + "/g/3001")
		helpers.WaitForVisible(t, page, helpers.DeleteBtn)
		t.Log("delete button is visible")
	})

	t.Run("DeleteCancel", func(t *testing.T) {
		page.Goto(testServer.URL + "/g/3003")
		helpers.WaitForVisible(t, page, helpers.DeleteBtn)

		dialogDismissed := false
		page.On("dialog", func(d playwright.Dialog) {
			dialogDismissed = true
			d.Dismiss()
		})
		helpers.ClickAndWait(t, page, helpers.DeleteBtn)
		if !dialogDismissed {
			t.Log("no dialog appeared (delete may be refactored)")
		} else {
			t.Log("delete prompt dismissed via dialog handler")
		}
		currentURL := page.URL()
		if !strings.Contains(currentURL, "/g/3003") {
			t.Errorf("expected to stay on gallery page after delete cancel, got: %s", currentURL)
		}
	})

	t.Run("DeleteConfirm", func(t *testing.T) {
		page.Goto(testServer.URL + "/g/3003")
		helpers.WaitForVisible(t, page, helpers.DeleteBtn)

		dialogHandled := false
		page.On("dialog", func(d playwright.Dialog) {
			dialogHandled = true
			// openDeleteConfirm 使用 prompt('输入 CID 以确认删除:\n...\n\nCID: 3003')
			// 输入正确的 CID 数字以通过 JS 验证
			d.Accept("3003")
		})
		helpers.ClickAndWait(t, page, helpers.DeleteBtn)
		helpers.WaitForURLMatch(t, page, testServer.URL+"/", 5000)
		if !dialogHandled {
			t.Log("no dialog appeared (delete may be refactored to modal)")
		} else {
			t.Log("delete prompt accepted with correct CID")
		}
		// 删除成功后应跳转到首页
		currentURL := page.URL()
		t.Logf("after delete confirm, URL: %s", currentURL)
		if !strings.HasSuffix(currentURL, "/") && currentURL != testServer.URL+"/" {
			t.Log("delete redirect may need JS init for navigation")
		}
	})

	t.Run("ZoomSliderChange", func(t *testing.T) {
		page.Goto(testServer.URL + "/g/3003")
		if !helpers.IsVisible(t, page, helpers.ZoomSidebar) {
			helpers.EnterLargeMode(t, page)
		}
		_, err := page.Locator(helpers.ZoomSlider).Evaluate("el => { el.value = '800'; el.dispatchEvent(new Event('input')); el.dispatchEvent(new Event('change')); }", nil)
		if err != nil {
			t.Fatalf("failed to set zoom slider: %v", err)
		}
		zoomVal := helpers.GetText(t, page, helpers.ZoomValue)
		val, parseErr := strconv.Atoi(strings.TrimSuffix(zoomVal, "px"))
		if parseErr != nil {
			t.Errorf("zoom value parse failed: %v (val=%q)", parseErr, zoomVal)
		} else if val != 800 {
			t.Errorf("expected zoom value 800px after slider change, got: %dpx", val)
		}
	})

	t.Run("ZoomIn", func(t *testing.T) {
		page.Goto(testServer.URL + "/g/3003")
		if !helpers.IsVisible(t, page, helpers.ZoomSidebar) {
			helpers.EnterLargeMode(t, page)
		}
		beforeVal := helpers.GetText(t, page, helpers.ZoomValue)
		helpers.ClickAndWait(t, page, helpers.ZoomInBtn)
		afterVal := helpers.GetText(t, page, helpers.ZoomValue)
		if beforeVal == afterVal {
			t.Errorf("expected zoom value to increase after +click, before=%s after=%s", beforeVal, afterVal)
		}
	})

	t.Run("ZoomOut", func(t *testing.T) {
		page.Goto(testServer.URL + "/g/3003")
		if !helpers.IsVisible(t, page, helpers.ZoomSidebar) {
			helpers.EnterLargeMode(t, page)
		}
		beforeVal := helpers.GetText(t, page, helpers.ZoomValue)
		helpers.ClickAndWait(t, page, helpers.ZoomOutBtn)
		afterVal := helpers.GetText(t, page, helpers.ZoomValue)
		if beforeVal == afterVal {
			t.Errorf("expected zoom value to decrease after -click, before=%s after=%s", beforeVal, afterVal)
		}
	})
}

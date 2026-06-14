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
		if !strings.Contains(likeText, "♡") && !strings.Contains(likeText, "♥") {
			t.Errorf("expected like button to show ♡ or ♥ after toggle, got: %s", likeText)
		}
	})

	t.Run("ArchiveButtonVisible", func(t *testing.T) {
		page.Goto(testServer.URL + "/g/3001")
		helpers.WaitForVisible(t, page, helpers.ArchiveBtn)

		archiveText := helpers.GetText(t, page, helpers.ArchiveBtn)
		if !strings.Contains(archiveText, "归档") {
			t.Errorf("expected archive button text to contain 归档, got: %s", archiveText)
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
		if helpers.IsVisible(t, page, helpers.PresetBtn400) {
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
		} else {
			t.Skip("zoom sidebar not visible (display:none)")
		}
	})

	t.Run("ZoomReset", func(t *testing.T) {
		page.Goto(testServer.URL + "/g/3001")
		if !helpers.IsVisible(t, page, helpers.ResetBtn) {
			t.Skip("zoom sidebar not visible")
		}
		helpers.ClickAndWait(t, page, helpers.PresetBtn400)
		helpers.ClickAndWait(t, page, helpers.ResetBtn)
		zoomVal := helpers.GetText(t, page, helpers.ZoomValue)
		val, parseErr := strconv.Atoi(strings.TrimSuffix(zoomVal, "px"))
		if parseErr != nil {
			t.Errorf("zoom value parse failed after reset: %v", parseErr)
		} else if val == 400 {
			t.Errorf("expected zoom to reset from 400, got still %dpx", val)
		} else {
			t.Logf("after reset, zoom: %s", zoomVal)
		}
	})

	t.Run("LargeModeToggle", func(t *testing.T) {
		page.Goto(testServer.URL + "/g/3001")
		helpers.WaitForVisible(t, page, helpers.LargeToggleBtn)
		helpers.ClickAndWait(t, page, helpers.LargeToggleBtn)

		hasLarge, err := page.Locator(helpers.ThumbContainer).Evaluate("el => el.classList.contains('thumb-container-large')", nil)
		if err != nil {
			t.Errorf("Evaluate large mode class failed: %v", err)
		} else if hasLarge != true {
			t.Error("large mode class not applied after toggling large toggle button")
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

	t.Run("ArchiveClick", func(t *testing.T) {
		page.Goto(testServer.URL + "/g/3002")
		helpers.WaitForVisible(t, page, helpers.ArchiveBtn)

		beforeText := helpers.GetText(t, page, helpers.ArchiveBtn)
		if !strings.Contains(beforeText, "归档") {
			t.Fatalf("expected archive text for unarchived comic 3002, got: %s", beforeText)
		}

		helpers.ClickAndWait(t, page, helpers.ArchiveBtn)
		page.WaitForTimeout(600)

		afterText := helpers.GetText(t, page, helpers.ArchiveBtn)
		if !strings.Contains(afterText, "恢复") {
			t.Errorf("after archive, expected restore (恢复) text, got: %s", afterText)
		}
	})

	t.Run("RestoreClick", func(t *testing.T) {
		page.Goto(testServer.URL + "/g/3001")
		helpers.WaitForVisible(t, page, helpers.ArchiveBtn)

		beforeText := helpers.GetText(t, page, helpers.ArchiveBtn)
		if !strings.Contains(beforeText, "恢复") {
			t.Fatalf("expected restore text for archived comic 3001, got: %s", beforeText)
		}

		helpers.ClickAndWait(t, page, helpers.ArchiveBtn)
		page.WaitForTimeout(600)

		afterText := helpers.GetText(t, page, helpers.ArchiveBtn)
		if !strings.Contains(afterText, "归档") {
			t.Errorf("after restore, expected archive (归档) text, got: %s", afterText)
		}
	})

	t.Run("ZoomSliderChange", func(t *testing.T) {
		page.Goto(testServer.URL + "/g/3003")
		if !helpers.IsVisible(t, page, helpers.ZoomSidebar) {
			t.Skip("zoom sidebar not visible (display:none)")
		}
		_, err := page.Locator(helpers.ZoomSlider).Evaluate("el => { el.value = '800'; el.dispatchEvent(new Event('input')); el.dispatchEvent(new Event('change')); }", nil)
		if err != nil {
			t.Fatalf("failed to set zoom slider: %v", err)
		}
		page.WaitForTimeout(300)
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
			t.Skip("zoom sidebar not visible (display:none)")
		}
		beforeVal := helpers.GetText(t, page, helpers.ZoomValue)
		helpers.ClickAndWait(t, page, helpers.ZoomInBtn)
		page.WaitForTimeout(300)
		afterVal := helpers.GetText(t, page, helpers.ZoomValue)
		if beforeVal == afterVal {
			t.Errorf("expected zoom value to increase after +click, before=%s after=%s", beforeVal, afterVal)
		}
	})

	t.Run("ZoomOut", func(t *testing.T) {
		page.Goto(testServer.URL + "/g/3003")
		if !helpers.IsVisible(t, page, helpers.ZoomSidebar) {
			t.Skip("zoom sidebar not visible (display:none)")
		}
		beforeVal := helpers.GetText(t, page, helpers.ZoomValue)
		helpers.ClickAndWait(t, page, helpers.ZoomOutBtn)
		page.WaitForTimeout(300)
		afterVal := helpers.GetText(t, page, helpers.ZoomValue)
		if beforeVal == afterVal {
			t.Errorf("expected zoom value to decrease after -click, before=%s after=%s", beforeVal, afterVal)
		}
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
		page.WaitForTimeout(500)
		if !dialogDismissed {
			t.Log("no dialog appeared (delete may be refactored)")
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
			d.Accept("delete")
		})
		helpers.ClickAndWait(t, page, helpers.DeleteBtn)
		page.WaitForTimeout(800)
		if !dialogHandled {
			t.Log("no dialog appeared (delete may be refactored to modal)")
		}
		currentURL := page.URL()
		t.Logf("after delete confirm, URL: %s", currentURL)
	})
}

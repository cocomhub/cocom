// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"strconv"
	"strings"
	"testing"

	"github.com/cocomhub/cocom/tests/e2e/helpers"
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
		t.Logf("after like toggle, button text: %s", likeText)
	})

	t.Run("ArchiveButtonVisible", func(t *testing.T) {
		page.Goto(testServer.URL + "/g/3001")
		helpers.WaitForVisible(t, page, helpers.ArchiveBtn)

		archiveText := helpers.GetText(t, page, helpers.ArchiveBtn)
		t.Logf("archive button text: %s", archiveText)
	})

	t.Run("RestoreButton", func(t *testing.T) {
		page.Goto(testServer.URL + "/g/3001")
		helpers.WaitForVisible(t, page, helpers.ArchiveBtn)

		archiveText := helpers.GetText(t, page, helpers.ArchiveBtn)
		if strings.Contains(archiveText, "恢复") {
			t.Log("restore button visible for archived comic 3001")
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
		t.Logf("after reset, zoom: %s", zoomVal)
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
}

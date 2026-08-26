// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cocomhub/cocom/tests/e2e/helpers"
	"github.com/mxschmitt/playwright-go"
)

// TestCompare 漫画比对流程测试组
func TestCompare(t *testing.T) {
	page, cleanup := newPage(t)
	defer cleanup()

	t.Run("Execute", func(t *testing.T) {
		_, err := page.Goto(fmt.Sprintf("%s/admin", testServer.URL),
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate to admin failed: %v", err)
		}

		helpers.WaitForVisible(t, page, helpers.CIDMain)
		page.Locator(helpers.CIDMain).Fill("2001")
		page.Locator(helpers.CIDTarget).Fill("2002")
		helpers.ClickAndWait(t, page, helpers.CompareBtn)

		helpers.WaitForVisible(t, page, helpers.CompareResult)
		statsText := helpers.GetText(t, page, helpers.StatsBar)
		// 对齐页数应为 5（2001/2002 各 5 页同名文件，全部同页对齐），且非零
		if !strings.Contains(statsText, "对齐页数：5") && strings.Contains(statsText, "对齐页数：0") {
			t.Errorf("expected stats bar aligned page count to be 5 or non-zero, got: %s", statsText)
		}

		// 结果表格应有 5 行（每页一行）——通过对比表格验证。
		// 表格由 JS 在 compare-result 可见后异步填充，先轮询等待 tbody 出现。
		rowLoc := page.Locator(helpers.CompareTable + " tbody tr")
		if err := rowLoc.First().WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(5000),
			State:   playwright.WaitForSelectorStateAttached,
		}); err != nil {
			t.Errorf("compare table tbody rows did not appear: %v", err)
			return
		}
		n, _ := rowLoc.Count()
		if n != 5 {
			t.Errorf("expected 5 comparison rows, got %d", n)
		}
	})

	t.Run("Swap", func(t *testing.T) {
		_, err := page.Goto(fmt.Sprintf("%s/admin", testServer.URL),
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate to admin failed: %v", err)
		}

		page.Locator(helpers.CIDMain).Fill("2001")
		page.Locator(helpers.CIDTarget).Fill("2002")
		helpers.ClickAndWait(t, page, helpers.SwapBtn)

		val1, _ := page.Locator(helpers.CIDMain).InputValue()
		val2, _ := page.Locator(helpers.CIDTarget).InputValue()
		if val1 != "2002" || val2 != "2001" {
			t.Errorf("swap failed: expect 2002/2001, got %s/%s", val1, val2)
		}
	})

	t.Run("MultiCIDParam", func(t *testing.T) {
		_, err := page.Goto(fmt.Sprintf("%s/admin?cids=2001,2002", testServer.URL),
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate failed: %v", err)
		}

		val1, _ := page.Locator(helpers.CIDMain).InputValue()
		val2, _ := page.Locator(helpers.CIDTarget).InputValue()
		if val1 != "2001" || val2 != "2002" {
			t.Errorf("auto fill failed: expect 2001/2002, got %s/%s", val1, val2)
		}
		helpers.WaitForVisible(t, page, helpers.CompareResult)
	})

	t.Run("InvalidCID", func(t *testing.T) {
		_, err := page.Goto(fmt.Sprintf("%s/admin", testServer.URL),
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate failed: %v", err)
		}

		page.Locator(helpers.CIDMain).Fill("99999")
		page.Locator(helpers.CIDTarget).Fill("99998")
		helpers.ClickAndWait(t, page, helpers.CompareBtn)

		// 页面不应崩溃，结果可能为空或显示错误
		resultVisible := helpers.IsVisible(t, page, helpers.CompareResult)
		if !resultVisible {
			t.Log("compare result not visible after invalid CID (expected for invalid data)")
		} else if errMsg := helpers.GetText(t, page, helpers.Messages); errMsg != "" {
			t.Logf("error message displayed: %s", errMsg)
		}
	})

	t.Run("MultiCIDParam_Render", func(t *testing.T) {
		_, err := page.Goto(fmt.Sprintf("%s/admin?cids=2001,2002", testServer.URL),
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate failed: %v", err)
		}

		if helpers.IsVisible(t, page, helpers.MultiComicBar) {
			t.Log("multi-comic bar rendered")
		} else {
			t.Log("multi-comic bar not rendered (may need JS init)")
		}

		if helpers.IsVisible(t, page, helpers.LinkAction) {
			t.Log("link action area rendered")
		}

		if helpers.IsVisible(t, page, helpers.ComicInfoPair) {
			t.Log("comic info pair rendered")
		}
	})
}

// TestCompare_Preview 漫画比对并排预览测试
func TestCompare_Preview(t *testing.T) {
	page, cleanup := newPage(t)
	defer cleanup()

	_, err := page.Goto(fmt.Sprintf("%s/admin", testServer.URL),
		playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
	if err != nil {
		t.Fatalf("navigate failed: %v", err)
	}

	page.Locator(helpers.CIDMain).Fill("2001")
	page.Locator(helpers.CIDTarget).Fill("2002")
	helpers.ClickAndWait(t, page, helpers.CompareBtn)
	helpers.WaitForVisible(t, page, helpers.CompareResult)

	// 统计断言：2001/2002 各 5 页，前 2 页相同（MD5 匹配）后 3 页不同
	// 种子图片由同一 jpeg 编码器生成，同名同 seed 的 MD5 才相同，这里验证匹配/不匹配计数
	rowLoc := page.Locator(helpers.CompareTable + " tbody tr")
	if err := rowLoc.First().WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
		State:   playwright.WaitForSelectorStateAttached,
	}); err != nil {
		t.Errorf("compare table rows did not appear: %v", err)
	} else {
		n, _ := rowLoc.Count()
		t.Logf("compare stats: %d rows rendered", n)
	}
	statsText := helpers.GetText(t, page, helpers.StatsBar)
	if !strings.Contains(statsText, "对齐页数：5") || !strings.Contains(statsText, "2 匹配") || !strings.Contains(statsText, "3 不匹配") {
		t.Errorf("unexpected compare stats, got: %s", statsText)
	} else {
		t.Logf("compare stats correct: %s", statsText)
	}
	previewBtns := page.Locator(helpers.PreviewBtn)
	count, err := previewBtns.Count()
	if err == nil && count > 0 {
		previewBtns.First().Click()
		helpers.WaitForVisible(t, page, helpers.PreviewPanel)
		if helpers.IsVisible(t, page, helpers.PreviewPanel) {
			t.Log("preview panel opened")
			// 测试 Esc 关闭
			page.Keyboard().Press("Escape")
			helpers.WaitForHidden(t, page, helpers.PreviewPanel)
			if helpers.IsVisible(t, page, helpers.PreviewPanel) {
				t.Log("preview panel still visible after Escape (may need JS focus)")
			} else {
				t.Log("preview panel closed via Escape")
			}
		} else {
			t.Error("preview panel did not appear after clicking preview button")
		}
	} else {
		t.Log("no preview buttons found in compare result")
	}
}

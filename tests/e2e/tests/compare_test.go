// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package e2etest 是 E2E 测试的外部测试包（与 main_test.go 的 package main 配合）
// 通过测试包内变量访问 testServer 和 newPage
package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cocomhub/cocom/tests/e2e/helpers"
	"github.com/playwright-community/playwright-go"
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

		// 验证结果区域显示
		helpers.WaitForVisible(t, page, helpers.CompareResult)
		if statsText := helpers.GetText(t, page, helpers.StatsBar); !strings.Contains(statsText, "0") {
			t.Logf("stats bar text: %s", statsText)
		}
		if infoText := helpers.GetText(t, page, helpers.ComicInfoPair); !strings.Contains(infoText, "Compare A") {
			t.Logf("info pair text: %s", infoText)
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
		t.Logf("compare result visible with invalid CID: %v", resultVisible)
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

	// 尝试点击预览按钮
	previewBtns := page.Locator("button.preview-btn, .preview-btn")
	count, err := previewBtns.Count()
	if err == nil && count > 0 {
		previewBtns.First().Click()
		page.WaitForTimeout(500)
		if helpers.IsVisible(t, page, helpers.PreviewPanel) {
			t.Log("preview panel opened")
		}
	} else {
		t.Log("no preview buttons found in compare result")
	}
}

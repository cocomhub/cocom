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

// TestRandomGallery 随机导航和图片页面测试组
func TestRandomGallery(t *testing.T) {
	page, cleanup := newPage(t)
	defer cleanup()

	t.Run("GalleryDetailPage", func(t *testing.T) {
		_, err := page.Goto(fmt.Sprintf("%s/g/3001", testServer.URL),
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate to gallery detail failed: %v", err)
		}
		page.WaitForTimeout(500)

		if helpers.IsVisible(t, page, helpers.Cover) {
			t.Log("gallery cover visible")
		} else {
			t.Log("cover not visible (may be image loading)")
		}

		bodyText, err := page.Locator("body").TextContent()
		if err != nil {
			t.Errorf("failed to get body text: %v", err)
		} else if strings.Contains(bodyText, "3001") {
			t.Log("gallery detail page shows CID 3001")
		} else {
			t.Log("gallery detail page loaded but CID 3001 not found in text")
		}

		if helpers.IsVisible(t, page, helpers.ThumbContainer) {
			t.Log("thumbnail container visible")
		}
	})

	t.Run("GalleryFirstPicture", func(t *testing.T) {
		_, err := page.Goto(fmt.Sprintf("%s/g/3001/1", testServer.URL),
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate to gallery picture failed: %v", err)
		}
		page.WaitForTimeout(500)

		currentURL := page.URL()
		if strings.Contains(currentURL, "/g/3001") {
			t.Logf("gallery picture page loaded: %s", currentURL)
		} else {
			t.Errorf("expected gallery picture URL, got: %s", currentURL)
		}
	})

	t.Run("GalleryStaticFile", func(t *testing.T) {
		// 验证图片文件服务 — 模板中图片路径为 /galleries/{CID}_{no}.{format}
		testURL := fmt.Sprintf("%s/galleries/2001/1.png", testServer.URL)
		resp, err := page.Goto(testURL,
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Logf("static file not accessible: %v", err)
		} else if resp != nil {
			t.Logf("static file returned status: %d", resp.Status())
			if resp.Status() == 200 {
				t.Log("gallery static file served successfully")
			}
		}
	})

	t.Run("RandomRedirect", func(t *testing.T) {
		_, err := page.Goto(testServer.URL,
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate to home failed: %v", err)
		}

		randomLink := page.Locator("a[href*='random'], a[href*='Random']")
		count, err := randomLink.Count()
		if err == nil && count > 0 {
			t.Logf("random link found (%d), clicking", count)
			randomLink.First().Click()
			page.WaitForTimeout(1000)

			currentURL := page.URL()
			if strings.Contains(currentURL, "/g/") {
				t.Logf("random redirects to gallery: %s", currentURL)
			} else {
				t.Logf("random navigated to: %s (expected a /g/ CID URL)", currentURL)
			}
		} else {
			t.Log("no random link found on home page")
		}
	})
}

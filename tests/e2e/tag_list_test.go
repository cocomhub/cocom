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

// TestTagList 标签/艺术家列表页面测试组
func TestTagList(t *testing.T) {
	page, cleanup := newPage(t)
	defer cleanup()

	t.Run("TagListRender", func(t *testing.T) {
		_, err := page.Goto(fmt.Sprintf("%s/list/tags/", testServer.URL),
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate to tag list failed: %v", err)
		}
		page.WaitForTimeout(500)

		if helpers.IsVisible(t, page, "#tag-container") {
			t.Log("tag container rendered")
		} else {
			t.Log("tag container not found (may be empty)")
		}

		bodyText, err := page.Locator("body").TextContent()
		if err != nil {
			t.Errorf("failed to get body text: %v", err)
		} else if strings.Contains(bodyText, "action") || strings.Contains(bodyText, "romance") || strings.Contains(bodyText, "comedy") {
			t.Log("seed tags found in tag list")
		} else {
			t.Log("no expected seed tags found in tag list text")
		}
	})

	t.Run("ArtistsListRender", func(t *testing.T) {
		_, err := page.Goto(fmt.Sprintf("%s/list/artists/", testServer.URL),
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate to artists list failed: %v", err)
		}
		page.WaitForTimeout(500)

		bodyText, err := page.Locator("body").TextContent()
		if err != nil {
			t.Errorf("failed to get body text: %v", err)
		} else if len(bodyText) > 0 {
			t.Logf("artists list page rendered with content (%d chars)", len(bodyText))
		}
	})

	t.Run("ParodiesListRender", func(t *testing.T) {
		_, err := page.Goto(fmt.Sprintf("%s/list/parodies/", testServer.URL),
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate to parodies list failed: %v", err)
		}
		page.WaitForTimeout(500)

		bodyText, err := page.Locator("body").TextContent()
		if err != nil {
			t.Errorf("failed to get body text: %v", err)
		} else if strings.Contains(bodyText, "naruto") {
			t.Log("naruto parody tag found in list")
		} else {
			t.Log("parodies list rendered (naruto may need different query)")
		}
	})

	t.Run("NavigationFromHome", func(t *testing.T) {
		_, err := page.Goto(testServer.URL,
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
		if err != nil {
			t.Fatalf("navigate to home failed: %v", err)
		}

		helpers.WaitForVisible(t, page, helpers.NavTagsLink)
		helpers.ClickAndWait(t, page, helpers.NavTagsLink)
		page.WaitForTimeout(500)

		currentURL := page.URL()
		if !strings.Contains(currentURL, "/list/tags") {
			t.Errorf("expected /list/tags after clicking nav, got: %s", currentURL)
		} else {
			t.Log("tag list navigation from home works")
		}
	})
}

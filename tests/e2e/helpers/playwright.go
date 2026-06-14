// Copyright 2026 The Cocomhub Authors. All rights reserved.
// Use of this source code is governed by a Apache-2.0 license that can be
// found in the LICENSE file.

package helpers

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/playwright-community/playwright-go"
)

// ScreenshotDir 截图保存目录
const ScreenshotDir = "screenshots"

// EnsurePlaywright 确保 Playwright 实例和工作浏览器已就绪
// 返回 pw（*playwright.Playwright）和 browser（playwright.Browser）
func EnsurePlaywright(tb testing.TB) (*playwright.Playwright, playwright.Browser) {
	tb.Helper()
	pw, err := playwright.Run()
	if err != nil {
		tb.Fatalf("could not start playwright: %v", err)
	}
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		tb.Fatalf("could not launch Chromium: %v", err)
	}
	return pw, browser
}

// TakeScreenshot 捕获页面截图并保存
func TakeScreenshot(tb testing.TB, page playwright.Page, name string) {
	tb.Helper()
	path := filepath.Join(ScreenshotDir, fmt.Sprintf("%s_%s.png", tb.Name(), name))
	if _, err := page.Screenshot(playwright.PageScreenshotOptions{
		Path: playwright.String(path),
	}); err != nil {
		tb.Logf("screenshot failed: %v", err)
	}
}

// WaitForVisible 等待元素在页面上可见
func WaitForVisible(tb testing.TB, page playwright.Page, selector string) {
	tb.Helper()
	locator := page.Locator(selector)
	if err := locator.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
		State:   playwright.WaitForSelectorStateVisible,
	}); err != nil {
		tb.Fatalf("element %s not visible: %v", selector, err)
	}
}

// ClickAndWait 点击元素并等待可能触发的导航完成
func ClickAndWait(tb testing.TB, page playwright.Page, selector string) {
	tb.Helper()
	locator := page.Locator(selector)
	if err := locator.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
		State:   playwright.WaitForSelectorStateVisible,
	}); err != nil {
		tb.Fatalf("element %s not found: %v", selector, err)
	}
	if err := locator.Click(); err != nil {
		tb.Fatalf("click %s failed: %v", selector, err)
	}
}

// GetText 获取元素文本内容
func GetText(tb testing.TB, page playwright.Page, selector string) string {
	tb.Helper()
	locator := page.Locator(selector)
	if err := locator.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(3000),
		State:   playwright.WaitForSelectorStateVisible,
	}); err != nil {
		return ""
	}
	text, err := locator.TextContent()
	if err != nil {
		return ""
	}
	return text
}

// IsVisible 检查元素是否可见（不报错）
func IsVisible(tb testing.TB, page playwright.Page, selector string) bool {
	tb.Helper()
	locator := page.Locator(selector)
	visible, err := locator.IsVisible()
	if err != nil {
		return false
	}
	return visible
}

// CreateMobileContext 创建移动端设备浏览器上下文
// pw 参数用于从 Playwright 实例中查询设备描述符
func CreateMobileContext(pw *playwright.Playwright, browser playwright.Browser, deviceName string) (playwright.BrowserContext, error) {
	device, ok := pw.Devices[deviceName]
	if !ok {
		return nil, fmt.Errorf("unknown device: %s", deviceName)
	}
	return browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent:         playwright.String(device.UserAgent),
		Viewport:          &playwright.Size{Width: device.Viewport.Width, Height: device.Viewport.Height},
		Screen:            &playwright.Size{Width: device.Screen.Width, Height: device.Screen.Height},
		DeviceScaleFactor: playwright.Float(device.DeviceScaleFactor),
		IsMobile:          playwright.Bool(device.IsMobile),
		HasTouch:          playwright.Bool(device.HasTouch),
	})
}

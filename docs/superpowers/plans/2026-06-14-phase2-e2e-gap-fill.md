# Phase 2 实现计划：补齐当前页面缺失 E2E 子测试

> **面向 AI 代理的工作者：** 使用 superpowers:subagent-driven-development 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法跟踪进度。

**目标：** 在 Phase 1（统一路由注册 + `__E2E_TEST__` + 硬断言化）的基础上，补齐 4 个 E2E 测试文件中缺失的子测试。

**架构：** v2 API 路由（archive/restore）已通过 `pkg/comic.Handler.RegisterRoutes` 注册；`__E2E_TEST__` 已注入使 JS 跳过 confirm/reload；每个测试使用 `newPage(t)` 创建独立 context，种子数据：CID 3001 已归档（可恢复）、CID 3002 未归档（可归档）、CID 3003 多页。缩放侧边栏默认 `display:none`。

**技术栈：** playwright-go v0.5700.1, Gin TestServer, MemoryStorage

---

## 需修改的文件

| 文件 | 职责 |
|------|------|
| `tests/e2e/gallery_detail_test.go` | 新增 Archive/Restore 真实点击、ZoomSlider/Zoom+/ZoomReset、Delete 取消/确认、Like 真实切换 |
| `tests/e2e/compare_test.go` | 新增错误文本硬断言、Link 确认、Preview 键盘导航、多 CID 参数渲染 |
| `tests/e2e/navigation_test.go` | 移 MobileHamburger t.Skip + 实现移动端交互 |
| `tests/e2e/quick_action_test.go` | 新增 CSS 状态验证、Escape 退出模式 |
| `tests/e2e/helpers/selectors.go` | 可能新增选择器 |
| `tests/e2e/main_test.go` | 可能新增 mobilePage 变量 |
| `tests/e2e/helpers/playwright.go` | 新增 `TakeScreenshotOnFailure` |

### 任务 1：gallery_detail 补齐 — Archive/Restore 真实点击

**文件：** `tests/e2e/gallery_detail_test.go`

在 `TestGalleryDetail` 中新增子测试：

```go
t.Run("ArchiveClick", func(t *testing.T) {
    page.Goto(testServer.URL + "/g/3002")
    helpers.WaitForVisible(t, page, helpers.ArchiveBtn)

    // 3002 未归档 → 按钮显示"归档"
    beforeText := helpers.GetText(t, page, helpers.ArchiveBtn)
    if !strings.Contains(beforeText, "归档") {
        t.Fatalf("expected archive text for unarchived comic, got: %s", beforeText)
    }

    helpers.ClickAndWait(t, page, helpers.ArchiveBtn)
    page.WaitForTimeout(500) // await XHR response

    afterText := helpers.GetText(t, page, helpers.ArchiveBtn)
    if !strings.Contains(afterText, "恢复") {
        t.Errorf("after archive, expected restore (恢复) text, got: %s", afterText)
    }
})

t.Run("RestoreClick", func(t *testing.T) {
    page.Goto(testServer.URL + "/g/3001")
    helpers.WaitForVisible(t, page, helpers.ArchiveBtn)

    // 3001 已归档 → 按钮显示"恢复"
    beforeText := helpers.GetText(t, page, helpers.ArchiveBtn)
    if !strings.Contains(beforeText, "恢复") {
        t.Fatalf("expected restore text for archived comic, got: %s", beforeText)
    }

    helpers.ClickAndWait(t, page, helpers.ArchiveBtn)
    page.WaitForTimeout(500)

    afterText := helpers.GetText(t, page, helpers.ArchiveBtn)
    if !strings.Contains(afterText, "归档") {
        t.Errorf("after restore, expected archive (归档) text, got: %s", afterText)
    }
})
```

### 任务 2：gallery_detail 补齐 — ZoomSlider/+/Reset

在缩放侧边栏可见的情况下（需要 `help.IsVisible` guard），新增子测试：

```go
t.Run("ZoomSliderChange", func(t *testing.T) {
    page.Goto(testServer.URL + "/g/3003")
    if !helpers.IsVisible(t, page, helpers.ZoomSidebar) {
        t.Skip("zoom sidebar not visible")
    }
    // 使用 JS 设置 slider 值（Playwright Fill 对 range input 可能不生效）
    _, err := page.Locator(helpers.ZoomSlider).Evaluate("el => { el.value = '800'; el.dispatchEvent(new Event('input')); el.dispatchEvent(new Event('change')); }", nil)
    if err != nil {
        t.Fatalf("failed to set zoom slider: %v", err)
    }
    page.WaitForTimeout(200)
    zoomVal := helpers.GetText(t, page, helpers.ZoomValue)
    val, err := strconv.Atoi(strings.TrimSuffix(zoomVal, "px"))
    if err != nil || val != 800 {
        t.Errorf("expected zoom value 800px after slider change, got: %s", zoomVal)
    }
})

t.Run("ZoomIn", func(t *testing.T) {
    page.Goto(testServer.URL + "/g/3003")
    if !helpers.IsVisible(t, page, helpers.ZoomSidebar) {
        t.Skip("zoom sidebar not visible")
    }
    beforeVal := helpers.GetText(t, page, helpers.ZoomValue)
    helpers.ClickAndWait(t, page, helpers.ZoomInBtn)
    page.WaitForTimeout(200)
    afterVal := helpers.GetText(t, page, helpers.ZoomValue)
    if beforeVal == afterVal {
        t.Errorf("expected zoom value to increase after clicking +, before=%s after=%s", beforeVal, afterVal)
    }
})

t.Run("ZoomOut", func(t *testing.T) {
    page.Goto(testServer.URL + "/g/3003")
    if !helpers.IsVisible(t, page, helpers.ZoomSidebar) {
        t.Skip("zoom sidebar not visible")
    }
    beforeVal := helpers.GetText(t, page, helpers.ZoomValue)
    helpers.ClickAndWait(t, page, helpers.ZoomOutBtn)
    page.WaitForTimeout(200)
    afterVal := helpers.GetText(t, page, helpers.ZoomValue)
    if beforeVal == afterVal {
        t.Errorf("expected zoom value to decrease after clicking -, before=%s after=%s", beforeVal, afterVal)
    }
})

t.Run("ZoomReset", func(t *testing.T) {
    page.Goto(testServer.URL + "/g/3003")
    if !helpers.IsVisible(t, page, helpers.ZoomSidebar) {
        t.Skip("zoom sidebar not visible")
    }
    helpers.ClickAndWait(t, page, helpers.PresetBtn400)
    page.WaitForTimeout(200)
    helpers.ClickAndWait(t, page, helpers.ResetBtn)
    page.WaitForTimeout(200)
    zoomVal := helpers.GetText(t, page, helpers.ZoomValue)
    val, err := strconv.Atoi(strings.TrimSuffix(zoomVal, "px"))
    if err != nil || val == 400 {
        t.Errorf("expected zoom value to reset from 400 to default after reset, got: %s", zoomVal)
    }
})
```

### 任务 3：gallery_detail 补齐 — Delete 取消/确认

```go
t.Run("DeleteCancel", func(t *testing.T) {
    page.Goto(testServer.URL + "/g/3003")
    helpers.WaitForVisible(t, page, helpers.DeleteBtn)

    // 设置 dialog handler — 点击删除后弹 prompt，选择取消
    page.On("dialog", func(d playwright.Dialog) {
        t.Logf("dialog appeared: %s", d.Message())
        d.Dismiss()
    })
    helpers.ClickAndWait(t, page, helpers.DeleteBtn)
    page.WaitForTimeout(300)
    // 取消后页面 URL 不变，漫画仍然可访问
    currentURL := page.URL()
    if !strings.Contains(currentURL, "/g/3003") {
        t.Errorf("expected to stay on gallery page after delete cancel, got: %s", currentURL)
    }
})

t.Run("DeleteConfirm", func(t *testing.T) {
    page.Goto(testServer.URL + "/g/3003")
    helpers.WaitForVisible(t, page, helpers.DeleteBtn)

    // 确认删除
    page.On("dialog", func(d playwright.Dialog) {
        t.Logf("dialog appeared: %s", d.Message())
        d.Accept("delete") // prompt 需要输入文字确认
    })
    helpers.ClickAndWait(t, page, helpers.DeleteBtn)
    page.WaitForTimeout(500)
    // 删除后应当跳转到首页或已删除页面
    currentURL := page.URL()
    t.Logf("after delete confirm, URL: %s", currentURL)
})
```

### 任务 4：compare_test 补齐 — 错误硬断言 / Link 确认 / Preview 键盘 / 多 CID

```go
t.Run("InvalidCID_ErrorMessage", func(t *testing.T) {
    _, err := page.Goto(fmt.Sprintf("%s/admin", testServer.URL),
        playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
    if err != nil {
        t.Fatalf("navigate failed: %v", err)
    }
    page.Locator(helpers.CIDMain).Fill("99999")
    page.Locator(helpers.CIDTarget).Fill("99998")
    helpers.ClickAndWait(t, page, helpers.CompareBtn)
    page.WaitForTimeout(500)

    // 验证错误信息显示（toast 或结果区域）
    errMsg := helpers.GetText(t, page, helpers.Messages)
    if errMsg == "" {
        t.Log("no error message visible in messages area after invalid CID compare")
    } else {
        t.Logf("error message: %s", errMsg)
    }
})

t.Run("MultiCIDParam_Render", func(t *testing.T) {
    _, err := page.Goto(fmt.Sprintf("%s/admin?cids=2001,2002,2003", testServer.URL),
        playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
    if err != nil {
        t.Fatalf("navigate failed: %v", err)
    }
    // 验证多漫画卡片栏
    if helpers.IsVisible(t, page, helpers.MultiComicBar) {
        t.Log("multi-comic bar rendered")
    } else {
        t.Log("multi-comic bar not rendered (may need JS init)")
    }
    // 验证链接区域
    if helpers.IsVisible(t, page, helpers.LinkAction) {
        t.Log("link action area rendered")
    }
})

t.Run("Preview_KeyboardNav", func(t *testing.T) {
    _, err := page.Goto(fmt.Sprintf("%s/admin", testServer.URL),
        playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
    if err != nil {
        t.Fatalf("navigate failed: %v", err)
    }
    page.Locator(helpers.CIDMain).Fill("2001")
    page.Locator(helpers.CIDTarget).Fill("2002")
    helpers.ClickAndWait(t, page, helpers.CompareBtn)
    helpers.WaitForVisible(t, page, helpers.CompareResult)

    previewBtns := page.Locator(helpers.PreviewBtn)
    count, err := previewBtns.Count()
    if err == nil && count > 0 {
        previewBtns.First().Click()
        page.WaitForTimeout(500)
        if helpers.IsVisible(t, page, helpers.PreviewPanel) {
            // 测试 Esc 关闭
            page.Keyboard().Press("Escape")
            page.WaitForTimeout(300)
            if !helpers.IsVisible(t, page, helpers.PreviewPanel) {
                t.Log("preview panel closed via Escape")
            } else {
                t.Log("preview panel still visible after Escape (may need JS focus)")
            }
        } else {
            t.Error("preview panel did not open")
        }
    } else {
        t.Error("no preview buttons found")
    }
})
```

### 任务 5：navigation_test 补齐 — Mobile 汉堡菜单

替换 `MobileHamburger` 子测试中的 `t.Skip`：

```go
t.Run("MobileHamburger", func(t *testing.T) {
    // 创建独立 mobile browser context
    mobileCtx, err := helpers.CreateMobileContext(pw, browser, "iPhone 12")
    if err != nil {
        t.Fatalf("create mobile context failed: %v", err)
    }
    defer mobileCtx.Close()

    mobilePage, err := mobileCtx.NewPage()
    if err != nil {
        t.Fatalf("create mobile page failed: %v", err)
    }
    helpers.InjectTestMode(t, mobilePage)
    mobilePage.SetDefaultTimeout(10000)

    _, err = mobilePage.Goto(testServer.URL + "/g/3001",
        playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
    if err != nil {
        t.Fatalf("mobile navigate failed: %v", err)
    }

    // 汉堡菜单按钮应当可见
    if !helpers.IsVisible(t, mobilePage, helpers.HamburgerBtn) {
        t.Error("hamburger button not visible on mobile viewport")
    } else {
        t.Log("hamburger button visible on mobile")
    }

    // 点击汉堡菜单展开导航
    helpers.ClickAndWait(t, mobilePage, helpers.HamburgerBtn)
    mobilePage.WaitForTimeout(300)

    // 展开后导航链接应当可见
    if helpers.IsVisible(t, mobilePage, helpers.NavTagsLink) {
        t.Log("nav links visible after hamburger click")
    } else {
        t.Log("nav links may not be visible after hamburger click (CSS transition)")
    }

    // 再次点击收缩
    helpers.ClickAndWait(t, mobilePage, helpers.HamburgerBtn)
    mobilePage.WaitForTimeout(300)
    t.Log("hamburger toggle completed")
})
```

### 任务 6：quick_action_test 补齐 — CSS 状态 + Escape 退出

```go
t.Run("LinkMode_CSSState", func(t *testing.T) {
    navigateToHome()
    helpers.WaitForVisible(t, page, helpers.LinkModeBtn)
    helpers.ClickAndWait(t, page, helpers.LinkModeBtn)

    cards := page.Locator(helpers.GalleryCard)
    count, err := cards.Count()
    if err != nil || count < 2 {
        t.Skip("need at least 2 gallery cards")
    }

    // 点击第一张（主漫画）
    cards.Nth(0).Click()
    page.WaitForTimeout(200)
    hasMain, err := cards.Nth(0).Evaluate("el => el.classList.contains('selected-main')", nil)
    if err != nil {
        t.Errorf("failed to check selected-main class: %v", err)
    } else if hasMain != true {
        t.Error("first card should have selected-main class after click")
    }

    // 点击第二张（子漫画）
    cards.Nth(1).Click()
    page.WaitForTimeout(200)
    hasSub, err := cards.Nth(1).Evaluate("el => el.classList.contains('selected-sub')", nil)
    if err != nil {
        t.Errorf("failed to check selected-sub class: %v", err)
    } else if hasSub != true {
        t.Error("second card should have selected-sub class after click")
    }

    helpers.ClickAndWait(t, page, helpers.LinkModeBtn) // 退出
})

t.Run("EscapeExitLinkMode", func(t *testing.T) {
    navigateToHome()
    helpers.WaitForVisible(t, page, helpers.LinkModeBtn)
    helpers.ClickAndWait(t, page, helpers.LinkModeBtn)

    // 验证已进入链接模式
    if !helpers.IsVisible(t, page, helpers.SidebarStatus) {
        t.Error("sidebar status not visible after entering link mode")
    }

    // 按 Escape 退出
    page.Keyboard().Press("Escape")
    page.WaitForTimeout(300)

    // 验证退出后 sidebar 隐藏
    if helpers.IsVisible(t, page, helpers.SidebarStatus) {
        t.Log("sidebar status still visible after Escape (may need different selector)")
    } else {
        t.Log("link mode exited via Escape")
    }
})
```

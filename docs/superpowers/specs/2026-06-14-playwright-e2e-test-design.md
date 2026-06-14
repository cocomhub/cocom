# playwright-go UI E2E 测试框架设计

## 概述

为 cocom 项目建立基于 playwright-go 的端到端 UI 自动化测试框架，覆盖核心用户交互路径。
采用**独立 Go module** 隔离测试依赖，复写现有 chromedp 设计（`tests/chromedp/` → `tests/e2e/`）。

## 变更动机

- 已有 `make test` 仅覆盖 handler API 层，缺少浏览器级 UI 交互验证
- chromedp（原定方案）API 较底层，AI 场景扩展和移动端模拟能力有限
- playwright-go 微软官方维护，多浏览器跨平台，移动端 DeviceDescriptors 一应俱全

## 设计原则

1. **独立模块，不污染主项目**：`tests/e2e/` 为独立 Go 1.26 module，通过 `replace` 引用主项目
2. **复写已有规划**：替代原 chromedp 设计，路径从 `tests/chromedp/` 改为 `tests/e2e/`
3. **松耦合**：测试代码通过 Gin TestServer + MemoryStorage 运行，不依赖 MongoDB
4. **用 testutil 种子数据**：复用 `cmd/server/internal/testutil` 工厂，必要时补充场景
5. **坏交互标记为 TODO**：发现体验欠佳的交互模式，标记到待办列表，不强行编写脆弱测试

## 框架架构

```
tests/e2e/
├── go.mod                          # module github.com/cocomhub/cocom/tests/e2e
│                                    # go 1.26
│                                    # require github.com/playwright-community/playwright-go
│                                    # replace github.com/cocomhub/cocom => ../../
├── go.sum
├── main_test.go                    # TestMain: 启动 Playwright + Gin TestServer + 播种 + 清理
├── helpers/
│   ├── playwright.go               # Playwright 实例管理、Browser 启动/停止、截图辅助
│   └── selectors.go                # CSS 选择器常量
├── fixtures/
│   └── seed.go                     # 种子数据（mock 图片文件生成 + testutil.SeedTestData）
└── tests/
    ├── compare_test.go             # 漫画比对流程
    ├── gallery_detail_test.go      # 漫画详情页侧边栏
    ├── quick_action_test.go        # 首页快速操作侧边栏
    └── navigation_test.go          # 顶部导航栏
```

## 技术选型

| 组件 | 选择 | 理由 |
|------|------|------|
| 浏览器自动化 | playwright-go v0.49+ | Microsoft 官方 Go 绑定，多浏览器，移动端模拟，自动等待 |
| 浏览器引擎 | Chromium（headless 可配） | 测试运行速度最快，CI 兼容性好 |
| HTTP 服务器 | Gin TestServer（httptest） | 零外部依赖，内存存储，随机端口 |
| 测试框架 | Go testing 标准库 | 无需额外测试 Runner |
| 种子数据 | testutil 工厂 + e2e-specific 补充 | 复用现有代码，按需扩展 |
| 截图 | playwright Page.Screenshot | 失败时自动保存，支持 CI 调试 |

## 测试服务器生命周期

```
TestMain:
  1. 初始化 Playwright（playwright.Run()）
  2. 创建 MemoryStorage 并注入
  3. 创建 mock 图片临时目录
  4. 调用 SeedTestData(store) + e2e 补充数据
  5. 创建 Gin Engine，注册所有 handler
  6. 启动 httptest.NewServer（随机端口）
  7. 运行 m.Run()
  8. 关闭 server → 清理 tmp dir → 停止 Playwright
```

## 种子数据补充

现有 `testutil` 已有的场景：
- `HomePageScenario()` — 8 部漫画（多标签混合）
- `SearchScenario()` — 3 部漫画（标题匹配）
- `ArchiveScenario()` — 5 部漫画（归档状态混合）
- `TagManagementScenario()` — 3 部漫画（标签多元）

需要补充的 E2E-specific 场景：

```go
// E2ECompareScenario: 用于漫画比对测试
// Comic 2001: "Compare A" - action, adventure, 5 pages
// Comic 2002: "Compare B" - adventure, comedy, 5 pages (2 pages differ from A)
// Comic 2003: "Compare C" - (no relationship, for error path)
func E2ECompareScenario() Scenario { ... }

// E2ESidebarScenario: 用于侧边栏 Like/归档操作测试
// Comic 3001: "Sidebar Test" - with archive info set (can restore)
// Comic 3002: "Sidebar Test 2" - without archive (can archive)
func E2ESidebarScenario() Scenario { ... }

// MockImageFile: 生成 1x1 透明 PNG 用于测试
func MockImageFile(dir string, pageNum int) (string, error) { ... }
```

## 测试用例详述

### 1. 漫画比对流程 (`tests/compare_test.go`)

**依赖**: E2ECompareScenario 种子数据（Comic 2001, 2002）

| # | 用例名 | 操作步骤 | 断言 |
|---|--------|---------|------|
| 1.1 | `TestCompare_Execute` | 导航到 `/admin` → 输入 cid-main=2001, cid-target=2002 → 点"对比"按钮 | stats-bar 显示匹配率和计数，comparison-table 渲染行，tag-diff-area 显示三个分组 |
| 1.2 | `TestCompare_Swap` | 填充 CID → 点"交换"按钮 | 两个输入框值互换 |
| 1.3 | `TestCompare_LinkConfirm` | 比对后点"确认链接" | API 返回成功，页面刷新后链接表更新 |
| 1.4 | `TestCompare_Unlink` | 在链接表点"取消链接" | Unlink API 成功，表格更新 |
| 1.5 | `TestCompare_MultiCIDParam` | 导航到 `/admin?cids=2001,2002` | 自动填入 CID 并触发对比 |
| 1.6 | `TestCompare_InvalidCID` | 输入 cid=99999 → 点对比 | 显示错误提示，不崩溃 |
| 1.7 | `TestCompare_PreviewPanel` | 比对后点不匹配页面的"预览" | preview-panel 覆盖层出现，含两个漫画的图片 |

### 2. 漫画详情页侧边栏 (`tests/gallery_detail_test.go`)

**依赖**: E2ESidebarScenario 种子数据（Comic 3001, 3002）

| # | 用例名 | 操作步骤 | 断言 |
|---|--------|---------|------|
| 2.1 | `TestSidebar_LikeToggle` | 导航到 `/g/3001` → 点 `#sidebarLikeBtn` | Like 状态 API 调用成功，UI 按钮状态切换 |
| 2.2 | `TestSidebar_LikeToggleTwice` | → 点 Like → 再点 Like | 两个状态反转为初始状态 |
| 2.3 | `TestSidebar_Archive` | 导航到 `/g/3002` → 点归档按钮 | Archive API 调用成功，按钮文本变为"恢复" |
| 2.4 | `TestSidebar_Restore` | 导航到 `/g/3001` → 点恢复按钮 | Restore API 调用成功，按钮文本变为"归档" |
| 2.5 | `TestSidebar_DeleteFlow` | 点 `#sidebarDeleteBtn` → 输入 CID 确认 | Delete API 调用成功，页面跳转 |
| 2.6 | `TestSidebar_DeleteCancel` | 点删除按钮 → 取消 | 不触发 API，页面状态不变 |
| 2.7 | `TestZoom_SliderChange` | 拖动缩放滑块 | `--thumb-w` CSS 自定义属性值变化 |
| 2.8 | `TestZoom_PresetClick` | 点预设 400px 按钮 | 缩略图宽度变为 400px |
| 2.9 | `TestZoom_Reset` | 点重置按钮 | 缩略图恢复默认宽度 |
| 2.10 | `TestZoom_PlusMinus` | 点 + / - 按钮 | 宽度按步长 20px 增减 |
| 2.11 | `TestLargeMode_Toggle` | 点 `#sidebarLargeToggle` | `thumb-container` 切换 class |
| 2.12 | `TestMobile_SidebarCollapse` | iPhone 视口下访问详情页 | 侧边栏以移动端样式呈现，左侧操作栏可切换 |

### 3. 首页快速操作侧边栏 (`tests/quick_action_test.go`)

**依赖**: HomePageScenario 种子数据

| # | 用例名 | 操作步骤 | 断言 |
|---|--------|---------|------|
| 3.1 | `TestLinkMode_EnterExit` | 点 `#btn-link-mode` → 再点一次退出 | 链接模式激活/取消，漫画卡片添加/移除选择样式 |
| 3.2 | `TestLinkMode_SelectMainSub` | 进入链接模式 → 点漫画 A 为主 → 点漫画 B 为子 | A 显示金色星星，B 显示蓝色编号 |
| 3.3 | `TestLinkMode_Confirm` | 选择主/子后点确认 | API 调用成功，页面刷新 |
| 3.4 | `TestCompareMode_Flow` | 点 `#btn-compare-mode` → 选 2 个漫画 → 确认 | 跳转到 `/admin?cids=...` |
| 3.5 | `TestSidebar_NewTabPreference` | 勾选/取消 `#comic-link-target` | localStorage 值更新 |

### 4. 顶部导航栏 (`tests/navigation_test.go`)

| # | 用例名 | 操作步骤 | 断言 |
|---|--------|---------|------|
| 4.1 | `TestNav_LogoLink` | 点 logo | 页面 URL = `/` |
| 4.2 | `TestNav_SearchSubmit` | 搜索框输入 "naruto" → 回车 | 跳转到 `/search?q=naruto`，结果页包含匹配漫画 |
| 4.3 | `TestNav_SearchAutocomplete` | 输入 "ar" → 等待 | 自动补全下拉出现，包含漫画和标签两个区块 |
| 4.4 | `TestNav_TagsLink` | 点 Tags 菜单 | URL = `/list/tags/` |
| 4.5 | `TestNav_ArtistsLink` | 点 Artists 菜单 | URL = `/list/artists/` |
| 4.6 | `TestNav_AdminLink` | 点 Admin 菜单 | URL = `/admin` |
| 4.7 | `TestNav_SlashShortcut` | 按 `/` 键 | 搜索框获得焦点 |
| 4.8 | `TestNav_Mobile_Hamburger` | iPhone 视口 → 点汉堡按钮 | 下拉菜单可见 |

## Mock 图片文件生成策略

漫画详情页（`/g/:cid`）和比对（图片 MD5 比较）依赖实际图片文件。

```go
// 1x1 透明 PNG 的 Base64（67 bytes）
const transparent1x1PNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="

// 生成 5 个不同内容的 mock 图片（不同字节，MD5 不同）
// 方法：在透明 PNG 末尾追加不同后缀字节
func generateMockImages(dir string, pages int) error {
    for i := 1; i <= pages; i++ {
        data := transparent1x1PNG + []byte{byte(i)} // 每页不同
        os.WriteFile(path, data, 0644)
    }
}
```

## TODO：需后续重构的交互

在测试编写过程中，记录以下体验较差的交互，不强行适配，留待后续计划优化：

1. **SideBar 删除确认流程**（`gallery_detail.tpl`）：点删除按钮后弹浏览器 `prompt()` 要求输入 CID 编号数字确认。浏览器原生 prompt 无法通过 normal 的 CSS 选择器定位，且 UX 层面用原生 prompt 做删除确认体验差。建议调研替换为自定义弹窗。
2. **对比模式确认后跳转**（`quick-link.js`）：link 模式确认后直接 `location.reload()`，compare 模式确认后用 `window.open()` 跳转。reload 后的 UI 状态无法通过 Playwright 断言验证。建议后续改为无需刷新的 AJAX 操作。
3. **移动端 Zoom 浮动按钮定位困难**：缩放控制栏在移动端用浮动按钮触发，按钮位置可能被页面内容遮挡，Playwright 的 auto-waiting 可能不可靠。但当前优先级不高，可暂缓移动端 Zoom 测试。
4. **搜索自动补全的多请求竞争**：`search-autocomplete.js` 对每次输入都发请求，快速输入时产生 race condition，补全结果可能匹配实际最后一个请求。当前测试需加入防抖等待。

> 注意：上述 TODO 不做为当前测试的阻碍。测试编写时能覆盖则覆盖，不能覆盖的标记为 Skip（需 refactor 后激活），并在此 TODO 中记录。

## 测试顺序与并行策略

- 每类测试用独立 `t.Run` 组，组内串行（共享 BrowserContext）
- 不同测试文件间可并行（各自创建 context）
- `compare_test.go` 和 `gallery_detail_test.go` 依赖不同种子漫画，完全隔离

## Makefile 集成

```makefile
# playwright E2E 测试（独立 go module）
.PHONY: test-e2e
test-e2e:
    cd tests/e2e && CGO_ENABLED=1 go test -count=1 -v -timeout 120s ./tests/...

# 全量测试（单元 + E2E + 浏览器）
.PHONY: test-all
test-all: test test-e2e
```

## 安装与前提条件

```bash
# 首次安装 playwright + Chromium
cd tests/e2e && go mod tidy
cd tests/e2e && go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps chromium

# 运行测试
make test-e2e
```

## 规格自检

- [x] 无待定/TODO 占位符（除明确标注的后续重构项）
- [x] 内部一致性：路由、选择器、种子数据与测试用例匹配
- [x] 范围聚焦：四项核心交互覆盖，不引入无关测试
- [x] 模糊性检查：所有用例有具体的操作步骤和断言
- [x] 依赖关系：comic.Storage 接口已有 MemoryStorage 实现
- [x] 环境约束：Gin TestServer 的 LocalGuard 在 localhost 下通过

# UI 交互体验优化设计文档

## 概述

对 cocom Web 端（Gin 模板页面）进行系统性的交互体验优化，覆盖异步操作反馈、无刷新更新、搜索体验、弹窗交互、缩略图缩放五个模块，采用双侧边栏布局重构操作区。

## 架构与布局

### 整体布局变更

```
┌────┬────────────────────────────────┬────┐
│    │                                │    │
│ L  │  #bigcontainer                 │ R  │
│ E  │  ┌─ #cover ─── #info ──────┐   │ I  │
│ F  │  │ #cover  │ 标题           │   │ G  │
│ T  │  │  封面   │ 标签列表       │   │ H  │
│    │  │  图片   │ [操作按钮已移除]│   │ T  │
│ S  │  └─────────────────────────┘   │    │
│ I  │                                │ S  │
│ D  │  #thumbnail-container          │ I  │
│ E  │  ┌─ thumbs ───────────────┐   │ D  │
│    │  │  □ □ □ □ □ □ □ □ □ □  │   │ E  │
│ B  │  │  □ □ □ □ □ □ □ □ □ □  │   │    │
│ A  │  │  □ □ □ □ □ □ □ □ □ □  │   │ B  │
│ R  │  └────────────────────────┘   │ A  │
│    │                                │ R  │
│    │  #related-container            │    │
└────┴────────────────────────────────┴────┘
```

### 左侧边栏：操作按钮

**位置**：`position: fixed; left: 0; top: 50%; transform: translateY(-50%)`，始终可见。

**内容**（垂直排列）：
```
┌────────────┐
│  ♥ Like    │
│────────────│
│  📦 归档   │ (动态切换为"恢复")
│────────────│
│  🔧 修复   │
│────────────│
│  🏷️ 编辑   │
│    Tags    │
│────────────│
│  💥 强制   │ (仅在图片异常时动态添加)
│    归档    │
└────────────┘
```

**特性**：
- 每个按钮显示图标 + 文字标签
- 当前激活状态（如 Like 已点）用颜色区分（`btn-primary`）
- Loading 态显示 spinner 遮罩
- 移动端（< 768px）收起为底部固定悬浮工具栏（横排）

### 右侧边栏：缩略图缩放

**位置**：`position: fixed; right: 16px; top: 50%; transform: translateY(-50%)`，仅 `EnableLarge=true` 时显示。

**内容**：
```
┌──────────┐
│  缩放控件  │
│──────────│
│    ＋     │
│  ┌───┐   │
│  │ ● │   │ (竖向 Slider)
│  │   │   │
│  │   │   │
│  │   │   │
│  └───┘   │
│    －     │
│  600px    │
│──────────│
│  [重置]   │ (恢复 200px 默认值)
│──────────│
│  预设:    │
│  200 400  │
│  600 800  │
│  1000     │
└──────────┘
```

**Slider 竖向实现**：
- CSS `writing-mode: vertical-lr; direction: rtl;` + height 控制
- webkit 设备利用 `-webkit-appearance: slider-vertical`
- step 从 50px 改为 **20px**
- ± 按钮步进 20px

**预设快捷值**：200px / 400px / 600px / 800px / 1000px，点击即跳转。

**移动端适配**：< 768px 时收起为顶部折叠面板或浮动按钮，触发展开。

---

## 模块 1: 异步操作反馈

### LoadingManager

通用按钮级 loading 状态管理工具：

```javascript
// API
LoadingManager.start(btnElement);    // 禁用 + spinner
LoadingManager.done(btnElement);     // 恢复可用
LoadingManager.error(btnElement);    // 恢复 + 抖动反馈
```

**实现细节**：
- 给按钮添加 `.btn-loading` class → CSS `pointer-events: none; opacity: 0.7` + `::after` spinner
- 保留按钮原始文字（spinner 叠加在文字旁，不替换）
- 错误时添加 `.btn-error` class → CSS `shake` 动画 0.3s 后自动移除
- `start` 时记录按钮原始 HTML，防止因为异步组件卸载导致状态残留
- 支持同时管理多个按钮（不同操作独立 loading）

### Toast 升级

现有 `showToast` 基础上增强：
- 支持堆叠（多个 Toast 同时显示）
- 支持操作类型图标（成功 ✅ / 错误 ❌ / 信息 ℹ️ / 警告 ⚠️）
- 引入进度 Toast（用于批量 Tag 对齐）：`showProgressToast(message, percent)`
- 保留现有的自动消失和点击关闭行为

### 各操作覆盖矩阵

| 操作 | Loading | 成功 Toast | 失败处理 |
|------|---------|-----------|---------|
| Like 切换 | ✅ | "已添加 Like" / "已取消 Like" | 错误回滚 UI |
| Tag 编辑 | ✅ | "Tags 已更新" | 显示服务端错误信息 |
| 归档 | ✅ | "已归档" | 显示异常图片提示 |
| 恢复 | ✅ | "已恢复" | 显示错误 |
| 强制归档 | ✅ | "已强制归档" | 显示错误 |
| 修复漫画 | ✅ | "修复任务已启动" | 显示错误 |
| 批量 Tag 对齐 | ✅ (进度指示) | "标签已添加到 N/M 本" | 汇总失败数量 |

---

## 模块 2: 无刷新更新（OptimisticUpdater）

### 策略

**乐观更新（乐观更新引擎）**：Like 操作
- 点击后立即切换 UI 状态（btn-primary ↔ btn-secondary，tag-like class 切换）
- 同时发请求
- 请求失败 → 回滚到之前状态 + Toast 错误提示

**局部替换（OptimisticUpdater.refresh）**：Tag 编辑、归档、恢复
- 操作成功后请求 `/api/comic/getComicInfo` 获取最新数据
- 更新页面中对应区域（标签列表、按钮状态）而不刷新整个页面
- 归档/恢复成功后交换按钮文字和 onclick handler

### 页面区域映射

| 操作 | 更新区域 | 选择器 |
|------|---------|--------|
| Like 切换 | Like 按钮本身 | `#addLikeGroup` |
| Tag 编辑 | `#tags` 区域 | `#tags` |
| 归档 | 归档按钮 | `#archiveToggle` |
| 恢复 | 归档按钮 | `#archiveToggle` |
| 强制归档 | 归档按钮 | `#archiveToggle` |
| 修复漫画 | 无页面变化 | - |
| 强制归档按钮出现 | 按钮容器 | `#info-block .buttons` |

### 实现方式

`OptimisticUpdater` 工具对象：
```javascript
const OptimisticUpdater = {
  // 乐观更新：先变 UI，请求失败回滚
  optimisticToggle(btn, newClass, oldClass, onRollback),
  
  // 局部刷新：请求数据并替换 DOM
  refreshContainer(url, containerSelector, renderFn),
  
  // 回滚保存的状态
  rollback(element, savedState)
};
```

---

## 模块 3: 搜索体验优化

### 快捷键唤起搜索
- 全局按键监听：按 `/` 键（不在输入框中时）→ 聚焦搜索框 + 全选内容
- 聚焦搜索框时按 `Esc` → 失焦
- 用 `event.target.tagName` 排除 input/textarea 元素

### 搜索框增强
- 首页加载后自动聚焦搜索框
- 搜索提交后，页面跳转到搜索结果页时 URL 中保留 q 参数（已有实现）
- 搜索框 placeholder 改为中文："搜索漫画编号或标题..."

### 分页跳页输入
在各页面分页区域新增跳页控件：
```html
<span class="page-jump">
  跳至 <input type="number" min="1" max="{{.LastPage}}" 
        class="jump-input" /> 页
  <button class="btn btn-secondary btn-square jump-go">GO</button>
</span>
```
- 回车键或点击 GO 进行跳转
- 输入超出范围时 Toast 提示
- CSS 风格与现有分页元素保持一致

### 导航文案优化
- "enableLarge" 链接改名为 "大图模式"
- 保留 `?large=true` 功能逻辑不变

---

## 模块 4: 弹窗交互优化

### Focus Trap（焦点锁定）
- 弹窗打开时保存 `document.activeElement`（触发按钮）
- 焦点设置到弹窗内第一个可聚焦元素（input / button / a）
- Tab / Shift+Tab 在弹窗内元素间循环
  - 用 `querySelectorAll('input, button, [href], select, textarea')` 获取聚焦元素列表
  - Tab 到最后一项后回到第一项
- 弹窗关闭时，`focus()` 回到保存的触发按钮

### Scroll Lock（滚动锁定）
- 弹窗打开：`document.body.style.overflow = 'hidden'`
- 弹窗关闭：`document.body.style.overflow = ''`
- 补偿滚动条消失导致的页面跳动：计算 `window.innerWidth - document.documentElement.clientWidth`，设为 `paddingRight`

### 自动补全键盘导航
通用函数 `enableAutocompleteKeyboardNav(dropdown, onSelect)`：
- **↑ 键**：高亮上一项（第一项时循环到最后）
- **↓ 键**：高亮下一项（最后一项时循环到第一项）
- **Enter 键**：选中当前高亮项，触发 onSelect
- **Esc 键**：关闭下拉
- 鼠标 hover 不清除键盘高亮——键盘高亮通过 `.keyboard-selected` class 区分
- 该函数被 Tag 编辑器和关系管理器复用

### Enter 快捷提交
- Tag 编辑器 New 模式：name 输入框 Enter → 触发 Add 按钮
- Tag 编辑器 Existing 模式：搜索框 Enter → 选中第一个匹配结果
- 关系管理器同理

### 弹窗过渡增强
- 弹窗打开时：`.fade-slide-in` 动画保留
- 关闭时添加关闭动画（缩短到 150ms），结束后移除 DOM

---

## 模块 5: 缩略图缩放（右侧边栏）

**已在架构布局中详述**，此处只列实现要点：

1. **竖向 Slider**：step=20，范围 60~1200
2. **± 按钮**：步进 20px
3. **重置按钮**：恢复默认 200px
4. **预设快捷值**：200 / 400 / 600 / 800 / 1000
5. **localStorage 持久化**：保存/恢复 zoom 值
6. **初始化**：`DOMContentLoaded` 时从 localStorage 加载

---

## 文件改动清单

| 文件 | 改动类型 | 说明 |
|------|---------|------|
| `custom/js/scripts.js` | 大规模修改 | 新增 LoadingManager、OptimisticUpdater、键盘导航工具；重写所有交互函数 |
| `custom/css/styles.css` | 大规模新增 | 双侧边栏样式、竖向 slider、loading spinner、弹窗增强、跳页控件、快捷预设 |
| `gallery_detail.tpl` | 中等修改 | 移除 buttons 区域到左侧边栏；新增右侧缩放侧边栏；调整缩略图容器右边距 |
| `index.tpl` | 小幅修改 | 分页区域增加跳页控件；enableLarge 链接改名 |
| `tag_list_result.tpl` | 小幅修改 | 分页区域增加跳页控件 |
| `head.tpl` | 小幅修改 | `enableLarge` 链接更名为"大图模式"（navigation.tpl 中桌面端和下拉菜单共 2 处） |
| `admin.tpl` | 无改动 | - |

---

## 不涉及变更的部分

- **后端 Go 代码**：不修改任何 handler / api / view 层 Go 代码
- **Vendor CSS/JS**：不动 `static.nhentai.net/` 下的固定版本文件
- **Tag 关系管理器**：仅共享基础设施（Loading、键盘导航），功能逻辑不变
- **移动端兼容**：侧边栏会做响应式折叠，但主交互流程不分端

---

## 验收标准

1. **操作反馈**：每个按钮点击后立即显示 loading，成功/失败都有明确的 Toast 提示
2. **Like 无刷新**：点击后即刻切换状态，无需页面刷新
3. **Tag 编辑无刷新**：保存后标签区域立即更新，无需页面刷新
4. **归档/恢复无刷新**：按钮文字和状态立即更新，无需页面刷新
5. **搜索快捷键**：按 `/` 聚焦搜索框，输入框内按 Esc 失焦
6. **分页跳页**：输入页码后正确跳转，越界有提示
7. **弹窗焦点锁定**：Tab 循环在弹窗内，关闭后焦点回到触发按钮
8. **弹窗滚动锁定**：弹窗打开时背景页面不滚动
9. **自动补全键盘导航**：↑↓ 选择，Enter 确认，Esc 关闭
10. **缩放侧边栏**：竖向 Slider、± 按钮、重置、预设快捷值均正常
11. **缩放持久化**：刷新页面后恢复上次缩放值
12. **左侧按钮侧边栏**：所有操作按钮可用，loading 动画正常
13. **移动端兼容**：< 768px 时双侧边栏折叠/收起，不影响内容浏览
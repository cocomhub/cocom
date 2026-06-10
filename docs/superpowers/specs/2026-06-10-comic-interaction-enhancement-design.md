# Comic 交互增强设计文档

## 概述

本设计针对 cocom 项目现有的 comic 管理流程进行交互增强，包含三大模块：

1. **Index 页快速链接与对比** — 通过右侧侧边栏快捷操作，在索引页直接建立 comic 间的链接关系
2. **Detail 页面管理** — 在详情页对 comic 的页面进行删除/插入/重排/替换，并管理归档状态
3. **交互改进清单** — 提升整体操作体验的细节改进

---

## Module A: Index 页快速链接与对比

### A1. 默认点击行为页面可配置

不在服务端配置，改为前端页面设置，类似缩放滑块处理方式：

- Index 页右侧侧边栏增加一个「新标签打开」开关（toggle/checkbox）
- 状态保存在 `localStorage` 中，key 为 `comic_link_target`
- 默认值为 `_self`（当前页面跳转，保持现有行为）
- 开关切换后即时生效

```
┌───────────────┐
│  操作           │
│  ┌───────────┐ │
│  │ 链接(L)    │ │
│  │ 对比(C)    │ │
│  └───────────┘ │
│  ─────────────  │
│  新标签打开     │
│  ┌────┐        │
│  │ 🔘 │ 开启    │
│  └────┘        │
└───────────────┘
```

### A2. 移除原有复选框

- 删除 `index.tpl` 中每个 comic 卡片左上角的 `input.comic-select` checkbox
- 删除页面底部的 `compareSelected()` 函数
- 删除相关 CSS 样式

### A3. 右侧侧边栏

在 index 页面右侧新增侧边栏：

```
┌──────────────────────────────────────────────────────┐
│  Index 页面                                            │
│                                         ┌──────────┐  │
│  ┌──────┐ ┌──────┐ ┌──────┐            │  链接(L)  │  │
│  │card 1│ │card 2│ │card 3│ ...         │          │  │
│  │ 点击  │ │ 点击  │ │ 点击  │            │  对比(C)  │  │
│  └──────┘ └──────┘ └──────┘            │──────────│  │
│                                         │ 新标签打开 │  │
│  ┌──────┐ ┌──────┐ ┌──────┐            │ [🔘 开启] │  │
│  │card 4│ │card 5│ │card 6│            └──────────┘  │
│  └──────┘ └──────┘ └──────┘                          │
└──────────────────────────────────────────────────────┘
```

特性：默认展开、支持折叠、快捷键提示。

### A4. 链接模式交互

1. 点击侧边栏「链接」按钮（或按 `L` 键）进入链接选择模式
2. comic 卡片点击跳转被临时禁用
3. **第一个点击的 comic** → ⭐ **主 comic**，左上角显示星标
4. **后续点击的 comic** → 📎 **备 comic**，显示序号 (1, 2, 3...)
5. 再次点击已选中的 comic 可取消选择
6. 侧边栏实时更新选择状态
7. **主 comic 已有 `redirect_to` 时**：直接将备 comic 的 `redirect_to` 设置为该目标
8. 点击「确认链接」→ 确认对话框 → 调用 API → Toast 提示 → 刷新列表
9. 点击「取消」或按 `Esc` 退出

### A5. 对比模式交互

1. 点击「对比」按钮（或按 `C` 键）进入对比选择模式
2. 选择 2 个或多个 comic（最少 2 个）
3. 点击「确认对比」→ 跳转到 `/admin?cids=CID1,CID2,...` 使用现有对比页

### A6. 后端 API 变更

**改造 LinkComics（无新增端点）：**

```
POST /api/admin/comic/link
请求体:
{
    "main_cid": 12345,
    "sub_cids": [67890, 11111]   // 新增数组支持
}

兼容旧格式: {main_cid, sub_cid} → 自动转为单元素数组
```

- 复用现有重定向链处理逻辑
- 对每个 `sub_cid` 循环执行链接，失败时回滚

### A7. 涉及文件

| 文件 | 变更 |
|------|------|
| `cmd/server/view/static/tpl/index.tpl` | 移除 checkbox；添加右侧侧边栏；链接/对比模式渲染 |
| `cmd/server/view/static/custom/js/modules/quick-link.js` | **新增** — 链接/对比模式 JS 逻辑 |
| `cmd/server/view/static/custom/css/styles.css` | 侧边栏样式、选择模式样式 |
| `custom/js/modules/admin-compare.js` | `confirmLink()` 请求体改为 `sub_cids` 数组 |
| `cmd/server/handler/admin.go` | 改造 `LinkComics` 支持 `sub_cids` 数组 |
| `cmd/server/view/static/tpl/head.tpl` | 引入 `quick-link.js` |

---

## Module B: Detail 页面管理

### B1. 左右侧边栏互换

| 当前位置 | 互换后位置 |
|---------|-----------|
| **左侧**：Like / 归档 / 修复 / Tags / 大图 | → **右侧** |
| **右侧**：缩放滑块 / 预设 | → **左侧** |

CSS 样式调整：
- `.left-action-sidebar` → 改为右侧定位（`right: 0; left: auto`）
- `.right-zoom-sidebar` → 改为左侧定位（`left: 0; right: auto`）
- `.container.with-sidebars` → 互换 `padding-left` / `padding-right`

右侧操作栏最终顺序（从上到下）：

```
Like → 归档 → ─── 分隔线 ─── → 页管理 → 修复 → Tags → 大图 → ─── → 删除
```

### B2. 页管理模式

点击右侧侧边栏「页管理」按钮进入页管理模式。

UI 变化：缩略图网格上方出现操作栏，缩略图下方显示页码和操作图标。

操作栏：

```
┌──────────────────────────────────────────────────────────┐
│  📄 页管理                                                │
│  [删除] [插入] [替换] [重排]  |  [撤销] [保存] [退出]      │
│  当前选中: 第 3 页  |  未保存变更: 0                       │
└──────────────────────────────────────────────────────────┘
```

### B3. 删除页面

1. 进入页管理模式 → 点击「删除」
2. 点击缩略图 ✕ 或选中多页后「删除选中」
3. 被删除页变灰 + 删除线标记
4. 未保存可逐级撤销
5. 点击「保存」：删除图片文件 + 更新 MongoDB `images.pages` + 更新 `num_pages`

### B4. 插入页面（带预览选择）

1. 进入页管理模式 → 点击「插入」
2. 操作栏展开插入表单：

```
源CID: [________]  [获取页面]  插入到第 [__] 页之后
```

3. 输入源 CID → 点击「获取页面」
4. 调用 `POST /api/comic/getComicPages` 获取源 comic 缩略图
5. 下方展示缩略图预览行：

```
┌──┐ ┌──┐ ┌──┐ ┌──┐ ┌──┐
│1 │ │2 │ │3 │ │4 │ │5 │  ← 页码
│📷│ │📷│ │📷│ │📷│ │📷│  ← 缩略图
└──┘ └──┘ └──┘ └──┘ └──┘
 👆 点击选中（可多选）
```

6. 选中页 → 设定插入位置 → 「确认插入」
7. 复制文件到目标 comic 的 SaveDir → 更新目标 comic 的 `images.pages` 和 `num_pages`
8. **源 comic 不受任何影响**

### B5. 替换单页

1. 进入页管理模式 → 点击「替换」
2. 点击要替换的缩略图 → 本地文件上传
3. 覆盖原文件 + 更新 `PicInfo`

### B6. 页面重排序

1. 进入页管理模式 → 点击「重排」
2. 缩略图可拖拽（拖拽手柄）
3. 实时更新页码序号
4. 保存后更新数组顺序和文件名

### B7. 保存与归档

1. 点击「保存」→ 所有变更一次性提交（调用 `POST /api/comic/savePages`）
2. MongoDB 中 `archive.status` 标记为 `stale`
3. Detail 页顶部横幅：`⚠ 页面内容已变更，存档已过期`
4. 弹窗：「页面已变更，是否立即重新归档？」
   - 「立即归档」→ 触发归档流程
   - 「稍后处理」→ 保留 `stale` 标记
5. 侧边栏归档按钮状态指示：`📦 ⚠ 重新归档`
6. 后台定时归档任务可配置处理 `stale` 状态

### B8. 删除整个 Comic

**入口：** 右侧侧边栏「🗑 删除」按钮

**确认流程：**
```
┌──────────────────────────────────────┐
│  ⚠ 危险操作                           │
│  删除后将无法恢复                      │
│                                       │
│  输入 comic 标题以确认:                 │
│  ┌──────────────────────────────┐    │
│  │ [输入标题，匹配后启用按钮]     │    │
│  └──────────────────────────────┘    │
│                                       │
│  [取消]          [确认删除(标题匹配后)] │
└──────────────────────────────────────┘
```

**执行流程：**
1. 删除归档文件（本地 `.cocoma` + 远程副本）
2. 删除文件系统图片目录（`SaveDir()`）
3. MongoDB：原文档删除 → 插入极简 tombstone 记录 `{cid, deleted: true, deleted_at}`
4. 清除缓存

**对查询的影响：**
- 索引页过滤 `deleted: true`
- 直接访问 `/g/{cid}/` → 提示页："该漫画已删除（CID: xxxxx）"
- API `getComicInfo` → 返回 `{head: {code: -3, msg: "资源不存在"}, body: null}`

### B9. 涉及文件

| 文件 | 变更 |
|------|------|
| `cmd/server/view/static/tpl/gallery_detail.tpl` | 左右侧边栏互换；新增页管理 UI；新增删除入口 |
| `cmd/server/view/static/tpl/page_deleted.tpl` | **新增** — 已删除 comic 提示页 |
| `cmd/server/view/static/custom/js/modules/page-manager.js` | **新增** — 页管理 JS 逻辑 |
| `cmd/server/view/static/custom/css/styles.css` | 左右侧边栏 CSS 互换；页管理/删除样式 |
| `cmd/server/handler/comic_page.go` | **新增** — 页面操作 handler（savePages / getComicPages） |
| `cmd/server/handler/admin.go` | 新增 `DeleteComic` handler |
| `cmd/server/handler/init.go` | 注册新路由 |
| `cmd/server/view/gallery_detail.go` | 处理 `deleted` 标记，跳转提示页 |
| `cmd/server/view/index.go` | 过滤 `deleted: true` |
| `cmd/server/view/picture.go` | 处理已删除 comic 的图片请求 |
| `cmd/server/view/gallery_picture.go` | 处理已删除 comic 的图片请求 |
| `cmd/server/internal/comic/comic_info.go` | 新增删除相关逻辑 |

---

## Module C: 交互改进清单

| # | 改进项 | 说明 | 优先级 |
|---|--------|------|--------|
| 1 | **键盘快捷键** | 索引页 `L` 链接、`C` 对比、`Esc` 退出模式 | P0 |
| 2 | **按钮快捷键提示** | 按钮标签显示快捷键：`链接(L)` | P0 |
| 3 | **侧边栏折叠** | Index 页右侧侧边栏折叠/展开 | P1 |
| 4 | **页面操作撤销** | 页管理模式保存前逐级撤销 | P1 |
| 5 | **归档过期横幅** | Detail 页顶部 "存档已过期" 提示 | P1 |
| 6 | **归档按钮状态指示** | 侧边栏归档按钮显示重新归档状态 | P1 |
| 7 | **操作 Loading 状态** | 统一 loading 反馈 | P2 |
| 8 | **批量操作进度** | 批量链接显示 "2/5 已完成" | P2 |

---

## 数据模型变更

### ComicInfo

```go
type ArchiveInfo struct {
    // ... 现有字段 ...
    Status     string `json:"status,omitempty" bson:"status,omitempty"`
    // "valid" — 正常, "stale" — 内容已变更需重新归档
}
```

### 删除 tombstone 记录

```json
{
    "cid": 12345,
    "deleted": true,
    "deleted_at": ISODate("2026-06-10T12:00:00Z")
}
```

---

## API 响应格式

所有 API 遵循 `pkg/httpwrap` 定义的统一格式：

```json
{
    "head": {
        "code": 0,
        "msg": "succ",
        "request_id": "2b08ace5-6419-4f69-a494-2bee0cee424b",
        "time": "2026-06-10T22:54:58.578339+08:00"
    },
    "body": { }
}
```

- 成功：`code=0, msg="succ"`
- 失败：错误码 < 0
- `request_id` 来自 `logging.GetTraceID(ctx)`
- `time` 为 RFC3339Nano 格式

详见 `pkg/httpwrap/http.go` (`ResponseInfo[T]`)、`pkg/httpwrap/ginresp.go` (`GinRespond`)。

---

## 错误处理

- 页面操作：校验 CID 存在性、页码合法性、源文件可读性
- 链接操作：校验主/备 CID 不重复、不允许自链接
- 删除操作：二次确认后执行，归档删除失败时回滚
- 已删除 comic：API 返回 `code=-3, msg="资源不存在"`

# COCOM 全周期发展路线图设计

> 设计于 2026-06-04，基于项目现有代码基底（Go 1.26 + Gin + MongoDB + Vue-like 前端）和近期 UI 迭代方向，结合用户重点关注领域（Web UI 用户体验、后端架构与性能、代码质量与测试）制定的 5 个里程碑路线图。

---

## 路线图总览

| 里程碑 | 主题 | 预计周期 |
|--------|------|----------|
| 🎯 M1 | 基础体验打磨 | 2-3 周 |
| 🔍 M2 | 搜索与标签体验升级 | 3-4 周 |
| 🖼️ M3 | 图片浏览与性能优化 | 3-4 周 |
| 🔧 M4 | 架构加固与代码健康 | 3-4 周 |
| 🚀 M5 | 扩展生态与高级功能 | 2-3 月 |

整体路线遵循「前端体验递进 → 还债加固 → 生态扩展」的节奏，每个里程碑内按 Web UI / 后端 / 代码质量 三个维度并行编排任务。

**不包含的范围：** 多用户管理与权限系统、国际化（i18n）、bookmark/收藏功能 — 已从路线图中移除。

---

## Milestone 1：基础体验打磨 🎯

**目标**：把近期新增的 UI 交互（Modal、Toast、LoadingManager、OptimisticUpdater、缩放控制、边栏）打磨到稳定可用状态，补齐缺失的基础体验。

| 维度 | 任务 | 具体说明 | 涉及文件 |
|------|------|----------|----------|
| **Web UI** | 移动端完美适配 | 左侧操作栏 → 底部栏、右侧缩放栏触屏交互优化、手机端缩略图点击区域放大、touch 事件处理 | `custom/css/styles.css`, `custom/js/scripts.js`, `gallery_detail.tpl` |
| **Web UI** | 移除导航大图模式按钮 | 删除导航栏上的大图模式切换入口，由左侧边栏统一管理 | `gallery_detail.tpl`, `head.tpl`, `navigation.tpl` |
| **Web UI** | 空状态/加载骨架屏 | 搜索无结果、加载中、网络错误时显示友好占位（骨架屏 Skeleton），避免白屏或闪动 | `custom/css/styles.css`, `custom/js/scripts.js`, `index.tpl`, `gallery_detail.tpl` |
| **Web UI** | 键盘快捷键统一 | 在已有 `/` `Esc` 基础上增加 `← →` 翻页（单页查看模式）、`L` 点赞快捷键 | `custom/js/scripts.js` |
| **后端** | 小版本 API 对齐 | 前端乐观更新依赖的 API（`/api/comic/tags/like`、`/api/comic/tags/update` 等）响应格式统一化，封装 JSON 结构 | `cmd/server/handler/`, `pkg/httpwrap/` |
| **代码质量** | JS 代码模块化 | 将 `scripts.js` (~1450 行) 拆分为独立模块文件，建立统一的错误处理模式 | `custom/js/scripts.js` → `custom/js/modules/*.js` |
| **代码质量** | CSS 变量提炼 | 提取主题色、间距、断点到 CSS 自定义属性，减少 magic number | `custom/css/styles.css` |

**验收标准：**
- 在 375px (iPhone SE) / 768px (iPad) / 1440px (Desktop) 三个视口下操作栏和缩放栏均可用
- 导航栏上不再有大图模式切换入口
- 所有 Toast 弹窗在 3s 后自动消失，无堆积遮挡
- 键盘 `← →` 在单页查看时触发翻页，`L` 触发点赞
- `scripts.js` 拆分为不超过 300 行/模块的文件

---

## Milestone 2：搜索与标签体验升级 🔍

**目标**：搜索功能从基础匹配变为生产力工具，标签管理从「能用」到「好用」。

| 维度 | 任务 | 具体说明 | 涉及文件 |
|------|------|----------|----------|
| **Web UI** | 搜索建议/自动补全 | 输入关键词时下拉展示匹配的标题和标签，支持键盘选中即时跳转；参考现有 `bindAutocompleteKeys` 扩展 | `custom/js/scripts.js`, `index.tpl`, `head.tpl` |
| **Web UI** | 搜索结果高亮 | 搜索列表页中匹配关键词的文字高亮显示（`<mark>` 标签 + 黄色底色） | `custom/css/styles.css`, `search_result.tpl`（或对应模板） |
| **Web UI** | 标签树/关系可视化 | 标签详情页支持查看相关标签链（父子关系、关联关系），层次化浏览标签体系 | `custom/js/scripts.js`, `custom/css/styles.css`, `tag_list_result.tpl` |
| **Web UI** | 标签对齐批量操作增强 | 支持选中多本漫画批量对齐标签；操作反馈纳入 LoadingManager | `custom/js/scripts.js`, `admin.tpl` |
| **后端** | 全文搜索索引 | 引入 MongoDB 文本索引（`$text` + `$search`），支撑快速模糊搜索 | `pkg/comic/storage/mongo.go`, `pkg/mongowrap/` |
| **后端** | 标签聚合 API 分页 | 大标签集（1000+ 条）的分页加载，避免 JSON 体量膨胀 | `cmd/server/api/`, `pkg/comic/` |
| **代码质量** | 搜索 API 端点测试 | 为 `/api/comic/tags/search`、`/v2/api/nhcomic/search` 等端点补充单元测试 | `pkg/comic/*_test.go` |

**验收标准：**
- 搜索框输入 2 个字符后 300ms 内展示下拉建议
- 搜索结果页中匹配文字有高亮标记
- 单个标签聚合 API 返回不超过 100 条/页，支持 `?page=2` 参数
- 搜索和标签 API 的测试覆盖率 ≥ 60%

---

## Milestone 3：图片浏览与性能优化 🖼️

**目标**：图片是核心资产，优化加载、缓存、浏览体验，让翻阅流畅无感知等待。

| 维度 | 任务 | 具体说明 | 涉及文件 |
|------|------|----------|----------|
| **Web UI** | 图片懒加载 + 虚拟滚动 | 详情页翻阅时按需加载图片（IntersectionObserver），而非一次性加载全部 URL；长列表支持虚拟滚动（仅渲染可见区域） | `custom/js/scripts.js` |
| **Web UI** | 大图模式翻页增强 | 大图模式下支持键盘 `← →` 翻页、触屏滑动翻页 | `custom/js/scripts.js` |
| **Web UI** | 缩略图渐进加载 | 先展示低质量模糊占位（LQIP — Low Quality Image Placeholder），逐步加载清晰图 | `custom/css/styles.css`, `custom/js/scripts.js`, `gallery_detail.tpl` |
| **后端** | 响应式图片服务 | 后端 API 支持 `?w=200&q=80` 参数动态生成指定尺寸的缩略图/预览图，避免前端硬放大图 | `cmd/server/`, `pkg/imaging/` |
| **后端** | 浏览器缓存策略 | 设置合理的 `Cache-Control`、`ETag`、`Last-Modified` 响应头，减少重复加载 | `cmd/server/`, `pkg/middlewares/` |
| **后端** | 图片预压缩 | 归档时或首次访问时自动生成多尺寸副本（thumbnail / preview / full），避免运行时实时转换 | `pkg/imaging/`, `pkg/archive/` |
| **代码质量** | 性能基准测试 | 建立页面 DOMContentLoaded 时间、API 响应时间的基准测试，防止回归 | `cmd/server/view/` 测试 |

**验收标准：**
- 进入详情页时仅加载首屏缩略图，滚动时按需加载后续
- 大图模式下键盘方向键可翻页
- 图片响应包含 `Cache-Control: public, max-age=31536000` 头
- 首次访问一张漫画详情页的网络请求数比优化前降低 ≥ 50%

---

## Milestone 4：架构加固与代码健康 🔧

**目标**：还技术债，填补测试空白，提升项目可持续维护性。

| 维度 | 任务 | 具体说明 | 涉及文件 |
|------|------|----------|----------|
| **后端** | 统一错误响应格式 | 所有 API 返回统一 JSON 错误格式 `{code: int, message: string, detail?: any}`，前端可统一处理 | `pkg/httpwrap/ginresp.go`, 各 handler |
| **后端** | API 版本管理规范化 | 梳理 `/v2/api/` 和 `/api/`（net/http 桥接）并存问题，明确弃用计划或统一路由策略 | `cmd/server/` |
| **后端** | 配置文档自动生成 | 从 Viper 键定义/结构体标签自动生成配置参考文档，避免 `docs/config.md` 过期 | `internal/config/`, `cmd/server/config.go`, `scripts/` |
| **代码质量** | 单元测试补全 | 补齐 `pkg/comic/`、`cmd/server/view/`、`pkg/middlewares/` 等未覆盖包的单元测试 | 各包 `*_test.go` |
| **代码质量** | Makefile 统一 | 确保三个子项目都提供 `make test` / `make lint` 命令，统一开发者体验 | 各项目 `Makefile` |
| **代码质量** | 过期 TODO 清理 | 审查 `docs/TODO.md` 中的条目，确认有效性/过时/重复，清理或迁移到新路线图 | `docs/TODO.md` |

**验收标准：**
- 所有 API 错误响应使用统一格式，前端不用再 `try-catch` 解析各种格式
- `docs/config.md` 与实际 Viper 键定义的差异 ≤ 5 项
- `make test` 通过率 100%，覆盖率 ≥ 40%
- `docs/TODO.md` 中无过期项目

---

## Milestone 5：扩展生态与高级功能 🚀

**目标**：让 cocom 成为更完整的漫画管理平台，打通三个子项目之间的集成。

| 维度 | 任务 | 具体说明 | 涉及文件 |
|------|------|----------|----------|
| **Web UI** | 管理面板仪表盘 | 统计数据总览（漫画总数、归档率、存储空间使用、验证状态等），可视化图表 | `custom/js/scripts.js`, `custom/css/styles.css`, `admin.tpl` |
| **后端** | 与 download-manager 集成 | cocom Web UI 中可触发 download-manager 下载任务，通过 SSE/WebSocket 推送进度到前端 | `cmd/server/`, `pkg/download/` |
| **后端** | 与 sproxy 集成 | 使用 sproxy 作为归档副本的外部传输通道（替代/补充百度网盘） | `pkg/archive/manager/` |
| **代码质量** | E2E 集成测试 | 模拟完整的「搜索→浏览→归档→恢复」全流程端到端测试 | `cmd/server/`, `test/` |
| **代码质量** | Docker Compose 全栈部署 | `docker-compose.yml` 一键启动 cocom + MongoDB，可选附带 download-manager + sproxy | `docker-compose.yml`, `Dockerfile` |

**验收标准：**
- 仪表盘页面可在 500ms 内加载完成基础统计
- download-manager 集成：UI 点击「下载」后任务创建成功，前端可见进度百分比
- sproxy 归档副本：通过 sclient 上传/下载存档文件测试通过
- `docker compose up` 在 30s 内启动全栈服务

---

## 交付物与变更文件清单（跨里程碑汇总）

| 文件路径 | 涉及里程碑 | 预计改动 |
|----------|-----------|----------|
| `custom/js/scripts.js` | M1, M2, M3 | 模块拆分、新增键盘快捷键、自动补全、懒加载、渐进加载 |
| `custom/css/styles.css` | M1, M2, M3 | CSS 变量提炼、骨架屏、移动端适配、渐进加载动画 |
| `cmd/server/view/static/tpl/*.tpl` | M1, M2 | 模板结构调整（导航栏、骨架屏占位、搜索增强） |
| `cmd/server/handler/*.go` | M1, M4 | API 响应格式统一、错误处理 |
| `pkg/httpwrap/ginresp.go` | M4 | 统一错误 JSON 格式 |
| `pkg/comic/storage/mongo.go` | M2 | MongoDB 文本索引 + 分页查询 |
| `pkg/imaging/` | M3 | 响应式缩放、预压缩 |
| `pkg/middlewares/` | M3 | 缓存策略调整 |
| `docs/config.md` | M4 | 配置文档自动生成流程 |
| `internal/config/` | M4 | 配置键结构体标签 |
| `Makefile`（各子项目） | M4 | `make test`/`make lint` 统一 |
| `docker-compose.yml` | M5 | 全栈编排 |
| `cmd/server/internal/` | M5 | download-manager 集成 |

---

## 规格自检

- [x] **占位符扫描**：无 TODO、待定、模糊需求
- [x] **内部一致性**：里程碑之间无矛盾，M1-M3 积累前端体验 → M4 还债 → M5 扩展递进合理
- [x] **范围检查**：聚焦 cocom 子项目，不涉及多用户管理和国际化，无溢出
- [x] **模糊性检查**：每个任务有「具体说明」和「验收标准」，可实现可验证
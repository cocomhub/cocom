# nhentai v2 详情页（HTML 预取脚本）解析 Spec

## Why
当前 v2 详情页实现直接请求 `/api/gallery/{id}`。在某些环境中该接口可能被阻断或遭到防护。实际页面（SvelteKit 渲染）内已通过 `data-sveltekit-fetched` 预取了与详情一致的 JSON 数据，可直接从详情页 HTML 中提取，降低对外部接口的依赖。

## What Changes
- 新增 HTML 版 v2 详情解析器：从 `script[type="application/json"][data-sveltekit-fetched][data-url^='/api/v2/galleries/{id}']` 的 `body` 字段提取详情 JSON。
- v2 详情页解析策略调整为“HTML 优先，API 兜底”：先尝试 HTML 预取脚本解析，失败则回退到 `/api/gallery/{id}`。
- 增加基于样例文件 `/pkg/comic/probe/640503.html` 的单测，确保字段映射兼容现有 `saveComicInfo` 与 `genDownList`。
- 保持 v1 与现有外部接口不变。

## Impact
- 受影响能力：
  - v2 详情页信息解析（数据来源策略）
- 受影响代码：
  - pkg/comic/probe/probe.go（新增 HTML 解析函数与选择逻辑）
  - pkg/comic/probe/v2_detail_html_test.go（新单测）

## ADDED Requirements
### Requirement: v2 详情页 HTML 解析
系统应支持从详情页 HTML 的 SvelteKit 预取脚本中解析出完整的详情信息。

#### Scenario: HTML 存在且可解析
- WHEN 加载详情页 HTML
- THEN 能找到 `data-sveltekit-fetched` 且 `data-url` 形如 `/api/v2/galleries/{id}?...` 的脚本
- AND 先 JSON 解码脚本，再对 `body` 做二次 JSON 解码
- AND 获得包含 `id`、`media_id`、`pages`、`title`、`num_pages`、`num_favorites`、`tags` 等字段的结构
- AND 转成现有存储结构（补充 `comic_id`、`comic_url`），与 `saveComicInfo`/`genDownList` 兼容

#### Scenario: HTML 缺失或不含预取脚本
- WHEN 未能从 HTML 中解析到有效脚本
- THEN 回退到调用 `/api/gallery/{id}` 获取详情

## MODIFIED Requirements
### Requirement: v2 详情解析策略
v2 详情解析应采用“HTML 优先，API 兜底”的策略，以提升稳定性与可用性。

## REMOVED Requirements
（无）


# nhentai 抓取 v2（SvelteKit 渲染）Spec

## Why
nhentai 首页与详情页已切换为 SvelteKit 渲染，现有 v1 通过 DOM/`window._gallery` 的解析方式在部分页面失效，需要新增 v2 提取路径以适配新版渲染，同时保留 v1 以便回退。

## What Changes
- 在 pkg/comic/probe 增加 v2 提取实现，基于 index.html 中的 `data-sveltekit-fetched` 脚本数据解析列表；详情页通过官方 JSON 接口获取并映射为现有结构。
- 提供模式切换参数：`--nhentai_mode=[v1|v2]`，默认 v1，不产生破坏性变更。
- 重构探测流程为策略选择：在探测任务中根据模式选择 v1/v2 逻辑，其他接口（保存、生成下载清单、上传）保持不变。
- 增加基础单元测试：使用仓库内样例文件 `pkg/comic/probe/index.html` 验证 v2 列表解析的稳定性。
- 记录必要日志，便于线上观测。

## Impact
- 受影响能力：
  - nhentai 列表页 ID 提取
  - nhentai 详情页信息解析
- 受影响代码：
  - pkg/comic/probe/probe.go（新增模式选择与 v2 入口）
  - 可能新增：pkg/comic/probe/v2.go（v2 列表与详情实现）、pkg/comic/probe/v2_test.go（测试）

## ADDED Requirements
### Requirement: 新增 v2 提取
系统应提供基于 SvelteKit 数据脚本与官方 JSON 的 nhentai v2 提取实现。

#### Scenario: 列表页解析成功
- WHEN 执行 v2 模式列表页解析
- THEN 能从 `data-sveltekit-fetched` 中匹配到 `data-url="/api/v2/galleries?page=N"` 的脚本
- AND 将脚本 JSON 中 `body.result[].id` 提取为漫画 ID
- AND 依据 `tag_ids` 过滤（保留包含 6346 或 29963 的条目）并按 `lastComic` 规则截断

#### Scenario: 详情页解析成功
- WHEN 执行 v2 模式详情页解析
- THEN 通过 JSON 接口获取到包含 `media_id`、`images.pages`、标题等字段的结构
- AND 映射为现有 `saveComicInfo` 与 `genDownList` 可用的数据结构

#### Scenario: 模式切换
- WHEN 启动探测任务
- THEN 可通过 `--nhentai_mode` 指定 v1 或 v2
- AND 默认 v1，不影响现有部署

## MODIFIED Requirements
### Requirement: 探测任务可配置
探测任务应提供配置项以切换不同提取策略，且切换不影响其他系统接口（保存、下载清单、上传）。

## REMOVED Requirements
（无）


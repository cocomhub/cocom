# nhentai v2 解析结果归一到 v1 结构 Spec

## Why
当前 v2 详情解析返回的结构与 v1 存在差异（例如 v2 的 pages 含有 path/width/height，而 v1 采用 images.pages 下仅包含 t/w/h）。导致下游依赖 v1 结构的能力（如 PageOriginUrlByIndex、下载清单生成等）存在不一致风险。需要将 v2 解析结果在入库/后处理阶段归一为 v1 结构。

## What Changes
- 新增归一化函数：将 v2 解析出的 gallery map 转换为 v1 等价结构（仅包含 v1 需要的字段形态）。
- 在 v2 解析路径（HTML 与 API 两种来源）完成获取后，统一调用归一化函数输出与 v1 一致的 comicInfo。
- 基于样例文件对比参考：
  - v1 样例：/pkg/comic/probe/comicInfo.v1.json
  - v2 样例：/pkg/comic/probe/comicInfo.v2.json
- 增加单测：验证归一后的 images.pages 元素以 t/w/h 形式存在，cover/thumbnail 为 t/w/h，去除 v2 的 path 字段对下游无影响。
- 不改动 v1 路径。

## Impact
- 受影响能力：
  - v2 详情解析产物形态
  - 依赖 images.Pages.PicType 的 URL 拼接逻辑
- 受影响代码：
  - pkg/comic/probe/probe.go（实现归一化并在 v2 分支调用）
  - pkg/comic/probe/normalize_v2_to_v1_test.go（单测）

## ADDED Requirements
### Requirement: v2→v1 归一
系统应将 v2 解析得到的详情数据归一为与 v1 相同的结构。

#### Scenario: Pages 映射
- WHEN v2 pages 含有 path 和 width/height
- THEN 归一化生成 images.pages 列表，元素仅含 `t`、`w`、`h`
- AND `t` 由 path 后缀（jpg/png/webp/gif）映射到 j/p/w/g
- AND 维持页序与数量一致

#### Scenario: Cover/Thumbnail 映射
- WHEN v2 含有 cover/thumbnail 的 path/width/height
- THEN 归一化生成 images.cover/images.thumbnail 仅含 `t`、`w`、`h`
- AND `t` 由 path 后缀映射

#### Scenario: 其他字段
- WHEN 归一化
- THEN 保留 media_id、num_pages、num_favorites、title、tags、comic_id、comic_url 等字段
- AND 归一后可成功反序列化为 api.ComicInfo，且 PageOriginUrlByIndex 正常工作

## MODIFIED Requirements
### Requirement: v2 解析输出
v2 详情解析输出的结构需与 v1 对齐，以保证下游逻辑一致性。

## REMOVED Requirements
（无）


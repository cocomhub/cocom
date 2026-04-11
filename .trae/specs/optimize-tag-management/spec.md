# 优化 Tag 管理 Spec

## Why
当前标签统计与展示分散：tag count 来自 comicInfo 的静态值，无法实时更新；缺少针对 tag 的“喜欢”标记与统一聚合。需要统一聚合与缓存，提升页面展示与交互一致性。

## What Changes
- 新增聚合接口：聚合所有漫画的标签统计，写入 comicTag 集合，供各页面读取，参考 AggregateTagSectionIndices 的分段聚合思路。
- TagListResultPage 直接使用聚合数据，避免实时查询 comicInfo 统计。
- TagResultPage 增加“喜欢/取消喜欢”按钮，行为参考 GalleryDetail 的 like 按钮，在 comicTag 中记录该 tag 的喜欢状态。
- 新增样式：标记为“喜欢”的 tag 显示红色（新增 CSS 类名 tag-like）。
- 展示层从 comicTag 读取最新聚合数据（含 count 与 like），不再依赖 comicInfo 中静态的 tag count；为 comicTag 读取增加缓存以减少数据库压力。

## Impact
- Affected specs: 标签聚合与展示、TagResultPage 交互、TagListResultPage 展示
- Affected code: 后端聚合逻辑与 API、Mongo 集合（新增 comicTag）、缓存键与读写、视图/模板与 CSS

## ADDED Requirements
### Requirement: 标签聚合接口
系统 SHALL 提供聚合标签统计的接口：
- 从 comicInfo 集合聚合标签统计信息（按 type/id/name/url 维度）
- 将聚合结果写入 comicTag 集合，结构包含：type、id、name、url、count、like、updated_at
- 提供读取接口（按 tagType/分页/排序），并对读取结果做缓存

#### Scenario: 成功聚合
- WHEN 管理端触发聚合或定时任务运行
- THEN comicTag 集合更新为最新统计数据，后续页面从该集合读取最新 count 与 like 状态

### Requirement: TagResultPage 喜欢标记
系统 SHALL 在 TagResultPage 增加喜欢/取消喜欢按钮（侧边栏形式）：
- 点击喜欢：将该 tag 的 like 标记设为 true（写入 comicTag）
- 再次点击取消：将该 tag 的 like 标记设为 false
- 页面按钮样式参考 GalleryDetail 的 like 交互（btn-secondary/btn-primary 切换）

### Requirement: 喜欢样式
系统 SHALL 使用红色样式显示被标记喜欢的标签：
- 新增 CSS 类名 tag-like
- 在 TagResultPage、GalleryDetail 及相关标签列表中，当 tag.like 为 true 时，应用 tag-like

### Requirement: 展示读取 comicTag
系统 SHALL 从 comicTag 读取最新聚合数据：
- 页面展示的标签 count 与 like 状态来源于 comicTag
- 针对 comicTag 读取增加缓存，避免高频 DB 访问

## MODIFIED Requirements
### Requirement: 标签展示数据源
现有从 comicInfo 读取的 tag count 修改为从 comicTag 读取：
- 页面数据层替换数据源，优先命中缓存，未命中时读取 comicTag 并回填缓存

## REMOVED Requirements
（无）

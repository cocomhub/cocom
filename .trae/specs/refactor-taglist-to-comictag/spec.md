# 重构 TagListPage 至 comicTag Spec

## Why
现有 TagListPage 仍依赖 comicInfo 进行分段与列表聚合，导致数据一致性与性能问题。需要统一使用物化的 comicTag 集合来实现分段索引与列表聚合，同时优化缓存与支持喜欢筛选。

## What Changes
- 使用 comicTag 集合实现 AggregateTagSectionIndices 与 AggregateTagList，替换原有基于 comicInfo 的实现。
- 调整缓存策略：按单个 ID 缓存 tag 数据，避免批量 ids 作为键导致命中率低。
- 支持筛选已喜欢标签（like=true），用于 TagListPage 的列表与分段索引。

## Impact
- Affected specs: TagListPage 行为（分段索引、标签列表、筛选）
- Affected code: internal/tag 聚合与读取、view/tag_list_result.go、gallery_detail.go 读取计数、缓存键设计、API /api/comic/tags 支持筛选

## ADDED Requirements
### Requirement: 使用 comicTag 实现分段与列表
系统 SHALL 基于 comicTag 提供：
- AggregateTagSectionIndices：返回标签分段索引（例如按首字母/分区），支持 like 筛选
- AggregateTagList：返回标签列表，支持类型、分页、排序与 like 筛选

#### Scenario: 成功替换
- WHEN 用户访问 TagListPage
- THEN 页面数据（分段与列表）均来源于 comicTag，展示与聚合一致

### Requirement: 单 ID 缓存
系统 SHALL 针对单个 tag id 做缓存：
- 提供 GetTagByID(ctx,type,id) 使用缓存键 comicTag:id:{type}:{id}
- 在需要批量读取时循环读取并聚合结果，提升缓存命中率

### Requirement: 喜欢筛选
系统 SHALL 支持筛选已喜欢的标签：
- TagListPage 支持 liked=true 过滤
- /api/comic/tags 支持 liked=true 过滤

## MODIFIED Requirements
### Requirement: TagListPage 数据源
TagListPage 的 AggregateTagSectionIndices 与 AggregateTagList 修改为从 comicTag 读取：
- 分段与列表均支持 liked=true 过滤参数

## REMOVED Requirements
（无）

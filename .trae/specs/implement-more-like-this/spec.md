# 实现 More Like This 推荐 Spec

## Why
当前 More Like This 仅返回占位数据，无法提供有效推荐。基于当前漫画的标签进行筛选，随机给出 5 条推荐，提升浏览体验。

## What Changes
- 在 GalleryDetail 页面实现推荐列表：依据当前 comic 的 tags 作为筛选条件，随机选择 5 个具有任意相同标签的漫画，排除当前漫画。
- 后端新增按标签筛选并随机返回的查询能力（或以较大集合查询后本地随机）。
- 视图层使用真实数据替换占位实现。

## Impact
- Affected specs: 推荐展示（More Like This）
- Affected code: GalleryDetail 视图逻辑与数据获取、Mongo 查询构建（comicInfo 集合）、可选后端 API/方法

## ADDED Requirements
### Requirement: 标签交集推荐
系统 SHALL 提供基于当前漫画标签的推荐：
- 从 comicInfo 中筛选与当前漫画 tags 有交集的漫画（至少一个标签匹配）
- 排除当前漫画 cid
- 随机选取 5 条返回

#### Scenario: 成功推荐
- WHEN 用户打开带有标签的漫画详情页
- THEN 系统展示 5 条“相似漫画”推荐，均与当前漫画至少一个标签相同

## MODIFIED Requirements
### Requirement: More Like This 展示
现有占位实现修改为真实推荐数据：
- 页面中的 More Like This 使用实际查询结果，不再重复当前条目

## REMOVED Requirements
（无）

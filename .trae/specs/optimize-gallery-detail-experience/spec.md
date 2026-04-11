# 优化 GalleryDetail 页面体验 Spec

## Why
当前“点赞”交互不直观且与标签系统割裂。将点赞统一到标签体系，并优化按钮交互，提高一致性与可维护性。

## What Changes
- 优化 Like 按钮交互：默认 btn-secondary，点击后变为 btn-primary；支持再次点击切换回 btn-secondary（即点赞/取消点赞的切换）。
- 新增标签类型 TagsType=Customs，对应后端标签数据 type="custom"；将 Like 信息保存为 tag，结构为：
  - { "type": "custom", "url": "/custom/like/", "count": 1, "id": 99999, "name": "like" }
- 新增迁移工具：将旧的 Like 信息迁移为上述 tag 数据结构，并计入对应 gallery。
- 后端提供 /custom/like/ 接口（或复用现有接口）支持点赞计数与切换逻辑。
- GalleryDetail 页面展示与数据加载支持 custom 标签类型，并能正确显示/切换 Like 状态与计数。

## Impact
- Affected specs: GalleryDetail 页面点赞交互、标签系统（新增 custom 类型）、数据迁移工具
- Affected code: 视图与模板（GalleryDetail）、标签类型枚举与映射、后端 API（/custom/like/）、数据模型与存储、迁移脚本/工具

## ADDED Requirements
### Requirement: Like 交互切换
系统 SHALL 在 GalleryDetail 页面上提供点赞按钮交互：
- 默认样式为 btn-secondary
- 点击后变为 btn-primary，并记录点赞为 tag（type="custom", name="like"）
- 再次点击变回 btn-secondary，并撤销/减少对应点赞记录

#### Scenario: 成功切换
- WHEN 用户第一次点击 Like
- THEN 按钮样式切换为 btn-primary，创建/增加对应 custom/like 标签计数
- WHEN 用户再次点击 Like
- THEN 按钮样式切换为 btn-secondary，减少/撤销 custom/like 标签计数

### Requirement: 新增 TagsType=Customs
系统 SHALL 新增标签类型 Customs：
- 前端展示与后端映射均支持 type="custom"
- GalleryDetail 的标签列表与统计应包含 custom 类型标签

### Requirement: /api/tags/ 接口
系统 SHALL 提供RestFul接口处理标签操作：
- 路径建议为 /api/tags/，通过 POST 方法创建/增加标签，DELETE 方法撤销/减少标签，标签数据结构为：
  - like tag：{ "type": "custom", "url": "/custom/like/", "count": 1, "id": 99999, "name": "like" }
- 注意后续期望扩展成操作其他类型标签（如添加作者、自定义标签等）
- 支持创建、增加、撤销/减少 custom/like 标签
- 返回最新计数与当前用户 Like 状态

### Requirement: 迁移工具
系统 SHALL 提供迁移工具，将旧 Like 信息迁移为 custom/like 标签：
- 对历史数据进行扫描，将点赞记录写入对应 gallery 的 custom/like 标签
- 记录迁移结果与可能的异常项，支持幂等重试

## MODIFIED Requirements
### Requirement: 现有 Like 功能
现有 Like 功能修改为通过标签系统实现与存储：
- 数据层不再以独立 Like 结构保存，而是以 type="custom", name="like" 的标签记录保存
- 前端根据标签状态与计数渲染按钮样式与数值

## REMOVED Requirements
（无）

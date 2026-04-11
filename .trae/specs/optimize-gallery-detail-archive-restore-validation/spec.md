# GalleryDetailPage 归档/恢复按钮校验与交互一致性 Spec

## Why
归档/恢复操作直接影响存档准确性与可恢复性。需要在 GalleryDetail 页面强化前置校验、提供异常可视化与强制策略，并统一服务端/前端交互格式，提高数据管理准确性与可用性。

## What Changes
- 服务端与前端统一采用 httpwrap.ResponseInfo 格式交互；异常通知消息在 UI 以「[$code] $msg」展示
- 归档操作新增「归档前校验」：检测当前漫画所有图片均有效
  - 全部有效：继续归档
  - 存在异常：返回错误码与异常图片列表，界面高亮标注并引导修复；新增「强制归档」能力
- 强制归档：在存在异常时允许继续归档，并将存档标记为 by_force=true
- 恢复操作新增「MD5 校验」：存档文件 MD5 与记录不一致时，阻断恢复并返回详细信息（参考归档异常处理的交互）

## Impact
- Affected specs:
  - add-gallery-detail-sidebar-actions：按钮与通知规范被加强
- Affected code:
  - 后端：/v2/api/nhcomic/:cid/archive 与 /:cid/restore handler 与 service；归档元数据（ArchiveInfo.has_valid）落库逻辑
  - 前端：GalleryDetail 页面按钮交互、错误提示、图片高亮、强制归档入口与引导文案

## ADDED Requirements
### Requirement: 统一 Response 格式与异常提示
系统 SHALL 在服务端与前端严格使用 httpwrap.ResponseInfo 格式：
- 成功返回：head.code=0，head.msg="succ"，body 为业务数据
- 失败返回：head.code!=0，head.msg 为错误描述
- UI 异常提示 SHALL 采用「[$code] $msg」格式 Toast/Message 展示

#### Scenario: 失败消息格式
- WHEN 服务端返回 head.code=-1001, head.msg="验证失败"
- THEN UI 显示「[-1001] 验证失败」

### Requirement: 归档前校验与强制归档
系统 SHALL 在归档前检查当前漫画所有图片有效性：
- 全部有效：继续归档并成功返回
- 存在异常：返回 head.code=-1001，head.msg="验证失败"，body.invalid_images 为异常图片清单
- UI SHALL：
  - 自动为异常图片加红框高亮
  - 提供引导操作「修复漫画状态」（调用现有验证/修复接口）
  - 显示「强制归档」按钮；点击后触发强制归档
- 强制归档 SHALL：继续归档流程，并将 ArchiveInfo.has_valid 记录为 false

字段约定：
- body.invalid_images: [{ index:number, path:string, reason:string }]（最小集即可，index 从 1 起对应页面序号）
- 强制归档触发方式：POST /v2/api/nhcomic/:cid/archive?force=true

#### Scenario: 校验通过归档
- WHEN 所有图片有效且触发归档
- THEN 返回 head.code=0，归档完成；ArchiveInfo.by_force=false

#### Scenario: 校验失败并展示异常
- WHEN 存在异常图片且触发归档
- THEN 返回 head.code=-1001，body.invalid_images 含异常清单；UI 高亮对应图片并展示引导与「强制归档」

#### Scenario: 强制归档
- WHEN 用户点击「强制归档」
- THEN 服务端执行归档并落库 by_force=true；返回 head.code=0

### Requirement: 恢复前 MD5 校验
系统 SHALL 在恢复前校验存档文件 MD5 是否与存档记录一致：
- 一致：继续恢复并返回成功
- 不一致：返回 head.code=-2001，head.msg="存档文件校验失败"，body 含 { expected_md5, actual_md5 }
- UI SHALL 参考归档异常处理显示错误提示与引导（可提示重新归档/修复）

#### Scenario: 校验不一致
- WHEN 恢复前 MD5 不匹配
- THEN 返回 head.code=-2001，UI 显示「[-2001] 存档文件校验失败」并提供修复引导

## MODIFIED Requirements
### Requirement: GalleryDetail 侧栏归档/恢复按钮
- 按钮交互 SHALL 基于 httpwrap.ResponseInfo 解析结果
- 失败时采用「[$code] $msg」格式提示
- 归档流程前置校验与异常反馈、强制归档入口按本 Spec 执行

## REMOVED Requirements
无


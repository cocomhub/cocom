# GalleryDetail 页面 Sidebar 控制按钮与通用弹出消息 Spec

## Why
为 GalleryDetail 页面提供就地的“归档/恢复”和“修复漫画状态”操作，并统一弹出消息反馈，降低操作成本、提升可见性与一致性。

## What Changes
- 前端：在 GalleryDetail 页面侧栏新增“控制”分组
  - 基于归档状态显示单个切换按钮：未归档显示“归档”，已归档显示“恢复”
  - 新增“修复漫画状态”按钮，调用现有检查修复接口
  - 引入可复用的弹出消息（Toast）能力，支持默认行为与参数覆盖
- 后端：提供漫画归档与恢复的同步接口，返回操作结果与归档状态；继续使用既有“检查并自动修复”接口

## Impact
- Affected specs: GalleryDetail 交互、通知反馈规范
- Affected code: 
  - 前端：GalleryDetail 页面/侧栏组件、Toast 组件或服务
  - 后端：nhcomic 路由与 handler、归档字段持久化与服务层

## ADDED Requirements
### Requirement: 归档/恢复控制按钮
系统 SHALL 在 GalleryDetail 页面侧栏提供基于归档状态显示的单按钮控制：
- 未归档时展示“归档”按钮；已归档时展示“恢复”按钮（两者互斥，仅显示其一）
- 点击“归档”时调用 POST /v2/api/nhcomic/:id/archive；点击“恢复”时调用 POST /v2/api/nhcomic/:id/restore
- 接口为同步语义，返回 JSON：{ success: bool, archived: bool, message: string }
- 成功时显示绿色 Toast；失败时显示红色 Toast
- “恢复”成功后自动刷新当前页面；“归档”成功不强制刷新（保留当前视图即可）

#### Scenario: 成功归档
- WHEN 用户点击“归档”
- THEN 后端返回 {success:true, archived:true}，页面显示绿色 Toast“已归档”

#### Scenario: 成功恢复并刷新
- WHEN 用户点击“恢复”
- THEN 后端返回 {success:true, archived:false}，页面显示绿色 Toast“已恢复”，随后自动刷新页面

#### Scenario: 操作失败
- WHEN 后端返回 success=false 或发生异常
- THEN 页面显示红色 Toast，内容为 message 或统一错误文案

### Requirement: 修复漫画状态按钮
系统 SHALL 在侧栏提供“修复漫画状态”按钮：
- 点击调用现有检查并自动修复接口 POST /v2/api/nhcomic/verify，携带当前漫画标识
- 接口为同步语义，返回 JSON：{ success: bool, fixed: bool, message: string }
- 成功或完成修复时显示绿色 Toast；失败显示红色 Toast；无需自动刷新页面

#### Scenario: 存在异常且已修复
- WHEN 用户点击“修复漫画状态”
- THEN 后端完成检查并修复并返回 {success:true, fixed:true}，页面显示绿色 Toast“修复完成”

### Requirement: 通用弹出消息（Toast）
系统 SHALL 提供可复用的 Toast 能力：
- 默认行为：5 秒后自动消失；用户点击即可立刻消失
- 颜色规范：成功为绿色，异常为红色，普通消息为白色
- 支持参数覆盖：{ type: 'success'|'error'|'info', duration: number=5000, dismissible: boolean=true, onClose?: fn }
- 支持同时显示多条（队列或堆叠），具备可访问性（role="alert"）

#### Scenario: 参数覆盖
- WHEN 调用 showToast("已归档", { duration: 10000, type:'success' })
- THEN Toast 显示为绿色并在 10 秒后自动消失，可手动点击提前关闭

## MODIFIED Requirements
（无）

## REMOVED Requirements
（无）


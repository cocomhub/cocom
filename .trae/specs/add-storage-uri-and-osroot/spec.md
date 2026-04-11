# Storage 唯一 URI 与 LocalFS 根目录沙箱化 Spec

## Why
现有 Storage 抽象尚未提供对象 Key 的唯一标识（URI）规范，不利于跨后端统一引用与日志追踪。同时 LocalFS 采用路径拼接，存在被错误使用而导致“目录遍历”风险的空间。需要提供规范化的 URI 能力，并利用 Go 1.24 的 os.Root 将本地文件系统操作限制在指定根目录内以提升安全性。

## What Changes
- 新增 Storage 接口方法：返回对象 Key 的唯一标识路径（URI），格式为 storageType://storageName/storagePath
- 约定 URI 规范：storageType 为实现类型（如 localfs、s3 等），storageName 为实例名（来自配置），storagePath 为实例根目录下的相对路径
- LocalFS 引入基于 Go 1.24 os.Root 的根目录沙箱化，确保所有文件操作被限制在配置的根目录下，防御目录遍历和越权访问
- 更新/新增单元测试：URI 生成、沙箱化下的正常与越界路径用例
- 更新 go.mod 至 Go 1.24（如当前版本低于 1.24）
- 更新文档：记录 URI 规范与安全注意事项
- **BREAKING**：Storage 接口新增方法，对第三方自定义驱动需要补充实现

## Impact
- Affected specs: 存储统一标识、LocalFS 安全沙箱
- Affected code:
  - pkg/storage（接口与类型定义）
  - pkg/storage/localfs（驱动实现与路径解析）
  - 可能引用对象唯一标识的模块（如 pkg/archive 管理日志/定位器）
  - go.mod 最低版本提升至 1.24

## ADDED Requirements
### Requirement: 唯一 URI 规范与接口
系统 SHALL 为任意存储对象提供唯一 URI：storageType://storageName/storagePath，以便跨后端统一引用与追踪。

#### Scenario: 成功生成 URI
- WHEN 业务向 Storage 请求某 Key 的唯一标识
- THEN 返回符合规范的 URI，包含实现类型、实例名与根目录下相对路径

### Requirement: LocalFS 根目录沙箱化
系统 SHALL 在 LocalFS 内使用 Go 1.24 的 os.Root 将所有文件操作限制在配置的根目录内，阻止目录遍历与越权访问。

#### Scenario: 阻止越界访问
- WHEN 传入包含 ../ 或符号链接指向根目录外的路径
- THEN 操作被拒绝并返回安全错误，不会访问根目录之外的文件

## MODIFIED Requirements
### Requirement: 存储实现需补充 URI 方法
所有 Storage 实现（含 LocalFS）SHALL 实现新增的唯一 URI 方法；LocalFS 需基于其根目录计算 storagePath 并返回正确 URI。

## REMOVED Requirements
### Requirement: 无
**Reason**: 在既有能力基础上增强标识与安全性，无需移除特性  
**Migration**: 第三方 Storage 实现需新增 URI 方法实现；LocalFS 用户无需改动配置，默认保持现有根目录含义并启用沙箱化

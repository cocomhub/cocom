# 存档功能优化与 Storage 抽象 Spec

## Why
当前 pkg/archive 仅支持简单的本地打包与文件操作，难以适配多种后端存储（本地、网盘、对象存储等），也不支持在不同存储实现间迁移。需要抽象统一的存储接口，提升可扩展性与可维护性。

## What Changes
- 新增 pkg/storage 包，抽象统一文件操作接口（上传、下载、查询、列举、删除、复制/移动、元数据管理）
- 默认实现 LocalFS（本地文件系统），后续可扩展：百度网盘、Alist、S3、GCS 等
- 在 pkg/archive 中使用 storage 抽象管理存档文件与相关信息，替代直接本地 IO
- 增加跨存储迁移能力：在不同 storage 实现之间迁移文件及其元数据
- 增加可配置的存储后端选择与凭据管理（不暴露敏感信息）
- 提供基础校验（存在性、完整性校验、尺寸与哈希比对）与错误处理策略
- 提供最小化文档与示例，明确如何新增一个 storage 驱动

## Impact
- Affected specs: 文件存储抽象、存档管理、迁移能力
- Affected code:
  - 主要影响 [archiver.go](file:///d:/workdir/leon/cocom/pkg/archive/archiver.go)
  - 新增 [pkg/storage](file:///d:/workdir/leon/cocom/pkg/storage/) 目录与默认实现
  - 配置读取处（例如初始化组件的入口）需要支持选择 storage 实现与参数

## ADDED Requirements
### Requirement: 新增统一存储抽象
系统 SHALL 提供统一的存储接口与本地默认实现，支持上传、下载、查询、列举、删除、复制/移动、元数据读取。

#### Scenario: 成功上传与下载
- WHEN 用户通过 archive 保存产物
- THEN 系统经由 storage 接口写入文件并返回对象标识（URI 或 Key）
- WHEN 用户请求下载该产物
- THEN 系统经由 storage 接口读取文件并正确返回流或字节数据

### Requirement: 支持跨存储迁移
系统 SHALL 支持在不同 storage 实现间迁移存档文件与元数据，并进行完整性校验与失败重试。

#### Scenario: 成功迁移
- WHEN 管理员发起从 LocalFS → 其他后端的迁移
- THEN 系统遍历需要迁移的对象，执行复制与校验，记录迁移状态，并在失败时可重试/断点续传（最小实现可先提供幂等重试与失败清单）

## MODIFIED Requirements
### Requirement: 存档功能使用 storage 抽象
现有 pkg/archive 的文件写入/读取 SHALL 通过 storage 接口完成，不再直接依赖本地文件系统路径。配置项新增“存储后端”与对应参数。

## REMOVED Requirements
### Requirement: 无
**Reason**: 不移除现有能力，仅将存档读写路径抽象化并可配置  
**Migration**: 默认继续使用 LocalFS，无需迁移；如需更换后端，通过迁移工具/函数完成数据移动

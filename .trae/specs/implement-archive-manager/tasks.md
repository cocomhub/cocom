# Tasks
- [x] 任务 1：包结构与接口设计（pkg/archive/manager）
  - [x] 子任务 1.1：定义 Manager 接口（Register/Put/Get/Find/Delete/List/Replicate/Check/ApplyPolicy）
  - [x] 子任务 1.2：定义错误类型与上下文（NotFound/Conflict/Transient/PolicyViolation）
  - [x] 子任务 1.3：定义选项与配置（Option/Config）

- [x] 任务 2：元数据模型与序列化
  - [x] 子任务 2.1：定义 ArchiveMeta、Checksum、StorageLocator、ReplicaHealth 等结构
  - [x] 子任务 2.2：实现 JSON 编解码与版本化字段（兼容后续演进）

- [x] 任务 3：元数据持久化（IndexStore）
  - [x] 子任务 3.1：基于 Storage 抽象实现 IndexStore（默认本地 LocalFS）
  - [x] 子任务 3.2：提供 CRUD：Create/Read/Update/Delete 与分页 List/Filter
  - [x] 子任务 3.3：确保写入幂等与基本并发安全（文件级锁/临时文件写入后原子替换）

- [x] 任务 4：与打包流程集成（pkg/archive）
  - [x] 子任务 4.1：在 archiver 打包成功后调用 manager.Register/Put 更新元数据
  - [x] 子任务 4.2：提供读取接口以便上层按 ID/名称检索并访问存档内容

- [x] 任务 5：一致性检查（Checker）
  - [x] 子任务 5.1：对各 StorageLocator 执行 Stat/ETag/可选 Hash 校验
  - [x] 子任务 5.2：生成检查报告并更新副本健康状态与最后校验时间
  - [x] 子任务 5.3：处理不一致标记与最小修复建议（仅报告，不自动修复）

- [x] 任务 6：多存储备份与迁移（Replicate）
  - [x] 子任务 6.1：调用 [pkg/storage/migrate.go](file:///d:/workdir/leon/cocom/pkg/storage/migrate.go) 执行复制与校验
  - [x] 子任务 6.2：在成功后更新元数据位置列表与健康状态；失败生成重试清单

- [x] 任务 7：保留策略（Retention）
  - [x] 子任务 7.1：定义策略配置：时间保留/数量上限/空间配额/位置偏好
  - [x] 子任务 7.2：实现策略执行器：按配置选择要清理的副本并调用 storage 删除
  - [x] 子任务 7.3：实现“远端备份完成后自动清理本地副本”的场景

- [x] 任务 8：配置与文档
  - [x] 子任务 8.1：新增 manager 配置结构与 Option 模式
  - [x] 子任务 8.2：撰写 README/用例：如何注册存档、检索、备份与清理

- [x] 任务 9：测试与验证
  - [x] 子任务 9.1：元数据模型与 IndexStore 的单元测试（CRUD/并发最小验证）
  - [x] 子任务 9.2：一致性检查与报告的单元测试（含不一致用例）
  - [x] 子任务 9.3：Replicate 的集成测试（LocalFS→LocalFS）
  - [x] 子任务 9.4：保留策略的单元测试（时间/数量/空间/位置）

# Task Dependencies
- [任务 4] 依赖 [任务 1] 与 [任务 2] 与 [任务 3]
- [任务 5] 依赖 [任务 1] 与 [任务 2]
- [任务 6] 依赖 [任务 1] 与 [任务 2]
- [任务 7] 依赖 [任务 1] 与 [任务 2] 与 [任务 3]
- [任务 9] 依赖前述所有任务

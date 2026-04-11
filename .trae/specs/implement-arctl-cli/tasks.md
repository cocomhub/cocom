# Tasks
- [x] 任务 1：CLI 框架与入口
  - [x] 子任务 1.1：创建 cmd/arctl/main.go，解析全局参数（--config、--output、--verbose）
  - [x] 子任务 1.2：定义子命令骨架：pack、unpack、query、backup、check
  - [x] 子任务 1.3：统一错误处理与退出码，支持人类可读与 JSON 两种输出

- [x] 任务 2：打包并注册（pack）
  - [x] 子任务 2.1：加载配置，构造 archive.Config 与 manager 句柄
  - [x] 子任务 2.2：调用 helper.ArchiveAndRegister 执行打包注册
  - [x] 子任务 2.3：输出归档 ID、位置与校验摘要（支持 --output=json）

- [x] 任务 3：解包（unpack）
  - [x] 子任务 3.1：解析输入归档路径或通过 ID 解析到主位置
  - [x] 子任务 3.2：执行解包到目标目录，校验哈希/大小
  - [x] 子任务 3.3：输出完成摘要

- [x] 任务 4：查询（query）
  - [x] 子任务 4.1：支持按 ID/名称/标签查询 IndexStore
  - [x] 子任务 4.2：输出基本信息、位置列表与健康状态（支持分页/limit）
  - [x] 子任务 4.3：支持 --output=json

- [x] 任务 5：多存储备份（backup）
  - [x] 子任务 5.1：根据目标端点解析 Storage（基础版实现 LocalFS→LocalFS）
  - [x] 子任务 5.2：复用 pkg/storage 的复制/校验能力（参考 migrate.go）
  - [x] 子任务 5.3：成功后更新元数据位置列表与健康状态

- [x] 任务 6：一致性检查（check）
  - [x] 子任务 6.1：对各副本执行 Stat/ETag/可选 Hash 校验
  - [x] 子任务 6.2：生成检查报告，支持 --output=json
  - [x] 子任务 6.3：可选更新最后校验时间

- [x] 任务 7：配置与文档
  - [x] 子任务 7.1：定义 arctl 配置结构（存储端点、默认策略、输出模式）
  - [x] 子任务 7.2：撰写 README/使用示例与注意事项

- [x] 任务 8：测试与验证
  - [x] 子任务 8.1：pack/unpack 的端到端最小测试（LocalFS）
  - [x] 子任务 8.2：query 的列表与过滤测试
  - [x] 子任务 8.3：backup 的复制与校验测试（LocalFS→LocalFS）
  - [x] 子任务 8.4：check 的报告结构与健康状态断言

# Task Dependencies
- [任务 2] 依赖 [任务 1]
- [任务 3] 依赖 [任务 1]
- [任务 4] 依赖 [任务 1]
- [任务 5] 依赖 [任务 1] 与 pkg/storage 的能力
- [任务 6] 依赖 [任务 1]
- [任务 8] 依赖前述所有任务

# Tasks
- [x] 任务 1：设计 `cocom ar` 命令结构与共享执行层
  - [x] 子任务 1.1：梳理 `arctl` 现有子命令与参数，确定 `cocom ar` 需要保留的最小命令集合与参数命名
  - [x] 子任务 1.2：抽取可复用的 archive 命令执行辅助逻辑，避免 `cmd/ar` 与 `tools/arctl` 复制核心流程
  - [x] 子任务 1.3：定义统一输出与错误处理方式，兼容人类可读输出与 JSON 输出

- [x] 任务 2：实现 `cocom ar` 命令入口
  - [x] 子任务 2.1：新增 `cmd/ar.go` 并注册到 `rootCmd`
  - [x] 子任务 2.2：实现 `pack`、`unpack`、`query`、`backup`、`check` 子命令骨架与参数解析
  - [x] 子任务 2.3：复用 `cocom` 主配置与初始化链路，避免引入独立 `arctl.*` 配置空间

- [x] 任务 3：接入 archive manager 业务能力
  - [x] 子任务 3.1：打通 `pack` 与 `ArchiveAndRegister`，支持针对单个 `cid` 的定向归档
  - [x] 子任务 3.2：打通 `unpack`、`query`、`backup`、`check` 与 manager/helper 能力
  - [x] 子任务 3.3：确保输出包含 archive ID、cid、路径、校验摘要、位置列表与健康状态等关键字段

- [x] 任务 4：补齐 Mongo IndexStore 场景
  - [x] 子任务 4.1：在 `cocom` 启动期注册 `archive.manager.index.type=mongo` 对应的工厂
  - [x] 子任务 4.2：对接 `comicInfo.archive` 兼容写入路径，确保仅修改 `archive` 子树
  - [x] 子任务 4.3：明确 `cid` 缺失、文档不存在、Mongo 未初始化等错误场景的提示

- [x] 任务 5：文档与示例
  - [x] 子任务 5.1：补充 `cocom ar` 使用文档与示例命令
  - [x] 子任务 5.2：说明 file index 与 mongo index 两种模式下的配置方式与差异
  - [x] 子任务 5.3：说明 `cocom ar` 与 `arctl` 的职责边界与推荐使用场景

- [x] 任务 6：测试与验证
  - [x] 子任务 6.1：为 `cocom ar` 增加 file index 场景的命令级测试
  - [x] 子任务 6.2：增加 `backup/check/query` 的单记录测试，验证不会误处理其他 archive 记录
  - [x] 子任务 6.3：增加 mongo index 场景测试，验证 `comicInfo.archive` 兼容结构与单 `cid` 流程

# Task Dependencies
- [任务 2] 依赖 [任务 1]
- [任务 3] 依赖 [任务 1] 与 [任务 2]
- [任务 4] 依赖 [任务 3]
- [任务 5] 依赖 [任务 2] 与 [任务 4]
- [任务 6] 依赖前述所有任务

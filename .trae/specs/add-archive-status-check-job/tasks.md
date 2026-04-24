# Tasks

- [x] 任务 1：定义任务配置与调度注册
  - [x] 新增 `server.scheduler.archive_status_check` 默认配置，包括 `enabled`、`cron`、`name`、`tags`、`limit`
  - [x] 定义 `targets` 配置结构，至少支持 `backend` 与 `prefix`，并在配置缺失或无效时输出可诊断日志
  - [x] 在 `cmd/server/internal/scheduler` 注册 `ArchiveStatusChecker`，复用现有 cron 注册模式并避免重入

- [x] 任务 2：实现 Mongo 异常 cid 扫描
  - [x] 基于 `archive` 索引结构实现针对指定 backend 列表的扫描逻辑，识别“locator 不存在”和“locator unhealthy”的 cid
  - [x] 按 cid 聚合异常 backend 明细，并支持 `limit` 限流与去重
  - [x] 为扫描逻辑补充覆盖缺失 locator、不健康 locator、重复 cid 聚合等场景的测试

- [x] 任务 3：实现按异常类型触发修复
  - [x] 对缺失 locator 的 backend，解析目标 `storage.Storage` 与 `prefix`，逐个调用 `pkg/archive/manager.Replicate`
  - [x] 对存在不健康 locator 的 cid，调用一次 `pkg/archive/manager.Check(ctx, cid, false)`
  - [x] 明确同一 cid 的执行顺序：先补副本，再检查，并记录统计与错误日志

- [x] 任务 4：完成任务集成与验证
  - [x] 将扫描与修复流程接入调度任务执行入口，输出处理统计
  - [x] 验证任务在 `/admin/cron` 可见且可手动 Run
  - [x] 执行相关测试或最小构建校验，确认不影响现有调度任务

# Task Dependencies
- [任务 2] 依赖 [任务 1]
- [任务 3] 依赖 [任务 1][任务 2]
- [任务 4] 依赖 [任务 3]

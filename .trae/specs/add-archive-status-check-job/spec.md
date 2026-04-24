# 新增定时任务：存档状态检查 Spec

## Why
- `pkg/archive/manager.Check` 已能校验单个存档及其 locator 健康状态，但当前缺少面向服务端的定时批量检查与修复入口。
- 需要在 `cmd/server/internal/scheduler` 中新增任务，按配置关注指定 locator backend，自动发现“缺副本”和“副本不健康”的存档，并触发对应处理动作。

## What Changes
- 新增调度作业 `ArchiveStatusChecker`，注册到 `cmd/server/internal/scheduler`，支持在 `/admin/cron` 中查看并手动执行。
- 新增配置项 `server.scheduler.archive_status_check`：
  - `enabled`：是否启用任务。
  - `cron`：cron 表达式，支持含秒与不含秒。
  - `name`、`tags`：调度任务名称与标签。
  - `limit`：单次最多处理的 cid 数量，避免长时间占用资源。
  - `targets`：待检查的 backend 列表；每个 target 至少包含 `backend` 与 `prefix`，用于判定缺失副本并在需要时执行 `Replicate`。
- 任务运行流程：
  1. 针对每个配置 backend 直接通过 Mongo 条件查询异常 cid 列表，并在查询阶段应用 `limit`：
     - 使用“缺少该 backend 的 locator”条件查询缺副本 cid；
     - 使用“该 backend 的 locator 存在且 `healthy=false`”条件查询不健康 cid。
  2. 按 cid 去重后逐个处理，并保留 cid 对应的异常 backend 明细。
  3. 对于某 backend 缺失 locator 的 cid，调用 `pkg/archive/manager.Replicate` 将该存档复制到该 backend 的目标前缀。
  4. 对于存在不健康 locator 的 cid，调用 `pkg/archive/manager.Check` 重新检查该存档，并刷新健康状态。
- 运行约束：
  - 单个 cid 在一次任务中最多执行一次 `Check`，但可按异常 backend 数量执行多次定向 `Replicate`。
  - 单个 cid 的处理失败不应中断整个批次；任务应记录成功、跳过与失败统计。
  - 当配置 backend 在运行时无法解析到 `storage.Storage` 实例时，跳过该 backend 并记录告警。
  - 不允许先将全部 archive 文档读入内存后再过滤，异常筛选必须依赖 Mongo 查询条件完成。

## Impact
- Affected specs:
  - 新增“按 backend 维度批量检查存档状态”的调度能力。
  - 扩展现有存档修复链路，使 `Replicate` 与 `Check` 可以由定时任务自动触发。
- Affected code:
  - `cmd/server/internal/scheduler`：新增任务注册、执行与日志统计。
  - `cmd/server/config`：新增任务配置默认值。
  - `pkg/archive/manager`：复用现有 `Replicate`、`Check` 与 Mongo 索引模型。
  - 可能新增面向调度器的 Mongo 扫描辅助代码与测试。

## ADDED Requirements
### Requirement: 可配置的存档状态检查任务
系统应提供一个可定时执行的存档状态检查任务，允许按配置指定需要检查的 locator backend 列表。

#### Scenario: 任务注册
- **WHEN** `server.scheduler.archive_status_check.enabled=true` 且配置了有效的 `cron`
- **THEN** 系统注册 `ArchiveStatusChecker` 任务，并在 `/admin/cron` 中展示名称与标签

#### Scenario: 扫描缺失副本
- **WHEN** 任务针对某个 target backend 执行 Mongo 条件查询，且某个 cid 不存在该 backend 的 locator 记录
- **THEN** 该 cid 被加入本次处理列表，并标记该 backend 需要执行 `Replicate`

#### Scenario: 扫描不健康副本
- **WHEN** 任务针对某个 target backend 执行 Mongo 条件查询，且某个 cid 存在该 backend 的 locator 且 `healthy=false`
- **THEN** 该 cid 被加入本次处理列表，并标记该 cid 需要执行一次 `Check`

### Requirement: 按异常类型触发修复动作
系统应根据扫描出的异常类型，分别触发复制或存档检查动作。

#### Scenario: backend 副本缺失
- **WHEN** 处理某个 cid 时发现目标 backend 缺少 locator
- **THEN** 系统使用该 target 的 `backend` 与 `prefix` 调用 `Replicate`，并在成功后更新该 backend 的 locator 信息

#### Scenario: backend 副本不健康
- **WHEN** 处理某个 cid 时存在至少一个目标 backend 的 locator 处于不健康状态
- **THEN** 系统对该 cid 调用一次 `Check(force=false)`，并将检查结果写回 Mongo

#### Scenario: 同一 cid 同时存在缺失与不健康
- **WHEN** 同一 cid 同时包含“部分 backend 缺失 locator”与“部分 backend 不健康”
- **THEN** 系统先执行缺失 backend 的 `Replicate`，再对该 cid 执行一次 `Check`，且整个 cid 只执行一次 `Check`

### Requirement: 批处理健壮性
系统应保证任务具备限流、去重与失败隔离能力。

#### Scenario: 去重与限流
- **WHEN** 不同 backend 的条件查询结果中同一 cid 重复命中异常，或查询返回数量超过 `limit`
- **THEN** 系统按 cid 去重后处理，并仅处理最多 `limit` 个 cid；Mongo 查询本身应使用 `limit` 限制返回数量，剩余 cid 留待下次任务继续处理

#### Scenario: 单项失败隔离
- **WHEN** 某个 cid 的 `Replicate` 或 `Check` 失败
- **THEN** 系统记录该 cid、backend 与错误信息，并继续处理后续 cid，不提前终止整个任务

#### Scenario: backend 配置无效
- **WHEN** 某个 target backend 未注册或缺少必要的复制前缀配置
- **THEN** 系统跳过该 target，记录告警，且不影响其他 backend 与 cid 的处理

## MODIFIED Requirements
无

## REMOVED Requirements
无

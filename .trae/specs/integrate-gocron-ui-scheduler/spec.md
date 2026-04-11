# 服务端定时任务模块（集成 gocron + gocron-ui）Spec

## Why
- 统一在服务端托管周期任务，避免独立进程分散管理与运维复杂度。
- 通过可视化界面管理（添加/禁用/立即执行/查看日志），提升可操作性与可观测性。
- 将现有 cron_probe_comic 能力纳入服务端调度体系，形成可扩展的 Job 注册机制。

## What Changes
- 引入依赖：
  - github.com/go-co-op/gocron/v2
  - github.com/go-co-op/gocron-ui/server
- 新增调度模块（建议位置：cmd/server/internal/scheduler 或 pkg/scheduler）：
  - 创建单例 gocron Scheduler（支持时区、全局上下文、优雅关闭）。
  - 设计可扩展的 Job 注册机制（基于“提供者/注册表”模式，按域组织）。
  - 提供 Start/Stop/Liveness 接口，纳入 server 生命周期。
- 集成 gocron-ui：将 UI 与 API 路由挂载到 Gin 引擎下的「/admin/cron」，沿用现有 LocalGuard 访问保护。
- 提炼 cmd/cron_probe_comic 逻辑为可复用包（建议 pkg/comicprobe）：提供 ProbeComicJob(ctx) error 以被调度；保留原可执行命令复用新包。
- 新增配置项（基于 viper）：scheduler.enable、scheduler.timezone、scheduler.jobs.probeComic.cron、scheduler.jobs.probeComic.enabled、scheduler.ui.routeBase（默认 /admin/cron）。
- 日志与标记：Job 需具备名称、分组/标签、基础运行日志，便于 UI 展示与排障。
- 不改动现有 v1/v2 API 行为；现有基于 robfig/cron 的 verify 调度暂不迁移（后续可合并）。

（可能的变更点参考）
- 入口与路由：
  - [server.go](file:///Users/libing/GolandProjects/cocom/cmd/server/server.go)（在 Run/BuildEngine 挂载 UI 与启动调度）
  - [view/init.go](file:///Users/libing/GolandProjects/cocom/cmd/server/view/init.go)（沿用 LocalGuard 的 /admin 体系）
- 任务实现与抽取：
  - [cmd/cron_probe_comic/main.go](file:///Users/libing/GolandProjects/cocom/cmd/cron_probe_comic/main.go) 抽取核心逻辑至 pkg/comicprobe

## Impact
- Affected specs:
  - 服务端提供“定时任务管理界面”
  - 服务端统一调度并可通过 UI 对任务进行控制
- Affected code:
  - 新增 scheduler 包与初始化代码
  - 在 Gin 路由中增加 /admin/cron 路由组并接入 gocron-ui
  - 提炼并注册 ProbeComic 任务（来自 cmd/cron_probe_comic）
  - 调整 server 生命周期（启动/停止）以托管 Scheduler

## ADDED Requirements
### Requirement: 定时任务管理
系统应提供服务端统一的定时任务管理能力，包含任务注册、启停、立即执行、计划变更与日志展示。

#### Scenario: UI 可用
- WHEN 访问 /admin/cron
- THEN 展示 gocron-ui 页面，并能够查看当前任务列表与状态

#### Scenario: 任务注册与执行
- WHEN 服务启动且 scheduler.enable=true
- THEN 初始化 gocron 调度器，注册 ProbeComic 任务（名称、标签、Cron 来源于配置）
- AND 可在 UI 上看到该任务，并点击“Run”即可立即执行一次

#### Scenario: 优雅关闭
- WHEN 服务收到退出信号
- THEN 优雅停止调度器，等待在途任务安全退出

### Requirement: ProbeComic 任务
系统应将原 cron_probe_comic 能力以 Job 形式纳入调度。

#### Scenario: 成功抓取
- WHEN 触发 ProbeComic 任务
- THEN 完成抓取→保存信息→生成 downList→上传打包，且日志可在 UI 或服务日志中查看

## MODIFIED Requirements
### Requirement: cron_probe_comic 使用方式
新增：可在服务端通过 UI 与调度执行 ProbeComic。保留：原独立命令仍可使用（复用相同包实现）。

## REMOVED Requirements
无


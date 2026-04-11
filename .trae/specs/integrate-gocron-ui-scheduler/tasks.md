# Tasks

- [x] 任务 1：新增调度模块骨架
  - [x] 引入依赖 gocron/v2 与 gocron-ui/server（go.mod）
  - [x] 新建 cmd/server/internal/scheduler（或 pkg/scheduler）：初始化单例 Scheduler，提供 Start/Stop
  - [x] 支持 viper 配置（enable、timezone）
  - [x] 在 server.Run 中接入生命周期（启动/关闭）

- [ ] 任务 2：集成 gocron-ui 到 Gin
  - [x] 在 BuildEngine 中挂载 /admin/cron 路由组，沿用 LocalGuard
  - [x] 将 gocron-ui 的 HTTP 处理器注册到该路由下
  - [x] 验证访问 /admin/cron 可展示 UI

- [ ] 任务 3：提炼 cron_probe_comic 为可复用包
  - [x] 新建 pkg/comicprobe 包，抽取核心逻辑为 ProbeComicJob(ctx) error
  - [x] 保留 cmd/cron_probe_comic/main.go，改为调用新包
  - [x] 确认与现有 pkg/comic 依赖契合（如保存/查询接口）

- [ ] 任务 4：注册 ProbeComic 任务
  - [x] 在 scheduler 模块注册名为 ProbeComic 的任务，添加标签（comic、probe）
  - [x] 支持从配置读取 Cron 表达式与开关
  - [x] 在 UI 上可见并可手动「Run」

- [ ] 任务 5：验证与文档
  - [x] 构建通过，无编译错误
  - [x] 手测访问 /admin/cron，任务列表可见，能立即执行一次
  - [x] 验证优雅停机时任务退出
  - [x] 补充 README/注释中的配置说明（可选）

# Task Dependencies
- [任务 2] 依赖 [任务 1]
- [任务 4] 依赖 [任务 3]
- [任务 5] 依赖 [任务 1][任务 2][任务 4]

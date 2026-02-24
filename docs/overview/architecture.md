# 架构总览

- 入口与 CLI：`main.go:23-25` 调用 `cmd.Execute()`；根命令初始化与配置读取 `cmd/root.go:57-100`
- Web 服务：`cmd/server/server.go:60-101` 初始化 Gin、注册视图与处理器、提供关闭接口
- 视图层：`cmd/server/view/init.go:43-61` 注册静态资源与页面路由；模板函数 `cmd/server/view/init.go:64-76`
- 领域服务：`pkg/comic/service.go:10-21` 定义服务接口；实现 `pkg/comic/service.go:31-46`
- 校验器：任务与进度 `pkg/comic/verify.go:26-66,153-166`；并发池与调度 `pkg/comic/verify.go:310-353`
- 存储抽象：`pkg/comic/storage.go:12-24` 接口与过滤器；Mongo 实现见 `cmd/server/internal/*/storage.go`
- 路由分组：V2 API 在 `cmd/server/server.go:90-91` 注册 `/v2/api/{onecomic|nhcomic}`

## 运行流程
- 启动 `cocom server` 后：注册 `/api/*` 旧接口与 `/v2/api/*` 新接口，以及 `/debug/pprof/*`
- 旧接口通过自定义 `ServeMux` 统一包裹请求与响应 `cmd/server/handler/mux.go:28-38`；响应格式 `pkg/httpwrap/http.go:27-37,39-59`
- 新接口通过 `pkg/comic/Handler` 暴露校验任务、检索与信息查询 `pkg/comic/handler.go:26-39`

## 关键特性
- 漫画图片校验与自动修复；损坏图片生成下载列表 `pkg/comic/verify.go:471-486`
- 定时任务：基于 cron 时间表达式 `pkg/comic/verify.go:661-732`
- 下载器：HTTP 客户端与 Wget 双实现 `pkg/comic/comic.go:168-289,364-439`

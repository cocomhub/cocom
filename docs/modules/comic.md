# Comic 模块

## 领域模型
- 接口定义：`pkg/comic/comic.go:22-39`
- 默认实现：`pkg/comic/comic.go:55-70,94-122`
- 图片结构：`pkg/comic/comic.go:41-46`
- 校验信息：`pkg/comic/comic.go:48-53,124-160`

## 校验器与任务
- 任务/进度结构：`pkg/comic/verify.go:26-66,153-166,280-289`
- 状态枚举：`pkg/comic/verify.go:51-60`
- 并发池：`pkg/comic/verify.go:310-353`（verifyPool/fixPool）
- 启动任务：`pkg/comic/verify.go:356-403`
- 任务执行：`pkg/comic/verify.go:405-521`（按漫画并发验证）

## 校验流程
- 单漫画校验：`pkg/comic/verify.go:532-553`（遍历图片并记录异常）
- 单图片校验：`pkg/comic/verify.go:555-577`（调用 `imaging.VerifyImage`）
- 自动修复：`pkg/comic/verify.go:579-601`（下载到 `.fix` 验证后替换）
- 生成下载列表：`pkg/comic/verify.go:471-486`（当不自动修复时写入 URL 列表）

## 定时任务
- 配置与启动：`pkg/comic/verify.go:661-732`（支持 `@every` 与 cron 表达式）

## 下载器
- HTTP 下载器：断点续传/重试 `pkg/comic/comic.go:168-289`
- Wget 下载器：超时、重试与 UA `pkg/comic/comic.go:364-439`
- 缺少 wget 将触发 panic `pkg/comic/comic.go:441-464`

## 服务接口
- Service 定义：`pkg/comic/service.go:10-21`
- 启动校验/任务管理/检索与信息：`pkg/comic/service.go:48-137`

## 存储实现
- 抽象接口与过滤器：`pkg/comic/storage.go:12-24,31-41,43-56`
- 内存存储：`pkg/comic/storage.go:125-200`
- Mongo 存储（独立）：`pkg/comic/storage/mongo.go:14-127`
- 服务器内部存储：`cmd/server/internal/{comic,onecomic}/storage.go`

## 指标与监控
- 指标结构与收集：`pkg/comic/metrics.go:8-19,21-75`

## 旧接口与新接口
- Legacy `/api/*` 通过 `ServeMux` 统一响应 `cmd/server/handler/mux.go:28-38`，HTTP 包裹 `pkg/httpwrap/http.go:27-37,39-59`
- V2 接口由 `pkg/comic/handler.go:26-39` 提供路由与处理逻辑

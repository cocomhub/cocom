# 部署与运行

- 构建：`go build ./...`
- 启动服务：`cocom server -p 15456`，端口绑定 `cmd/server.go:60-65`
- 配置文件：默认从 `cmd/root.go:74-100` 所示路径加载，样例见 `conf/cocom.yaml:1-28`
- 日志：通过 `pkg/logging`初始化 `cmd/root.go:100`，支持文件与控制台输出
- Docker：基于仓库 `Dockerfile` 构建，挂载数据目录与配置文件，暴露 `port`

## 运行时管理

- 优雅关闭：POST  ` /admin/server/shutdown` 触发 `server.Shutdown`（`cmd/server/server.go:72-81`）
- pprof 调试：`/debug/pprof/*` 入口 `cmd/server/handler/mux.go:41-46`

## 目录建议

- 数据存储：`cocom.storage.path` 指向持久目录
- 下载目录：`download.downloadDir`，用于生成下载列表与批量下载
- 日志目录：`logging.filename`


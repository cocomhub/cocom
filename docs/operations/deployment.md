# 部署与运行

- 构建：`go build ./...`
- 启动服务：`cocom server`，监听 `server.listen.http.addr`（默认 `127.0.0.1:8080`，仅本机；需对外暴露请显式配置 `0.0.0.0:8080` 或具体 IP）。顶层 `port`/`COCOM_PORT`/`-p` 旧入口已移除。
- 配置文件：默认从 `cmd/root.go:74-100` 所示路径加载，样例见 `conf/cocom.yaml:1-28`
- 日志：通过 `pkg/logging`初始化 `cmd/root.go:100`，支持文件与控制台输出
- Docker：基于仓库 `Dockerfile` 构建（EXPOSE 8080，监听默认 `127.0.0.1:8080`），挂载数据目录与配置文件。容器内需对外访问请通过 `server.listen.http.addr` 或 `-p` 显式配置监听地址。

## 运行时管理

- 优雅关闭：POST  ` /admin/server/shutdown` 触发 `server.Shutdown`（`cmd/server/server.go:72-81`）
- pprof 调试：`/debug/pprof/*` 入口 `cmd/server/handler/mux.go:41-46`

## 目录建议

- 数据存储：`cocom.storage.path` 指向持久目录
- 下载目录：`download.downloadDir`，用于生成下载列表与批量下载
- 日志目录：`logging.filename`


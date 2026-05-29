# 部署与运行

- 构建：`go build ./...`
- 启动服务：`cocom server -p 15456`
- 配置文件：默认从 XDG 配置目录加载（可用 `--config` 指定路径），样例见 `conf/cocom.yaml`
- 日志：通过 `pkg/logging` 初始化，支持文件与控制台输出
- Docker：基于仓库 `Dockerfile` 构建，默认暴露 `15456`，探针路径 `/healthz` 与 `/readyz`，建议挂载数据目录 `/data/cocom`

## 运行时管理

- 优雅关闭：POST `/admin/server/shutdown`（需要本地请求或配置 `admin.token`）
- pprof 调试：`/debug/pprof/*`

## 目录建议

- 数据存储：`cocom.storage.path` 指向持久目录
- 下载目录：`download.downloadDir`，用于生成下载列表与批量下载
- 日志目录：`logging.filename`

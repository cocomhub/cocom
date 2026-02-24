# server 命令

- 启动 HTTP 服务：`cocom server -p <port>`（`cmd/server.go:60-65`）
- 服务初始化：Gin 中间件与视图注册 `cmd/server/server.go:60-71`
- 管理接口：`POST /admin/server/shutdown` 关闭服务 `cmd/server/server.go:72-81`
- V2 API 挂载：`/v2/api/onecomic`、`/v2/api/nhcomic` `cmd/server/server.go:90-91`

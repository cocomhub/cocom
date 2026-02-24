# 调试与管理接口

- pprof：
  - `/debug/pprof/` `cmd/server/handler/mux.go:41-46`
- 管理：
  - 关闭服务：`POST /admin/server/shutdown` `cmd/server/server.go:72-81`

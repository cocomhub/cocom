# Logging 模块（clog）

## 使用位置
- 初始化：`cmd/root.go:100`
- 请求追踪：注入 Trace ID 并打印请求 URI `cmd/server/handler/mux.go:34-37`
- 服务生命周期与错误：`cmd/server/server.go:48-58,94-101`

## 能力
- 文件/控制台双通道，级别可配置（见配置文档）
- 追踪上下文：`clog.NewTraceCtx` 与 `clog.GetTraceID`

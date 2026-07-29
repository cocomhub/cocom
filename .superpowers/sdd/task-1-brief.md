# Task 1: Config 核心层深度审查与修复

## 范围
- `internal/config/manager.go` — 检查 SetDefault 中是否有死键（设置了默认值但无代码读取的键）
- `internal/config/config.go` — 检查 Init 流程是否正确
- `internal/config/types.go` — 检查类型定义完整性
- `cmd/server/config.go` — 检查是否可移除（仅含 config-doc 注释）
- `cmd/root.go` — 检查初始化链
- `internal/rootcli/rootcli.go` — 检查初始化链

## 审查目标
1. `http.enable_proxy` 和 `http.proxy` 的 SetDefault 是否被任何代码消费？如无，移除这些死默认值
2. `cmd/server/config.go` 是否还有必要保留（仅 config-doc 注释）
3. 所有残留的空 `init()` 函数
4. types.go 中是否有其他未使用的类型/字段
5. Config 结构体是否有字段与 SetDefault 不匹配的

## 验证
- `go build ./internal/config/...`
- `go vet ./internal/config/...`
- `go test -tags=memory_storage_integration ./internal/config/...`
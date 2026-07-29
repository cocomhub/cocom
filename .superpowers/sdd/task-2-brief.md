# Task 2: pkg 层 Viper 消除深度审查与修复

## 范围
- `pkg/mongowrap/` — mongowrap.Init() 调用链、Client() 保护
- `pkg/middlewares/` — cors.go, ratelimit.go, localguard.go 配置注入
- `pkg/logging/` — config.go 空 init() 已移除，验证编译
- `pkg/archive/manager/` — config.go 的 SetFromViper 流程
- `pkg/storage/` — config.go 的 SetFromConfigs 流程
- `pkg/comic/` — config.go 仅注释
- `pkg/download/` — config.go 完整性, downloader.go 的 NewInitConfig 是否正确
- `pkg/imaging/` — 检查是否有 viper 残留

## 审查目标
1. 所有 pkg 层已无 `viper.Get` 调用残留
2. `pkg/download/config.go` 的 `DownloaderConfig` 中 `EnableProxy`/`ProxyURL` 是否从 Config 结构体注入
3. `pkg/mongowrap/mongo.go` 是否有 `init()` 残留
4. `pkg/imaging/` 是否有 viper 依赖
5. 所有 `config.go` 新文件字段是否完整

## 验证
- `go build ./pkg/...`
- `go vet ./pkg/...`
- `go test -tags=memory_storage_integration ./pkg/...`

# Task 3: 测试完整性深度审查与修复

## 范围
- `internal/config/config_test.go` — 测试用例覆盖
- `internal/config/config_keys_test.go` — 全键默认值测试
- `internal/rootcli/rootcli_test.go` — 基础测试
- `pkg/mongowrap/mongowrap_test.go` — MongoDB wrap 测试
- `pkg/middlewares/middlewares_test.go` — 中间件测试
- `pkg/logging/logging_test.go` — 日志测试
- `pkg/comic/comic_test.go` — 业务逻辑测试
- `pkg/comic/storage_test.go` — Storage 接口测试
- `pkg/download/download_test.go` — 下载测试

## 审查目标
1. 测试是否真正有效（非空壳、非 t.Skip()）
2. 测试是否覆盖了配置初始化路径
3. 测试数据是否覆盖边界条件
4. config_keys_test.go 的 keyTestCases 是否与 manager.go 的 SetDefault 完全对应
5. 是否有测试文件使用了 t.Skip() 跳过关键测试

## 验证
- `go test -tags=memory_storage_integration -count=1 ./internal/config/...`
- `go test -tags=memory_storage_integration -count=1 ./pkg/mongowrap/... ./pkg/logging/... ./pkg/middlewares/...`

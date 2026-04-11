# Register Global Storages From Config — 验证清单

- [x] 存在 RegisterKnownFromViper 并包含 gallery/archive/archive-temp 注册逻辑
- [x] 容错逻辑：空路径不注册；遇到 ErrStorageAlreadyRegistered 忽略错误
- [x] 解析 storage.backends 并在 type=localfs 时注册对应后端
- [x] 运行 registry 测试全部通过：go test ./pkg/storage/registry -v
- [x] 文档已更新：docs/config.md 新增“存储注册”段与示例

结论：全部验证项通过。

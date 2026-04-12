# BaiduPCS 命令存储驱动验收清单

- [x] `pkg/storage` 已新增 `baidupcs` 驱动，并可通过 `storage.backends` 成功注册具名实例
- [x] 驱动已实现 `Put/Get/Stat/List/Delete/Copy/Move`，且行为符合现有 `Storage` 接口语义
- [x] 逻辑 `key` 到远端根目录的映射稳定可预测，越界路径会被拒绝
- [x] 命令执行具备超时、退出码检查、输出解析与统一错误映射能力
- [x] `pkg/archive/manager` 与 `tools/arctl` 可直接使用 `baidupcs` 后端，无需额外接口改造
- [x] 单元测试覆盖命令拼装、结果解析、错误映射和主要对象操作
- [x] 至少存在一组基于 fake `BaiduPCS-Go` 的自动化验证，且不依赖真实百度网盘环境
- [x] 文档已更新：配置示例、依赖说明、临时文件策略、已知限制与使用建议

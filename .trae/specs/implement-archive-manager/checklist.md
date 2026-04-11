* [x] 已定义 Manager 接口与配置/选项，包结构清晰（pkg/archive/manager）

* [x] ArchiveMeta/StorageLocator/Checksum/ReplicaHealth 等模型完整并稳定

* [x] 元数据持久化（IndexStore）可用，支持 CRUD 与基本并发安全

* [x] 与 pkg/storage 交互正常：保存/检索/删除/列举均可工作

* [x] 一致性检查可对比 Stat/ETag/Hash 并输出报告与健康状态

* [x] 多存储备份/迁移（Replicate）可在不同后端间复制与校验并更新元数据

* [x] 保留策略可执行，支持“远端备份完成后自动清理本地副本”

* [x] 已与 pkg/archive 打包流程集成并通过最小端到端验证

* [x] 文档与示例完善：使用说明、策略配置、注意事项

* [x] 单元与集成测试通过（含 LocalFS→LocalFS 的复制与一致性检查）


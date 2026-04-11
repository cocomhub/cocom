* [x] arctl 可执行入口存在（cmd/arctl），支持全局参数解析

* [x] 子命令 pack/unpack/query/backup/check 均可执行（显示帮助与基本参数）

* [x] pack 可完成打包并注册，输出 ID/位置/校验摘要（含 JSON 输出）

* [x] unpack 可从归档或 ID 解包到目标目录并完成基本校验

* [x] query 可按 ID/名称/标签返回基本信息、位置与健康状态

* [x] backup 在 LocalFS→LocalFS 场景下完成复制与校验并更新元数据

* [x] check 可对副本执行 Stat/ETag/可选 Hash 校验并输出报告

* [x] 统一错误与退出码符合约定；--output=json 输出结构稳定

* [x] 文档与示例完整；最小端到端测试通过

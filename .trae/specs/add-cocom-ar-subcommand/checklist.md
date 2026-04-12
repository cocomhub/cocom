# cocom ar 子命令验收清单

- [x] `cocom` 根命令已新增 `ar` 子命令，帮助信息可见且参数清晰
- [x] `pack/unpack/query/backup/check` 可在 `cocom ar` 下执行，并面向单个 `cid` 或单个 archive 记录工作
- [x] `cocom ar` 复用了 `cocom` 主配置与初始化链路，没有引入独立的 `arctl.*` 配置空间
- [x] 核心归档逻辑复用了共享实现，没有在 `cmd/ar` 中重复复制 archive manager 业务
- [x] `archive.manager.index.type=mongo` 时，`cocom ar` 可成功构造 manager 并执行命令
- [x] Mongo 写入仅影响 `comicInfo.archive` 子树，兼容旧 `archive.path/size/algorithm/md5` 结构
- [x] 自动化测试覆盖 file index 场景下的单记录 pack/unpack/query/backup/check
- [x] 自动化测试覆盖 mongo index 场景下至少 pack/query/check 的兼容路径
- [x] 文档已更新：命令示例、配置方式、与 `arctl` 的使用边界

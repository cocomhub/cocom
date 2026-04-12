# cocom ar 子命令 Spec

## Why
当前单个归档的调试与验证主要依赖独立工具 `arctl` 或直接启动 server 后走整套在线流程，缺少集成在 `cocom` 主 CLI 中的最小化操作入口。需要新增 `cocom ar` 子命令，使开发者可以针对单个 `cid` 执行打包、解包、备份与校验，尤其要覆盖 `archive manager indexstore=mongo` 的场景。

## What Changes
- 在 `cocom` 根命令下新增 `ar` 子命令，提供针对单个 `cid` 的 archive 操作入口
- 支持 `pack`、`unpack`、`backup`、`check`、`query` 等子命令，面向单个 `cid` 或单个 archive 记录工作
- 复用 `pkg/archive/manager` 与现有 archive helper，避免在 `cmd` 层重复实现归档逻辑
- 为 `archive.manager.index.type=mongo` 提供启动期工厂注册与命令级兼容支持
- 明确 `comicInfo.archive` 兼容边界，保证 Mongo 写入仅修改 `archive` 子树，不破坏旧字段读路径
- 补充 CLI 文档、示例与端到端测试，覆盖 file index 与 mongo index 两类场景

## Impact
- Affected specs: cocom CLI 子命令扩展、archive manager 命令复用、mongo indexstore 装配
- Affected code: `cmd/`、`pkg/archive/manager`、Mongo 初始化/注册链路、CLI 文档与测试

## ADDED Requirements
### Requirement: cocom ar 命令入口
系统 SHALL 在 `cocom` 根命令下提供 `ar` 子命令，用于对单个 archive 记录执行定向操作，而不要求启动完整 server 业务流。

#### Scenario: 查看命令帮助
- **WHEN** 用户执行 `cocom ar --help`
- **THEN** CLI 展示 `ar` 的用途、子命令列表与主要参数

#### Scenario: 使用主配置初始化命令
- **WHEN** 用户执行任意 `cocom ar` 子命令
- **THEN** 命令复用 `cocom` 主配置加载与全局初始化链路，而不是单独维护一套 `arctl.*` 配置命名空间

### Requirement: 针对单个 cid 的归档打包
系统 SHALL 支持通过 `cocom ar pack` 针对单个 `cid` 对应内容执行打包并注册到 archive manager。

#### Scenario: pack 单个 cid
- **WHEN** 用户提供 `cid`、源目录或可推导输入，执行 `cocom ar pack`
- **THEN** 系统完成归档打包、写入 archive manager，并输出归档 ID、路径、校验摘要与后端定位信息

#### Scenario: Mongo 索引下 pack
- **WHEN** `archive.manager.index.type=mongo`
- **THEN** 系统通过已注册的 Mongo IndexStore 更新对应 `cid` 的 `archive` 子树，且不覆盖非 `archive` 字段

### Requirement: 针对单个 archive 的解包
系统 SHALL 支持通过 `cocom ar unpack` 按归档 ID、`cid` 或明确归档路径执行解包。

#### Scenario: 通过 cid 解包
- **WHEN** 用户提供 `cid` 且 IndexStore 中存在对应 archive 记录
- **THEN** 系统解析主归档位置并完成解包到目标目录

#### Scenario: 记录不存在
- **WHEN** 用户提供的 `cid` 或 ID 在 IndexStore 中不存在
- **THEN** 命令返回清晰的未找到错误，并保持非零退出码

### Requirement: 针对单个 archive 的备份与校验
系统 SHALL 支持通过 `cocom ar backup` 与 `cocom ar check` 对单个 archive 记录执行副本备份与一致性校验。

#### Scenario: backup 单个 cid
- **WHEN** 用户指定 `cid` 与目标 storage backend 执行 `backup`
- **THEN** 系统仅复制该 archive 记录对应主归档到目标后端，并更新位置列表与健康状态

#### Scenario: check 单个 cid
- **WHEN** 用户指定 `cid` 或归档 ID 执行 `check`
- **THEN** 系统输出该记录的主副本校验结果与健康状态摘要

### Requirement: query 面向单个 cid 的定位
系统 SHALL 支持通过 `cocom ar query` 按 `cid` 或 archive ID 查询归档元数据，便于测试时快速定位目标记录。

#### Scenario: 查询单个 cid
- **WHEN** 用户执行 `cocom ar query --cid <cid>`
- **THEN** 命令返回该 archive 记录的关键信息，包括路径、校验信息、位置列表与健康状态

### Requirement: Mongo IndexStore 启动期装配
系统 SHALL 在 `cocom` 启动链路中为 archive manager 的 `mongo` index type 注册可用工厂，使 `cocom ar` 与其他命令在不启动 server 的情况下也能构造 manager。

#### Scenario: mongo 工厂已注册
- **WHEN** 配置 `archive.manager.index.type=mongo`
- **THEN** `manager.New()` 成功构建对应的 Mongo IndexStore，而不会因类型未注册失败

#### Scenario: comicInfo 兼容写入
- **WHEN** `cocom ar pack` 或 `backup/check` 回写 Mongo 索引
- **THEN** 仅修改 `comicInfo.archive` 子树，保留旧 `archive.path/size/algorithm/md5` 与 `archive.manager` 的兼容结构

### Requirement: 共享归档命令实现
系统 SHALL 优先复用现有 `arctl` 与 archive manager 的可复用逻辑，避免在 `cmd/ar` 中复制核心业务。

#### Scenario: 命令逻辑复用
- **WHEN** 新增 `cocom ar` 命令
- **THEN** 共享的参数解析、输出结构或执行辅助逻辑被抽取到公共实现，减少与 `arctl` 的行为漂移

### Requirement: 可测试的 CLI 行为
系统 SHALL 提供自动化测试覆盖 `cocom ar` 在 file index 与 mongo index 场景下的关键行为。

#### Scenario: file index 场景
- **WHEN** 测试使用 LocalFS + file IndexStore
- **THEN** `pack/unpack/backup/check/query` 的最小路径全部可执行

#### Scenario: mongo index 场景
- **WHEN** 测试启用 Mongo 集成环境或可替代桩实现
- **THEN** 至少验证 `pack/query/check` 在 `comicInfo.archive` 模式下可用且兼容旧结构

## MODIFIED Requirements
### Requirement: archive manager CLI 接入方式
系统 SHALL 继续保留现有 `arctl` 工具能力，同时允许 `cocom` 主 CLI 通过 `ar` 子命令复用相同 archive manager 能力；未使用 `ar` 时，现有 `cocom` 行为保持不变。

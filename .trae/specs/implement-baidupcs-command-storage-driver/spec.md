# BaiduPCS 命令存储驱动 Spec

## Why
当前 `pkg/storage` 仅提供 `localfs` 驱动，归档副本与文件型索引只能落到本地文件系统。需要复用成熟的 `BaiduPCS-Go` 命令行能力，为 `pkg/storage` 增加百度网盘后端，而不直接耦合百度 PCS API 细节。

## What Changes
- 新增 `pkg/storage/baidupcs` 驱动，基于 `BaiduPCS-Go` 可执行命令实现 `Storage` 接口
- 新增 BaiduPCS 命令执行适配层，负责命令拼装、标准输出解析、超时控制与错误映射
- 扩展 `storage.backends` 配置，支持声明 `type=baidupcs` 的后端实例
- 明确远端根目录与逻辑 `key` 的映射规则，保持现有 `Storage` URI 约定
- 为归档复制、健康检查、文件型索引等现有 `storage` 使用方提供兼容支持
- 补充单元测试、桩命令测试与文档示例

## Impact
- Affected specs: `pkg/storage` 驱动扩展、全局存储注册、archive 存储后端接入
- Affected code: `pkg/storage`、`pkg/storage/config.go`、`cmd/root.go` 启动链路、相关文档与测试

## ADDED Requirements
### Requirement: BaiduPCS 命令驱动注册
系统 SHALL 提供一个可注册到 `pkg/storage` 的 `baidupcs` 驱动，并允许通过 `storage.backends` 配置创建具名实例。

#### Scenario: 从配置注册 BaiduPCS 后端
- **WHEN** 配置中存在 `type=baidupcs` 的后端定义，且必需元数据完整
- **THEN** 启动过程成功创建该后端实例并可通过 `storage.Get(name)` 获取

#### Scenario: 缺少必需配置
- **WHEN** 未提供命令路径、远端根目录或等效必需元数据
- **THEN** 驱动初始化失败，并返回可定位问题的配置错误

### Requirement: 基于命令实现对象读写
系统 SHALL 通过 `BaiduPCS-Go` 命令完成 `Put`、`Get`、`Stat`、`List`、`Delete`、`Copy`、`Move` 等对象操作，并保持与 `Storage` 接口一致的行为语义。

#### Scenario: 上传对象
- **WHEN** 调用方执行 `Put(key, reader, ...)`
- **THEN** 驱动将内容写入受控临时文件，再调用 `BaiduPCS-Go` 上传到远端目标路径，并返回对象元数据

#### Scenario: 下载对象
- **WHEN** 调用方执行 `Get(key)`
- **THEN** 驱动将远端对象下载到受控临时文件并返回可读取流，同时在读取完成后释放临时资源

#### Scenario: 复制与移动对象
- **WHEN** 调用方执行 `Copy` 或 `Move`
- **THEN** 驱动优先使用远端命令完成服务端复制或移动，而不是强制走本地回传

### Requirement: 逻辑 Key 到远端路径的映射
系统 SHALL 将 `Storage` 逻辑 `key` 稳定映射到配置的 BaiduPCS 远端根目录之下，并阻止越界路径。

#### Scenario: 规范化 key
- **WHEN** 调用方传入包含多余分隔符或 `.` 的逻辑路径
- **THEN** 驱动使用与 `pkg/storage` 一致的规范化结果拼接远端路径

#### Scenario: 越界路径
- **WHEN** 调用方传入会逃逸远端根目录的路径
- **THEN** 驱动拒绝该操作并返回安全错误

### Requirement: 命令执行可观测且可诊断
系统 SHALL 为 BaiduPCS 命令执行提供超时、退出码检查、标准输出/标准错误解析与统一错误映射。

#### Scenario: 远端对象不存在
- **WHEN** BaiduPCS-Go 返回“文件不存在”类错误
- **THEN** 驱动将其转换为 `pkg/storage` 的 `ErrNotFound`

#### Scenario: 命令执行失败
- **WHEN** BaiduPCS-Go 进程返回非零退出码或超时
- **THEN** 驱动返回包含后端名称、操作类型与原始诊断摘要的错误

### Requirement: 兼容 archive 使用方式
系统 SHALL 允许 `pkg/archive/manager` 与 `tools/arctl` 像使用其他 `storage` 后端一样使用 `baidupcs` 实例，无需修改调用方接口。

#### Scenario: 归档复制到百度网盘
- **WHEN** `ReplicateToStorage` 目标后端为 `baidupcs`
- **THEN** 归档副本可以写入远端，并记录正确的后端名、对象 key 与 URI

#### Scenario: 文件型索引落到百度网盘
- **WHEN** archive manager 的 `index.type=file` 且 `fileStoreName` 指向 `baidupcs`
- **THEN** 索引文件的读写、列举与删除保持可用

### Requirement: 可测试的命令适配层
系统 SHALL 提供无需真实百度网盘账户即可验证的测试机制，覆盖命令拼装、输出解析、错误映射与关键接口行为。

#### Scenario: 使用桩命令执行单元测试
- **WHEN** 测试环境注入 fake `BaiduPCS-Go` 可执行文件
- **THEN** 测试可验证驱动生成的命令参数、解析结果与接口行为，而不依赖外部网络

## MODIFIED Requirements
### Requirement: Storage 后端配置扩展
系统 SHALL 继续支持现有 `localfs` 后端配置，同时允许在相同 `storage.backends` 结构下声明 `baidupcs` 后端，并保持未使用新类型时的现有行为不变。

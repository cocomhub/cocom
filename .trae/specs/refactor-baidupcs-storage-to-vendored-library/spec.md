# BaiduPCS 存储驱动改为直接调用 vendored 库 Spec

## Why
当前 `pkg/storage/baidupcs` 假设 `BaiduPCS-Go` 命令输出为 JSON，并通过标准输出解析对象元数据；这一前提与实际命令行工具行为不符，导致驱动实现与真实能力面脱节。项目已经将 `BaiduPCS-Go` vendored 到仓库内，因此需要改为直接调用 `github.com/qjfoidnh/BaiduPCS-Go/baidupcs` 库，复用其真实 API、错误类型和认证模型。

## What Changes
- 将 `pkg/storage/baidupcs` 从“外部命令执行 + stdout/stderr 解析”重构为“直接封装 vendored `baidupcs.BaiduPCS` 客户端”
- 移除对命令路径、工作目录、全局参数和命令超时的核心依赖，改为基于库初始化参数构建客户端
- 新增面向 `baidupcs` 库的认证与连接配置，包括 BDUSS / cookies、可选 STOKEN / SBOXTKN、AppID、HTTPS、PCS 地址和 User-Agent 等
- 通过 vendored 库的 `FilesDirectoriesMeta`、`FilesDirectoriesList`、`Remove`、`Copy`、`Move`、`Mkdir`、`DownloadFile` 等能力实现 `Storage` 接口
- 保留现有逻辑 `key` 到远端根目录的映射、临时文件读写策略和 `storage` URI 约定
- 调整测试策略，不再伪造命令行可执行文件，而是通过可替换的库适配接口或 fake client 覆盖关键行为
- **BREAKING**：现有 `type=baidupcs` 的命令式配置项将不再作为主实现入口

## Impact
- Affected specs: `pkg/storage` 驱动扩展、全局存储注册、archive 存储后端接入
- Affected code: `pkg/storage/baidupcs`、`pkg/storage/config.go`、相关测试、配置示例与文档

## ADDED Requirements
### Requirement: 基于 vendored baidupcs 库实现驱动
系统 SHALL 直接通过 `github.com/qjfoidnh/BaiduPCS-Go/baidupcs` 提供的 Go API 实现 `pkg/storage/baidupcs`，而不是依赖 `BaiduPCS-Go` 可执行文件的命令输出格式。

#### Scenario: 创建 BaiduPCS 客户端
- **WHEN** `storage.backends` 中声明 `type=baidupcs` 且认证参数有效
- **THEN** 驱动初始化过程创建并持有 vendored `baidupcs.BaiduPCS` 客户端实例

#### Scenario: 执行对象操作
- **WHEN** 调用方执行 `Put/Get/Stat/List/Delete/Copy/Move`
- **THEN** 驱动通过 vendored 库的对应能力完成远端操作，而不是拼装命令并解析 stdout/stderr

### Requirement: BaiduPCS 驱动配置改为认证模型
系统 SHALL 使用与 vendored `baidupcs` 库匹配的配置模型初始化驱动，至少支持远端根目录和一组有效认证信息，并允许声明常用网络行为参数。

#### Scenario: 使用 BDUSS 初始化
- **WHEN** 配置提供 `root` 与 `bduss`，以及可选 `stoken`、`sboxtkn`
- **THEN** 驱动可创建 `baidupcs` 客户端并用于远端访问

#### Scenario: 使用 cookies 初始化
- **WHEN** 配置提供 `root` 与 cookie 字符串
- **THEN** 驱动可通过等效的 cookie 初始化方式创建 `baidupcs` 客户端

#### Scenario: 调整库级网络参数
- **WHEN** 配置提供 `appID`、`enableHTTPS`、`pcsAddr`、`pcsUserAgent` 或 `panUserAgent`
- **THEN** 驱动将这些配置下沉到 vendored 客户端，而不是保留为命令行参数

### Requirement: 保持 Storage 接口语义
系统 SHALL 在切换到底层库实现后，继续满足现有 `Storage` 接口的语义约束，包括 key 规范化、越界路径保护、对象元数据映射和 archive 调用兼容性。

#### Scenario: 逻辑 key 解析
- **WHEN** 调用方传入逻辑 `key`
- **THEN** 驱动继续将其规范化并映射到配置的远端根目录下，且拒绝逃逸根目录的路径

#### Scenario: 元数据映射
- **WHEN** vendored 库返回 `FileDirectory` 元信息
- **THEN** 驱动将其稳定映射为 `storage.ObjectMeta`，并保持 `URI`、`Size`、`ModTime`、`ETag` 等字段语义一致

#### Scenario: archive 与 arctl 复用
- **WHEN** `pkg/archive/manager` 或 `tools/arctl` 使用 `baidupcs` 后端
- **THEN** 现有调用方无需额外接口改造即可继续工作

### Requirement: 统一库错误到 storage 错误
系统 SHALL 将 vendored `baidupcs` / `pcserror` 返回的错误归一化为 `pkg/storage` 约定的错误类型，并保留诊断信息。

#### Scenario: 远端对象不存在
- **WHEN** vendored 库返回“文件不存在”或等价的远端错误
- **THEN** 驱动返回 `storage.ErrNotFound`

#### Scenario: 权限或认证失败
- **WHEN** vendored 库返回鉴权失败、权限不足或 cookie 无效等错误
- **THEN** 驱动返回包含后端名、操作类型和关键诊断摘要的错误，并映射到合适的 `storage` 错误类别

### Requirement: 可测试的库适配层
系统 SHALL 提供无需真实百度网盘环境的测试机制，覆盖库初始化、关键对象操作、元数据映射和错误映射。

#### Scenario: 使用 fake client 验证驱动行为
- **WHEN** 测试环境注入 fake `baidupcs` 适配实现
- **THEN** 测试可验证 `Put/Get/Stat/List/Delete/Copy/Move` 的调用、路径处理与错误映射，而不依赖命令行桩程序

## MODIFIED Requirements
### Requirement: Storage 后端配置扩展
系统 SHALL 继续支持现有 `localfs` 后端配置，并允许在相同 `storage.backends` 结构下声明 `baidupcs` 后端；`baidupcs` 后端的配置语义调整为“库客户端初始化参数 + 远端根目录 + 本地临时文件策略”，不再以命令路径和命令参数作为主配置模型。

## REMOVED Requirements
### Requirement: BaiduPCS 命令执行适配层
**Reason**: `BaiduPCS-Go` 命令输出并不保证符合当前驱动假设的 JSON 格式，继续依赖命令执行会放大解析脆弱性，并且无法直接复用 vendored 代码中的真实错误语义与客户端配置逻辑。
**Migration**: 将现有 `command`、`commandPath`、`args`、`globalArgs`、`workDir`、`timeout` 配置迁移为 `bduss` / `cookies`、可选 `stoken` / `sboxtkn`、`appID`、`enableHTTPS`、`pcsAddr`、`pcsUserAgent`、`panUserAgent` 等库初始化参数；`root` 与 `tempDir` 继续保留。

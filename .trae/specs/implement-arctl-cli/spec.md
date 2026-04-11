# arctl 命令行工具 Spec

## Why
为现有存档管理能力提供统一的命令行入口，便于开发者与 CI/运维以一致方式进行存档的打包、解包、查询、备份与一致性检查，复用 pkg/archive/manager 的能力，避免重复实现。

## What Changes
- 增加独立可执行工具 arctl（cmd/arctl），围绕 pkg/archive/manager 暴露 CLI 子命令
- 支持子命令：pack（打包并注册）、unpack（解包）、query（查询元数据/定位）、backup（多存储备份/迁移）、check（一致性检查）
- 提供统一配置加载与全局参数（如 --config、--output=json、--verbose）
- 标准化退出码与错误输出（人类可读与机器可解析的 JSON 输出）
- Storage 抽象复用现有实现，基础版本仅支持本地文件系统；保留云端后端扩展位
- 文档与示例：典型调用流程、配置示例、与 manager 的关系说明

## Impact
- 影响能力：命令行使用存档能力；面向 CI/自动化流程
- 影响代码：
  - 新增 [cmd/arctl](file:///d:/workdir/leon/cocom/cmd/arctl) 可执行入口
  - 复用 [executor.go](file:///d:/workdir/leon/cocom/pkg/archive/manager/executor.go) 中打包注册流程（如 helper.ArchiveAndRegister）
  - 复用 pkg/storage 的复制/校验能力（参考 [migrate.go](file:///d:/workdir/leon/cocom/pkg/storage/migrate.go)）
  - 可能需要在 pkg/archive/manager 暴露少量便捷方法/类型以简化 CLI 调用（非破坏性）

## ADDED Requirements
### Requirement: 打包并注册（pack）
系统 SHALL 基于输入源目录与目标路径执行打包，并在成功后注册元数据。
#### Scenario: 成功
- WHEN 执行 `arctl pack --src <dir> --dest <archive> [--config <file>]`
- THEN 生成归档文件并调用 manager 注册，输出归档 ID/路径/校验信息（支持 --output=json）

### Requirement: 解包（unpack）
系统 SHALL 支持将归档解包到指定目录，校验完整性后完成写出。
#### Scenario: 成功
- WHEN 执行 `arctl unpack --src <archive> --out <dir>`
- THEN 解包完成并校验哈希/大小，输出结果摘要

### Requirement: 查询（query）
系统 SHALL 支持基于 ID/名称/标签等查询元数据与定位信息。
#### Scenario: 成功
- WHEN 执行 `arctl query --id <id>` 或 `arctl query --name <name> [--limit N]`
- THEN 返回匹配的存档基本信息、存储位置列表与健康状态，可选择 JSON 输出

### Requirement: 多存储备份（backup）
系统 SHALL 支持将存档复制到多个 Storage 后端并更新元数据。
#### Scenario: 成功
- WHEN 执行 `arctl backup --id <id> --to <storage://...> [--verify]`
- THEN 完成复制并校验（如 ETag/Hash），更新副本位置与健康状态

### Requirement: 一致性检查（check）
系统 SHALL 对已注册存档的各副本执行一致性检查并生成报告。
#### Scenario: 成功
- WHEN 执行 `arctl check --id <id> [--all]`
- THEN 输出每个副本的 Stat/ETag/可选 Hash 结果与综合健康结论，可更新最后校验时间

## MODIFIED Requirements
### Requirement: manager 便捷方法暴露
为简化 CLI 使用，系统 SHOULD 暴露或新增少量便捷方法（如通过 ID 快速获取定位与元数据的组合读取）。不改变既有接口语义，保持后向兼容。

## REMOVED Requirements
无


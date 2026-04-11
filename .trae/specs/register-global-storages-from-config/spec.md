# 根据配置注册全局 Storage Spec

## Why
为统一管理与复用对象存储，在应用启动时应能基于配置一次性注册“已知”的全局存储，便于各模块通过名称检索（storage.Get）而无需重复拼装路径或实例化驱动。

## What Changes
- 在 pkg/storage 新增按配置批量注册全局存储的能力：提供函数用于读取配置并注册内置（已知）存储
- 支持 LocalFS 后端的最小实现；未来可扩展其他后端
- 定义配置约定与默认键；仅在对应路径非空时注册，避免意外覆盖
- 提供单元测试覆盖：配置→注册→storage.Get 成功返回
- 更新配置文档，增加示例

## Impact
- Affected specs: 存储发现与使用、归档副本校验（通过 storage.Get(locator.Backend)）
- Affected code: pkg/storage/*（新增配置读取与注册逻辑）、docs/config.md（文档补充）；后续调用方在合适位置调用注册函数完成初始化

## ADDED Requirements
### Requirement: 基于配置注册已知全局存储
系统 SHALL 提供一个入口以基于配置注册以下已知存储（若配置非空）：
- 名称 gallery → 根目录来自 cocom.storage.path → 类型 localfs
- 名称 archive → 根目录来自 cocom.archive.path → 类型 localfs
- 名称 archive-temp → 根目录来自 cocom.archive.temp_path → 类型 localfs

并且系统 SHOULD 预留可选扩展项 storage.backends 以注册额外存储项（name、type、root），当前仅支持 type=localfs，未知类型忽略并记录告警（测试中以忽略处理）。

#### Scenario: 成功注册与获取
- WHEN 配置存在 cocom.archive.path 且调用注册入口
- THEN storage.Get("archive") 返回 localfs 实例，URI 形如 localfs://archive/...

## MODIFIED Requirements
无

## REMOVED Requirements
无


# Tasks
- [x] 任务 1：在 pkg/storage 增加基于 Viper 的注册入口（最小实现）
  - [x] 子任务 1.1：定义函数 RegisterKnownFromViper()，基于 viper 读取已知键并注册：
    - gallery ← viper["cocom.storage.path"]
    - archive ← viper["cocom.archive.path"]
    - archive-temp ← viper["cocom.archive.temp_path"]
  - [x] 子任务 1.2：容错策略：空路径不注册；重复注册返回已存在错误并跳过
  - [x] 子任务 1.3：实现可选扩展解析 storage.backends（当前仅 type=localfs）

- [x] 任务 2：单元测试与验证
  - [x] 子任务 2.1：设置 viper 临时配置，调用 RegisterKnownFromViper()，断言 storage.Get 可取到对应实例
  - [x] 子任务 2.2：覆盖重复注册与空值不注册场景
  - [x] 子任务 2.3：覆盖 storage.backends 扩展（localfs）场景

- [x] 任务 3：文档更新
  - [x] 子任务 3.1：在 docs/config.md 增加 “存储注册” 配置段与示例
  - [x] 子任务 3.2：说明当前仅支持 localfs，后续可扩展其他后端


# Task Dependencies
- [任务 2] 依赖 [任务 1]
- [任务 3] 可与 [任务 1] 并行，但需在合入前更新

# 存档与存储安全审计与测试完善 Spec

## Why
当前 pkg/archive、pkg/archive/manager 与 pkg/storage 已具备核心功能，但需要一次系统性的功能完整性与安全审计，并通过覆盖边界与常规用例的测试用例来固化质量，防止路径穿越、并发与幂等问题、校验缺失等隐患。

## What Changes
- 审计与威胁建模：针对上述三个包进行接口契约、错误语义、并发/幂等性、原子性、一致性与路径/权限安全的全面检查，并形成文档。
- 安全加固：对发现的风险进行修复与加固，尤其是 LocalFS 的路径规范化/越界与软链逃逸、临时文件原子替换、权限与覆写策略、错误映射一致性。
- 测试体系扩充：新增覆盖边界与常规用例的单元与集成测试（空文件/大文件/特殊文件名/Unicode/极端路径/并发/中断与重试/不一致副本/策略执行等），引入属性测试与模糊测试用于路径与 URI 解析。
- 验证与门禁：集成 -race 与覆盖率门槛，确保 ./... 测试稳定通过；为关键函数添加 fuzz 测试目标，持续运行确保无崩溃与不变量违背。
- 文档：输出安全审计报告与测试矩阵，更新 README/SECURITY 注意事项与使用建议。
- 不引入对外行为变更，仅修复缺陷与强化契约；如需接口最小调整，将在 MODIFIED Requirements 中列明。

## Impact
- 受影响的能力：存档打包后元数据一致性、复制与保留策略可靠性、对象存储操作安全性与可追踪性。
- 受影响的代码：pkg/archive、pkg/archive/manager、pkg/storage（含 localfs 与迁移工具）、相关测试与文档。
- 相关既有 Spec：implement-archive-manager、introduce-storage-pkg、add-storage-uri-and-osroot（在其基础上做安全/测试增强）。

## ADDED Requirements
### Requirement: 存储路径与 URI 安全
系统应严格约束 LocalFS 在 os.Root 沙箱内运行，拒绝路径穿越与软链越界；所有对外暴露位置应可由 URI 还原且规范化。

#### Scenario: 拒绝越界访问
- WHEN 调用 Storage.Put/Get/List/Delete 等并传入包含 ../、绝对路径或软链逃逸的 key
- THEN 操作被拒绝并返回明确的安全错误，且无文件在沙箱外被访问或修改

### Requirement: 原子写与一致性
在本地存储落盘采用临时文件 + 原子替换策略，保证崩溃后不产生半写文件；IndexStore 的 CRUD 遵循幂等与最小并发安全约束。

#### Scenario: 写入中断
- WHEN 写入过程中程序异常中断
- THEN 不产生部分写入可见文件；后续重试可成功，IndexStore 状态一致

### Requirement: 校验与错误映射
提供可选内容校验（尺寸与可选哈希/ETag）与一致的错误映射（NotFound/Conflict/Transient/PolicyViolation 等）。

#### Scenario: 校验失败
- WHEN 复制或读取时发现尺寸/哈希不一致
- THEN 返回校验失败错误并在报告中标注副本健康异常

### Requirement: 并发与幂等
Manager 的 Register/Put/Delete/Replicate 等操作在重复请求下幂等；在基本并发下无数据竞争；策略执行遵循配置且可回溯。

#### Scenario: 重复注册与并发 Put/Delete
- WHEN 同一存档 ID 被重复注册或在 Put/Delete 并发发生
- THEN 最终状态确定、无重复记录或幽灵记录，竞态由最小锁或 CAS 序列化避免损坏

## MODIFIED Requirements
### Requirement: 统一错误语义（细化）
在既有错误类型基础上，统一跨包错误映射规范与文档，保证调用方可依赖错误分类进行重试与降级。

## REMOVED Requirements
（无）


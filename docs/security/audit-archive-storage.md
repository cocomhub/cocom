# pkg/archive 与 pkg/storage 安全与完整性审计

## 范围
- pkg/storage（含 localfs、migrate、URI/Path）
- pkg/archive/manager（IndexStore、Register/Put/Get/List/Delete、Replicate、Check/健康校验、Retention、Executor）
- pkg/archive（single/double 归档流程、文件列表与并发控制）

## 结论摘要
- 路径与沙箱
  - LocalFS 通过 os.OpenRoot 在根目录沙箱内操作，并对 key 执行 Clean 与越权阻断（“..”前缀）。
  - 建议：在测试中覆盖 Windows 盘符、UNC、绝对路径等输入，确保 Path/URI 与 LocalFS 双重阻断有效。
- 原子性
  - LocalFS.Put 原实现直接写最终文件；崩溃风险导致半写文件。
  - 已改为“临时文件 + rename”的原子提交策略（最佳努力，Windows 上先删除目标再重命名）。
- 符号链接
  - List/Stat 采用 root.Stat 可能跟随符号链接；os.Root 语义可防越界，但对指向根外的链接未做显式分类。
  - 建议：测试覆盖软链指向根外的行为，确保越界读取被拒绝；必要时 Lstat 区分策略。
- 错误映射
  - 存储层缺少统一错误分类（NotFound/Exists/Permission/Transient）；目前透传 os 错误。
  - 建议：逐步引入标准化错误，便于管理层可预期处理。
- 校验流程
  - 管理层 Check 阶段：原实现将 checksum 值用作 Storage.Get 的对象 key，逻辑错误。
  - 已修复：Primary 通过直接读 meta.Path 计算；副本通过 locator.Key 与后端读取校验。

## 风险分级
- 高：LocalFS 非原子写导致半写（已修复）；健康检查 key/value 混用（已修复）。
- 中：符号链接策略未明确（依赖 os.Root，测试需加强）。
- 中：覆盖写并发控制不足（上层或通过 ETag/If-Match 策略增强）。
- 低：错误分类不统一（建议逐步对齐）。

## 建议与后续
- 扩充 单元/集成/属性/模糊 测试，覆盖 0 字节、大文件、特殊名、越界、并发等。
- 启用 -race 与覆盖率门槛，保障关键路径稳定。

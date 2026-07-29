BASE_BEFORE_TASK1: 65b51c1c5cc966fa86b12401a075b0869575d0c5

# Task 1: Config 核心层深度审查报告

## 审查范围
- `internal/config/manager.go` — SetDefault 死键检查
- `internal/config/config.go` — Init 流程检查
- `internal/config/types.go` — 类型定义完整性检查
- `cmd/server/config.go` — 可移除性评估
- `cmd/root.go` — 初始化链检查
- `internal/rootcli/rootcli.go` — 初始化链检查

---

## 审查项 1: `http.enable_proxy` 和 `http.proxy` 死键检查

### 发现: 死键 ✓

**搜索结论**: 全库搜索 `http.enable_proxy` 和 `http.proxy`，仅发现以下命中：

| 位置 | 类型 |
|------|------|
| `internal/config/manager.go:288-291` | SetDefault 定义 |
| `internal/config/config_keys_test.go:147-148` | 测试用例 |
| `cocom-gen.yaml:338-342` | 注释说明 |
| `.superpowers/` 和 `docs/` 下的设计文档 | 历史设计文档 |

**没有任何 `.go` 代码通过 `viper.Get("http.enable_proxy")` 或 `viper.Get("http.proxy")` 读取这些键。**

原设计文档 `docs/superpowers/specs/2026-06-26-config-migration-fix-design.md` 明确指出：
> `http.*` 键（`http.enable_proxy`、`http.proxy`）仅通过 viper 全局直接读取，不走 Config 结构体，无需迁移

但实际搜索发现 **没有任何代码读取这些键**。`pkg/download/downloader.go` 中的 `EnableProxy`/`ProxyURL` 字段是通过 `DownloaderConfig` 结构体（`DownloaderConfig.EnableProxy` / `DownloaderConfig.ProxyURL`）读取的，走的是 `download.Config` 结构体 → `download.enableProxy` 和 `download.proxyURL` 键路径，**不是** `http.enable_proxy` / `http.proxy`。

**结论: 死键，可以安全移除。**

### 修复: 已移除

1. `internal/config/manager.go`: 移除 `http.enable_proxy` 和 `http.proxy` 的 SetDefault 和 config-doc 注释（2 行）
2. `internal/config/config_keys_test.go`: 移除 `http.enable_proxy` 和 `http.proxy` 的测试用例（2 行）

---

## 审查项 2: `cmd/server/config.go` 可移除性评估

### 发现: 可安全移除 ✓

**文件内容**: 仅含 `package server` 声明 + `config-doc` 注释 + `Default:` 注释。无代码、无 init()、无 import。

**`config-doc-gen` 工具行为**: 扫描所有 `.go` 文件中的 `// config-doc:` 注释。`cmd/server/config.go` 中的 `config-doc` 注释与 `internal/config/manager.go` 中的 `config-doc` 注释完全重复。

**结论**: 所有 `config-doc` 注释已完整迁移到 `manager.go`，`cmd/server/config.go` 可安全移除。`config-doc-gen` 仍能从 `manager.go` 读取到所有配置文档。

---

## 审查项 3: 残留的空 `init()` 函数检查

### 发现: 找到了 1 处可移除的空 `init()` ✓

| 文件 | 状态 | 说明 |
|------|------|------|
| `cmd/server/internal/mongo/mongo.go:44` | **已移除** | 空 `func init() {}`，注释说"保留以保持 import side-effect 兼容"，但没有任何代码以 `_` 方式 import 该包。所有 15 处 import 均为具名 import（如 `mongo.Wrap`），不依赖 init() 的 side-effect。 |

---

## 审查项 4: `types.go` 中未使用的类型/字段检查

### 发现: 无未使用字段 ✓

`Config` 结构体及其所有嵌套类型均有代码引用，详见审查报告。

---

## 审查项 5: Config 结构体字段与 SetDefault 匹配检查

### 发现: 字段全覆盖，补全了 config-doc 注释 ✓

**修复**: 补全了 `server.scheduler.*` 子键的 `config-doc` 注释（之前仅有 `server.scheduler.enabled` 有注释，其余 18 个键缺失）。

---

## 汇总

| 审查项 | 状态 | 说明 |
|--------|------|------|
| 1. `http.enable_proxy`/`http.proxy` 死键 | ✅ 已修复 | 移除 SetDefault + 测试用例 |
| 2. `cmd/server/config.go` 可移除 | ✅ 可移除 | 所有 config-doc 已迁移至 manager.go |
| 3. 空 `init()` 函数 | ✅ 已修复 | 移除 `cmd/server/internal/mongo/mongo.go` 的空 init |
| 4. types.go 未使用类型/字段 | ✅ 无 | 所有字段均有代码引用 |
| 5. Config 字段与 SetDefault 匹配 | ✅ 已修复 | 补全 server.scheduler.* 的 config-doc 注释 |
| Init 流程 | ✅ 正确 | 优先级链正确，初始化顺序合理 |
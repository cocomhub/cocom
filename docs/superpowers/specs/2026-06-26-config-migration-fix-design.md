# 配置迁移修复与 cocom-gen.yaml 生成设计

日期: 2026-06-26
状态: 设计完成，待实现

## 背景

commit 3b852db 发现 MongoDB 初始化 bug：`pkg/mongowrap/mongo.go` 的 `Client()` 在 `Init()` 未调用时以零值 `Config{}` 回退，而 `sync.Once` 确保第一个调用者永久获胜。若工厂函数的 `mongowrap.DB()` 先于显式 `mongowrap.Init(config.Get().Mongo)` 触发，则正确的 Mongo 配置永远不会被应用。

此 bug 暴露了 Viper→struct 配置迁移中的深层问题：
1. **tools 不调用 `config.Init()`** — pixm/arctl 的 `config.Get()` 返回硬编码默认值
2. **`mongowrap.Client()` 零值回退** — 已修复调用顺序，但 `Client()` 中残留的零值回退仍是隐患
3. **双键体系混乱** — `cocom.archive.*` 和 `archive.*` 字段重复，部分 YAML 键为孤立键
4. **cocom-new.yaml 不可用** — 包含孤立键、旧格式键，与最新 Config 结构体不匹配

## 目标

1. 修复影响功能正确性的 bug（tools config.Init 缺失、mongowrap 零值回退）
2. 按"cocom 专属放 `cocom.*`，可复用放 `archive.*`"原则重分配配置键
3. 生成两份干净的 YAML：开发版最小配置 + 完整生产配置
4. 更新代码中 `archivecli/commands.go` 的回退逻辑，不再引用已删除的键

## 代码修复

### 修复 1: tools 增加 `config.Init` 调用

**文件**: `tools/pixm/main.go`、`tools/arctl/main.go`

**问题**: 两个工具的 `cobra.OnInitialize` 中缺少 `config.Init`，导致 `config.Manager` 的内部 viper 实例从未从全局 viper 同步 YAML 文件和环境变量。`config.Get()` 返回的值完全来自 `setDefaultsOn()` 的硬编码默认值。

**修复**: 在 `init()` 函数的 `cobra.OnInitialize` 链中插入 `config.Init`：

```
// tools/pixm/main.go
cobra.OnInitialize(initConfig, config.Init, initArchiveManager)

// tools/arctl/main.go
cobra.OnInitialize(initConfig, config.Init, initArchiveManager)
```

这与 `cmd/root.go` 的模式一致（`rootcli.InitConfig → config.Init → initLogging → initArchiveManager`）。

### 修复 2: `mongowrap.Client()` 零值回退加固

**文件**: `pkg/mongowrap/mongo.go`

**问题**: 第 84-89 行，`Client()` 在 `onceInit` 未触发时以零值 `Config{}` 调用 `Init()`。commit 3b852db 通过调整调用顺序规避了此问题，但 `Client()` 中的零值回退仍是定时炸弹。`sync.Once.Do` 不可用于检测"是否已被调用过"（无对应的 `HasDone()` 方法），因此需要额外的状态标记。

**修复**: 引入 `initialized` atomic 标记替代零值回退：

```go
var initialized atomic.Bool

func Init(cfg Config) error {
    onceInit.Do(func() {
        initialized.Store(true)
        initEngine(cfg)
    })
    return initErr
}

func Client() (*mongo.Client, error) {
    if !initialized.Load() {
        return nil, errors.New("mongowrap: Init() must be called before Client()")
    }
    return client, initErr
}
```

同时移除包级变量 `initCfg`（不再需要）。

### 修复 3: `archivecli/commands.go` 回退逻辑简化

**文件**: `internal/archivecli/commands.go`

**问题**: `archiveConfig()` 中 `password`/`cmd` 有 `Cocom.Archive.* → Archive.*` 两级回退。按新原则，`password`/`cmd` 只在 `cocom.archive.*` 下，移除对 `Archive.Password`/`Archive.Cmd` 的回退：

```go
// 修改前（第 436-449 行）
password := strings.TrimSpace(cfg.Cocom.Archive.Password)
if password == "" {
    password = strings.TrimSpace(cfg.Archive.Password)
}
...
cmdPath := util.FirstNonEmpty(cfg.Cocom.Archive.Cmd, cfg.Archive.Cmd, "7z")

// 修改后
password := strings.TrimSpace(cfg.Cocom.Archive.Password)
...
cmdPath := util.FirstNonEmpty(cfg.Cocom.Archive.Cmd, "7z")
```

### 修复 4: `archive.root_dir` 添加 SetDefault

**文件**: `internal/config/manager.go`

**问题**: `Archive.RootDir` 在 `types.go` 有字段定义但 `setDefaultsOn()` 中无对应默认值，是唯一缺少默认值的 `archive.*` 键。虽然 `archivecli/commands.go:52` 有空值回退逻辑，但显式设置默认值更健壮：

```go
v.SetDefault("archive.root_dir", "")
```

**注意**: `archive.password`、`archive.cmd`、`archive.replicate`、`archive.algorithm.*` 的默认值在 `manager.go` 中**保留不变**，确保使用旧版 YAML 配置（只有 `archive.*` 没有 `cocom.archive.*`）的存量部署不受影响。新的 `cocom-gen.yaml` 中这些键不再出现，引导用户使用 `cocom.archive.*`。

### 修复 5: `archivecli/commands.go` 回退逻辑更新

**文件**: `internal/archivecli/commands.go`

**说明**: 此修改有破环性。`archiveConfig()` 中移除对 `Archive.Password`/`Archive.Cmd` 的回退。存量部署若只在 `archive.password`（旧键）设置了值而未在 `cocom.archive.password`（新键）设置，升级后将回退到默认值而非读取旧键值。升级指南应在 `cocom_v0.0.57_to_a8d21.txt` 中说明此变更。

## 配置键重分配

### 原则

- **`cocom.*`**: cocom 项目专属配置，不与其他项目共享
- **`archive.*`**: 可复用的归档基础设施，未来其他项目可直接引用

### cocom.archive.* (cocom 专属)

| 键 | 说明 | 此前位置 |
|---|------|---------|
| `cocom.archive.path` | 存档路径 | 不变 |
| `cocom.archive.temp_path` | 临时存档路径 | 不变 |
| `cocom.archive.password` | 存档密码 | 原 `archive.password` 也有重复 |
| `cocom.archive.cmd` | 7z 命令路径 | 原 `archive.cmd` 也有重复 |
| `cocom.archive.replicate` | 是否默认复制到远端 | 原 `archive.replicate` 也有重复 |
| `cocom.archive.algorithm.single.concurrency` | 单层加密并发 | 原 `archive.algorithm.single.concurrency` 也有重复 |
| `cocom.archive.algorithm.double.concurrency` | 双层加密并发 | 原 `archive.algorithm.double.concurrency` 也有重复 |

### archive.* (可复用)

| 键 | 说明 |
|---|------|
| `archive.root_dir` | 归档根目录 |
| `archive.manager.algorithm` | 归档管理器算法 |
| `archive.manager.meta_record_file_list` | 是否记录文件列表 |
| `archive.manager.replicates` | 远端副本目标列表 |
| `archive.manager.index.type` | 索引类型 |
| `archive.manager.index.file_store_name` | 文件索引存储名称 |
| `archive.manager.index.file_store_prefix` | 文件索引存储前缀 |
| `archive.manager.index.mongo_database` | MongoDB 索引数据库 |
| `archive.manager.index.mongo_collection` | MongoDB 索引集合 |
| `archive.manager.index.mongo_prefix` | MongoDB 索引前缀 |
| `archive.manager.index.mongo_id_field` | MongoDB 索引 ID 字段 |
| `archive.manager.index.mongo_name_field` | MongoDB 索引名称字段 |

### 删除的孤立键

| 键 | 原因 |
|---|------|
| `archive.password` | 已有 `cocom.archive.password` |
| `archive.cmd` | 已有 `cocom.archive.cmd` |
| `archive.replicate` | 已有 `cocom.archive.replicate` |
| `archive.algorithm.single.concurrency` | 已有 `cocom.archive.algorithm.single.concurrency` |
| `archive.algorithm.double.concurrency` | 已有 `cocom.archive.algorithm.double.concurrency` |
| `archive.path` | 无结构体字段，孤立键 |
| `archive.temp_path` | 无结构体字段，孤立键 |
| `archive.try_replicate` | 无结构体字段，孤立键 |
| `storage.backends` | 移到 `cocom.storage.backends` |
| `admin.*` | 移到 `server.admin.*` |
| `arctl.*` | 无结构体字段，孤立键 |

### 保持不变的键

`server.*`、`log.*`、`mongo.*`、`comic.*`、`download.*`、`recommend.*`、`client.*`、`http.*` 的键路径和结构保持不变。

## cocom-gen.yaml 生成两份配置

### 开发版最小配置 (`cocom-gen-dev.yaml`)

只包含启动服务必需的最小配置项，所有未列出项使用内置默认值。目标场景：本地开发，MongoDB + 本地文件系统。

### 完整生产配置 (`cocom-gen.yaml`)

包含所有配置项的完整参考文件，每个键附带中文注释说明用途和默认值。按新的键分配原则组织。旧 `cocom-new.yaml` 中的孤立键全部移除。

## 审查发现的其他问题（本次不改，记录待后续处理）

| # | 键 | 问题 | 建议 |
|---|-----|------|------|
| 1 | `comic.verify.concurrent` | 有定义和默认值但无代码消费 | 后续集成到 verify CLI 或删除 |
| 2 | `comic.verify.task_buffer_size` | 同上 | 同上 |
| 3 | `log.disableTraceID` | 有定义和默认值但 logging 未使用 | 后续实现或删除 |
| 4 | `probe_comic.limit/max_conn/backends` | 结构体有字段但无默认值、无消费 | 后续实现或删除字段 |
| 5 | `cocoma_archiver.cid_regex` | 有默认值但调度器未传递 | 后续集成到 RunOnce |
| 6 | `server.ratelimit.burst` | 传入中间件但 `_ = burst` 忽略 | 评估是否需要 |

## 配置键消费路径审查结论

经过逐键审查 56 个配置键的 mapstructure 标签 → SetDefault → 代码读取三环节，除上述已列入修复的 4 个问题外：
- 所有 `cocom.*`、`archive.*`、`server.*`、`log.*`、`mongo.*`、`comic.*`、`download.*`、`recommend.*`、`client.*` 的键在三环节中一致
- `http.*` 键（`http.enable_proxy`、`http.proxy`）仅通过 viper 全局直接读取，不走 Config 结构体，无需迁移

## 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `tools/pixm/main.go` | 修改 | 增加 `config.Init` |
| `tools/arctl/main.go` | 修改 | 增加 `config.Init` |
| `pkg/mongowrap/mongo.go` | 修改 | `Client()` 零值回退加固（atomic.Bool + 移除 `initCfg`） |
| `internal/archivecli/commands.go` | 修改 | 移除 `Archive.Password`/`Archive.Cmd` 回退（有破环性，需在升级指南说明） |
| `internal/config/manager.go` | 修改 | 添加 `archive.root_dir` 默认值 |
| `internal/config/config_keys_test.go` | 修改 | 更新配置键测试用例 |
| `cocom-gen-dev.yaml` | 新建 | 开发版最小配置 |
| `cocom-gen.yaml` | 新建 | 完整生产配置 |
| `cocom-new.yaml` | 不修改 | 保留作为历史参考 |
| `cocom.yaml` | 不修改 | 保留作为历史参考 |
| `cocom_v0.0.57_to_a8d21.txt` | 修改 | 补充本次配置键迁移的破环性变更说明 |

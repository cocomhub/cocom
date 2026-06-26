# 配置迁移修复与 cocom-gen.yaml 生成 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 修复 Viper→struct 配置迁移遗留的 4 个 bug，按"cocom 专属 / archive 可复用"原则重分配配置键，生成干净的开发版和生产版 YAML 配置。

**架构：** 5 个代码修复分散在 5 个文件，2 个新 YAML 文件，1 个测试更新，1 个文档更新。所有修改独立无依赖，可合并为一个 commit。

**技术栈：** Go 1.26, Viper, atomic.Bool, YAML

---

### 任务 1：tools/pixm 和 tools/arctl 增加 `config.Init` 调用

**文件：**
- 修改：`tools/pixm/main.go:41-44`
- 修改：`tools/arctl/main.go:39-42`

- [ ] **步骤 1：修改 pixm/main.go 的 cobra.OnInitialize**

当前代码（第 41-44 行）：
```go
	cobra.OnInitialize(
		initConfig,
		initArchiveManager,
	)
```

修改为：
```go
	cobra.OnInitialize(
		initConfig,
		config.Init,
		initArchiveManager,
	)
```

- [ ] **步骤 2：修改 arctl/main.go 的 cobra.OnInitialize**

当前代码（第 39-42 行）：
```go
	cobra.OnInitialize(
		initConfig,
		initArchiveManager,
	)
```

修改为：
```go
	cobra.OnInitialize(
		initConfig,
		config.Init,
		initArchiveManager,
	)
```

- [ ] **步骤 3：编译验证**

```bash
cd D:\workdir\leon\cocomhub\cocom
go build ./tools/pixm/...
go build ./tools/arctl/...
```

预期：两者均编译成功，无错误。

- [ ] **步骤 4：Commit**

```bash
git add tools/pixm/main.go tools/arctl/main.go
git commit -m "fix(tools): 补充 pixm/arctl 缺失的 config.Init 调用

tools 的 cobra.OnInitialize 链中缺少 config.Init，导致
config.Manager 内部 viper 实例从未从全局 viper 同步 YAML
文件和环境变量，config.Get() 始终返回 setDefaultsOn() 的
硬编码默认值。

在此链中插入 config.Init，与 cmd/root.go 的模式一致。"
```

---

### 任务 2：mongowrap.Client() 零值回退加固

**文件：**
- 修改：`pkg/mongowrap/mongo.go`

- [ ] **步骤 1：修改 pkg/mongowrap/mongo.go**

当前代码（第 6-23, 77-97 行）：
```go
import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/cocomhub/cocom/pkg/logging"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client   *mongo.Client
	initErr  error
	onceInit sync.Once
	initCfg  Config
)

func init() {
}
```

将 import 和包级变量修改为：
```go
import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cocomhub/cocom/pkg/logging"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client      *mongo.Client
	initErr     error
	onceInit    sync.Once
	initialized atomic.Bool
)
```

修改 `Init` 函数（第 77-82 行）：
```go
// 修改前：
func Init(cfg Config) error {
	onceInit.Do(func() {
		initEngine(cfg)
	})
	return initErr
}

// 修改后：
func Init(cfg Config) error {
	onceInit.Do(func() {
		initialized.Store(true)
		initEngine(cfg)
	})
	return initErr
}
```

修改 `Client` 函数（第 84-89 行）：
```go
// 修改前：
func Client() (*mongo.Client, error) {
	if err := Init(initCfg); err != nil {
		return nil, err
	}
	return client, nil
}

// 修改后：
func Client() (*mongo.Client, error) {
	if !initialized.Load() {
		return nil, errors.New("mongowrap: Init() must be called before Client()")
	}
	return client, initErr
}
```

同时删除空的 `init()` 函数（第 26-27 行）：
```go
func init() {
}
```

- [ ] **步骤 2：检查是否有其他文件引用 `initCfg`**

```bash
cd D:\workdir\leon\cocomhub\cocom
grep -rn "initCfg" pkg/
```

预期：仅 `mongo.go` 中有引用（已在上一步删除）。

- [ ] **步骤 3：编译验证**

```bash
go build ./pkg/mongowrap/...
go vet ./pkg/mongowrap/...
```

预期：编译成功，vet 无警告。

- [ ] **步骤 4：运行 mongowrap 相关测试**

```bash
go test -race -count=1 ./pkg/mongowrap/...
```

预期：测试通过。

- [ ] **步骤 5：Commit**

```bash
git add pkg/mongowrap/mongo.go
git commit -m "fix(mongowrap): 加固 Client() 零值回退，改用 atomic.Bool 检测初始化状态

移除包级 initCfg 变量和 Client() 中对 Init(零值 Config) 的隐式回退。
改为 atomic.Bool 标记，在 Init() 未被调用时返回明确错误，
防止 sync.Once 被零值配置永久占用。"
```

---

### 任务 3：移除 archivecli 中 Archive.Password/Cmd 回退

**文件：**
- 修改：`internal/archivecli/commands.go:434-453`

- [ ] **步骤 1：修改 archiveConfig 函数**

当前代码（第 434-453 行）：
```go
func archiveConfig(id int) (archive.Config, error) {
	cfg := config.Get()
	password := strings.TrimSpace(cfg.Cocom.Archive.Password)
	if password == "" {
		password = strings.TrimSpace(cfg.Archive.Password)
	}
	if password == "" {
		return archive.Config{}, errors.New("归档密码未配置：archive.password 为空")
	}
	tmpDir, tmpErr := rootcli.TempDir()
	if tmpErr != nil {
		return archive.Config{}, fmt.Errorf("获取临时目录失败：%w", tmpErr)
	}
	return archive.Config{
		ID:       id,
		CmdPath:  util.FirstNonEmpty(cfg.Cocom.Archive.Cmd, cfg.Archive.Cmd, "7z"),
		Password: password,
		TempDir:  tmpDir,
	}, nil
}
```

修改为：
```go
func archiveConfig(id int) (archive.Config, error) {
	cfg := config.Get()
	password := strings.TrimSpace(cfg.Cocom.Archive.Password)
	if password == "" {
		return archive.Config{}, errors.New("归档密码未配置：cocom.archive.password 为空")
	}
	tmpDir, tmpErr := rootcli.TempDir()
	if tmpErr != nil {
		return archive.Config{}, fmt.Errorf("获取临时目录失败：%w", tmpErr)
	}
	return archive.Config{
		ID:       id,
		CmdPath:  util.FirstNonEmpty(cfg.Cocom.Archive.Cmd, "7z"),
		Password: password,
		TempDir:  tmpDir,
	}, nil
}
```

- [ ] **步骤 2：编译验证**

```bash
go build ./internal/archivecli/...
go vet ./internal/archivecli/...
```

预期：编译成功。

- [ ] **步骤 3：Commit**

```bash
git add internal/archivecli/commands.go
git commit -m "refactor(archivecli): 移除 Archive.Password/Cmd 二级回退

按配置键重分配原则，password/cmd 仅从 cocom.archive.* 读取，
不再回退到 archive.*。这是破坏性变更，存量部署若仅在 archive.password
（旧键）设置了值，升级后需迁移到 cocom.archive.password。"
```

---

### 任务 4：添加 archive.root_dir 默认值

**文件：**
- 修改：`internal/config/manager.go`

- [ ] **步骤 1：在 setDefaultsOn 中添加 archive.root_dir 默认值**

在 `internal/config/manager.go` 的 `setDefaultsOn()` 函数中，`archive.manager.*` 默认值区域（第 230 行前）添加一行：

```go
	// === 从 pkg/archive/manager/config.go init() 移入 ===
	// config-doc: archive.root_dir 归档根目录
	v.SetDefault("archive.root_dir", "")
	// config-doc: archive.manager.algorithm 归档算法 (single/double)
	v.SetDefault("archive.manager.algorithm", string(archive.TypeDouble))
```

即在第 230 行（`// === 从 pkg/archive/manager/config.go init() 移入 ===`）注释后、第 231 行（`// config-doc: archive.manager.algorithm`）之前插入 `v.SetDefault("archive.root_dir", "")`。

- [ ] **步骤 2：编译验证**

```bash
go build ./internal/config/...
```

预期：编译成功。

- [ ] **步骤 3：运行配置测试**

```bash
go test -race -count=1 ./internal/config/...
```

预期：测试通过（新增的键没有测试用例，但旧有测试不应受影响）。

- [ ] **步骤 4：Commit**

```bash
git add internal/config/manager.go
git commit -m "fix(config): 补充 archive.root_dir 缺失的默认值

Archive.RootDir 在 types.go 有字段定义但 setDefaultsOn() 中
无对应默认值，是唯一缺少默认值的 archive.* 键。显式设置空字符串
默认值以保持与其他 archive.* 键的一致。"
```

---

### 任务 5：更新 config_keys_test.go 测试用例

**文件：**
- 修改：`internal/config/config_keys_test.go`

- [ ] **步骤 1：添加缺失的测试键用例**

在 `keyTestCases` 表中，`// === archive.* ===` 区域（第 32 行之后）和 `// === cocom.* ===` 区域（第 37 行之后）添加缺失的键：

在第 32 行 `{Key: "archive.algorithm.double.concurrency", ...}` 之后添加：
```go
	{Key: "archive.root_dir", Name: "archive root dir", DefaultValue: "", OverrideVal: "/tmp/root"},
```

在第 37 行 `{Key: "cocom.archive.temp_path", ...}` 之后添加：
```go
	{Key: "cocom.archive.password", Name: "cocom archive password", DefaultValue: "archive@123456", OverrideVal: "secret123"},
	{Key: "cocom.archive.cmd", Name: "cocom archive cmd", DefaultValue: "7z", OverrideVal: "/opt/7z"},
	{Key: "cocom.archive.replicate", Name: "cocom archive replicate", DefaultValue: false, OverrideVal: true},
	{Key: "cocom.archive.algorithm.single.concurrency", Name: "cocom algo single", DefaultValue: 4, OverrideVal: 8},
	{Key: "cocom.archive.algorithm.double.concurrency", Name: "cocom algo double", DefaultValue: 4, OverrideVal: 8},
```

- [ ] **步骤 2：运行测试验证**

```bash
go test -race -count=1 -run "TestDefaults_AllKeys|TestGetStruct_AllKeys|TestOverride_YAMLFile|TestOverride_EnvVar" ./internal/config/...
```

预期：所有测试通过。

- [ ] **步骤 3：Commit**

```bash
git add internal/config/config_keys_test.go
git commit -m "test(config): 补充 cocom.archive.* 和 archive.root_dir 测试用例

keyTestCases 表中新增 6 个配置键的默认值和覆盖测试：
cocom.archive.password/cmd/replicate/algorithm.single.concurrency/
algorithm.double.concurrency 以及 archive.root_dir。
确保新增键的 mapstructure→SetDefault→viper.Get 三环节一致。"
```

---

### 任务 6：创建开发版最小配置 cocom-gen-dev.yaml

**文件：**
- 新建：`cocom-gen-dev.yaml`

- [ ] **步骤 1：创建文件**

```yaml
# cocom 开发版最小配置 — 可直接用于本地开发
# 启动命令: ./build/cocom server --config ./cocom-gen-dev.yaml
# 所有未列出的配置项使用内置默认值，详见 cocom-gen.yaml

# --- MongoDB 连接（本地开发） ---
mongo:
  host: localhost:27017
  user: cocom
  password: cocom123
  database: cocom
  authSource: cocom

# --- 服务器监听地址 ---
server:
  listen:
    http:
      addr: 0.0.0.0:8080
```

- [ ] **步骤 2：Commit**

```bash
git add cocom-gen-dev.yaml
git commit -m "feat(config): 新增开发版最小配置 cocom-gen-dev.yaml

仅包含本地开发必需的 MongoDB 连接和 HTTP 监听地址，
其他配置项使用内置默认值。"
```

---

### 任务 7：创建完整生产配置 cocom-gen.yaml

**文件：**
- 新建：`cocom-gen.yaml`

- [ ] **步骤 1：创建文件**

基于 `cocom-new.yaml` 重写，按新原则重新组织，移除所有孤立键。

```yaml
# =============================================================================
# cocom 完整生产配置 — 所有配置项参考文件
# =============================================================================
# 使用方式：复制此文件并修改，启动时通过 --config 指定
#    ./build/cocom server --config ./cocom-gen.yaml
#
# 环境变量覆盖：所有键可通过 COCOM_ 前缀环境变量覆盖
#   例：export COCOM_SERVER_LISTEN_HTTP_ADDR=":9090"
#   嵌套键用 _ 分隔，详见下方注释
# =============================================================================

# =============================================================================
# cocom — cocom 项目专属配置（不与其他项目共享）
# =============================================================================
cocom:
  # --- 存储配置 ---
  storage:
    # 图片/画廊本地存储根目录（默认: /data/cocom/data/gallery）
    # 环境变量: COCOM_COCOM_STORAGE_PATH
    path: /data/cocom/data/gallery

    # 附加存储后端列表（默认: 无，不使用远程存储）
    # 每个后端需指定 name（唯一标识）、type（localfs/baidupcs）、metadata
    # 环境变量: 不支持（数组类型）
    # backends:
    #   - name: archive-manager-index
    #     type: localfs
    #     metadata:
    #       root: /data/cocom/index/archive-manager

  # --- 归档配置（cocom 专属） ---
  archive:
    # 归档文件存储根目录（默认: /data/cocom/data/archive）
    # 环境变量: COCOM_COCOM_ARCHIVE_PATH
    path: /data/cocom/data/archive

    # 归档临时文件目录（默认: /data/cocom/data/archive-temp）
    # 环境变量: COCOM_COCOM_ARCHIVE_TEMP_PATH
    temp_path: /data/cocom/data/archive-temp

    # 归档加密密码（默认: archive@123456）
    # 环境变量: COCOM_COCOM_ARCHIVE_PASSWORD
    password: archive@123456

    # 7z 命令路径（默认: 7z）
    # 环境变量: COCOM_COCOM_ARCHIVE_CMD
    cmd: "7z"

    # 是否默认复制归档到远端存储（默认: false）
    # 环境变量: COCOM_COCOM_ARCHIVE_REPLICATE
    replicate: false

    # 归档加密算法并发配置
    algorithm:
      # 单层加密并发数（默认: 4）
      # 环境变量: COCOM_COCOM_ARCHIVE_ALGORITHM_SINGLE_CONCURRENCY
      single:
        concurrency: 4
      # 双层加密并发数（默认: 4）
      # 环境变量: COCOM_COCOM_ARCHIVE_ALGORITHM_DOUBLE_CONCURRENCY
      double:
        concurrency: 4

  # --- 缓存配置 ---
  cache:
    # 缓存清理间隔（默认: 1m）
    cleanInterval: 1m
    # 缓存淘汰间隔（默认: 10m）
    evictionInterval: 10m

# =============================================================================
# archive — 可复用的归档基础设施（未来其他项目可共享）
# =============================================================================
archive:
  # 归档根目录（默认: 空字符串，回退到 data-dir）
  # 环境变量: COCOM_ARCHIVE_ROOT_DIR
  root_dir: ""

  # 归档管理器配置
  manager:
    # 归档算法：single（单层加密）或 double（双层加密）（默认: double）
    algorithm: double

    # 是否在元数据中记录文件列表（默认: false）
    meta_record_file_list: false

    # 远端副本目标存储名称列表（默认: []）
    replicates: []
    #  - archive-backup-baidupcs

    # 索引配置
    index:
      # 索引类型：memory/file/mongo/mongo-cocom/mongo-comicInfo（默认: memory）
      # 注意：mongo 类型需要先配置 mongo.* 连接
      type: memory

      # 文件索引存储名称（type=file 时使用）（默认: archive-manager-index）
      file_store_name: archive-manager-index

      # 文件索引存储前缀（默认: archive/index）
      file_store_prefix: archive/index

      # MongoDB 索引配置（type=mongo* 时使用）
      mongo_database: archiveManager
      mongo_collection: archiveInfo
      mongo_prefix: ""
      mongo_id_field: id
      mongo_name_field: name

# =============================================================================
# mongo — MongoDB 连接配置
# =============================================================================
mongo:
  # MongoDB 服务器地址（默认: localhost:27017）
  # 环境变量: COCOM_MONGO_HOST
  host: localhost:27017

  # MongoDB 用户名（默认: cocom）
  # 环境变量: COCOM_MONGO_USER
  user: cocom

  # MongoDB 密码（默认: cocom123）
  # 环境变量: COCOM_MONGO_PASSWORD
  password: cocom123

  # 业务数据库名（默认: cocom）
  # 环境变量: COCOM_MONGO_DATABASE
  database: cocom

  # 认证数据库名（默认: cocom）
  # 环境变量: COCOM_MONGO_AUTHSOURCE
  authSource: cocom

# =============================================================================
# comic — 漫画业务配置
# =============================================================================
comic:
  # 校验并发数（默认: 10）（注意：当前版本 verify CLI 使用 --workers flag，此配置未实际消费）
  verify:
    concurrent: 10
    task_buffer_size: 100

  # 下载并发限制（默认: 5）
  download:
    maxDownloadSize: 5

  # 漫画 MongoDB 集合配置
  mongo:
    database: cocom
    collections:
      comicInfo: comicInfo
      oneComicInfo: oneComicInfo
      videoInfo: videoInfo
      settings: settings
      custom: custom
      comicTag: comicTag
      tagRelation: tagRelation

# =============================================================================
# server — HTTP 服务配置
# =============================================================================
server:
  # --- 监听配置 ---
  listen:
    http:
      # HTTP 监听地址（默认: 0.0.0.0:8080）
      # 环境变量: COCOM_SERVER_LISTEN_HTTP_ADDR
      # 也可通过 -p 命令行 flag 覆盖端口
      addr: 0.0.0.0:8080

    # HTTPS/TLS 配置（可选）
    tls:
      cert: ""
      key: ""

    # Unix socket 配置（可选）
    unix:
      path: ""

  # --- 管理端点配置 ---
  admin:
    # 管理端点鉴权 token（默认: 空，仅放行 loopback）
    token: ""
    # 是否允许远程访问管理端点（默认: false）
    allow_remote: false

  # --- 优雅关闭 ---
  # 优雅关闭超时时间（默认: 5s）
  shutdown_timeout: 5s

  # --- 调度器 ---
  scheduler:
    # 是否启用调度器（默认: false）
    enabled: false

    # 调度器时区（默认: Local）
    timezone: Local

    # 漫画探测任务
    probe_comic:
      enabled: false
      name: ProbeComic
      cron: "0 */10 * * * *"
      tags:
        - probe
        - comic
      # limit/max_conn/backends 字段已定义但当前版本未实际消费

    # 归档状态检查任务
    archive_status_check:
      enabled: false
      name: ArchiveStatusChecker
      cron: "0 */30 * * * *"
      tags:
        - archive
        - check
      limit: 100
      max_conn: 3
      backends: []

    # Cocoma 归档任务
    cocoma_archiver:
      enabled: false
      cron: "* * * * *"
      limit: 10000
      # CID 正则（注意：当前版本未实际传递给 RunOnce）
      cid_regex: "^(\\d+)\\.cocoma$"
      scan_dir: ""
      archive_dir: ""
      notmatch_dir: ""

  # --- 访问日志 ---
  access_log:
    # 记录访问日志的 URL 前缀模式列表（默认: /debug,/api,/v1,/v2）
    patterns:
      - /debug
      - /api
      - /v1
      - /v2

  # --- CORS 跨域配置 ---
  cors:
    # 是否启用 CORS（默认: false）
    enabled: false
    # 允许的源（默认: *）
    allow_origins: "*"
    # 允许的 HTTP 方法（默认: GET,POST,PUT,DELETE,OPTIONS）
    allow_methods: "GET,POST,PUT,DELETE,OPTIONS"
    # 允许的请求头（默认: *）
    allow_headers: "*"

  # --- Gzip 压缩配置 ---
  gzip:
    # 是否启用 Gzip（默认: false）
    enabled: false
    # 压缩级别 1-9（默认: 1，即 BestSpeed）
    level: 1

  # --- 限流配置 ---
  ratelimit:
    # 是否启用限流（默认: false）
    enabled: false
    # 每秒请求数限制（默认: 10）
    rps: 10
    # 限流突发大小（默认: 20，注意：当前版本中间件忽略此参数）
    burst: 20

# =============================================================================
# log — 日志配置
# =============================================================================
log:
  # 是否启用文件日志（默认: false）
  enableFile: false
  # 日志文件名（默认: app.log）
  filename: app.log
  # 日志文件最大大小 MB（默认: 256）
  maxSize: 256
  # 日志文件保留天数（默认: 30）
  maxAge: 30
  # 保留的旧日志最大数量（默认: 5）
  maxBackups: 5
  # 是否使用本地时间（默认: true）
  localtime: true
  # 是否压缩旧日志（默认: true）
  compress: true

  # 是否启用控制台日志（默认: true）
  enableConsole: true
  # 是否记录调用者信息（默认: true）
  enableCaller: true
  # 是否记录来源 IP（默认: false）
  enableSourceIP: false
  # 是否记录进程 ID（默认: true）
  enablePID: true

  # 文件日志级别（默认: info）
  fileLevel: info
  # 控制台日志级别（默认: debug）
  consoleLevel: debug
  # 文件日志编码格式 json/console（默认: json）
  fileEncoding: json
  # 控制台日志编码格式 json/console（默认: console）
  consoleEncoding: console

  # 应用名称（默认: 空）
  appName: ""
  # 来源网卡名称（默认: eth3）
  sourceEth: eth3
  # 是否禁用 Trace ID（默认: false）（注意：当前版本日志模块未消费此字段）
  disableTraceID: false

# =============================================================================
# download — 下载配置
# =============================================================================
download:
  # 最大并行下载任务数（默认: 10）
  maxRunning: 10
  # 下载文件存放目录（默认: Downloads）
  downloadDir: Downloads

# =============================================================================
# recommend — 推荐配置
# =============================================================================
recommend:
  # 各维度推荐数量上限（默认: 5）
  limit: 5

# =============================================================================
# client — 客户端模式配置（cocom 作为客户端连接其他 cocom 服务时使用）
# =============================================================================
client:
  # 服务端地址（默认: http://localhost:15456）
  server_addr: http://localhost:15456

# =============================================================================
# http — HTTP 代理配置（非 server 内部代理，仅特定场景使用）
# =============================================================================
# 注意：http.enable_proxy 和 http.proxy 不走 Config 结构体，
# 直接通过 viper 全局读取。生产环境如不需要可不配置。
# http:
#   enable_proxy: false
#   proxy: ""
```

- [ ] **步骤 2：Commit**

```bash
git add cocom-gen.yaml
git commit -m "feat(config): 新增完整生产配置 cocom-gen.yaml

基于 cocom-new.yaml 重写，按'cocom 专属放 cocom.*，可复用放 archive.*'
原则重新组织配置键。移除所有孤立键（archive.path/temp_path/
try_replicate/arctl），添加中文注释说明每个键的用途、默认值和环境
变量名。storage.backends 迁移到 cocom.storage.backends。"
```

---

### 任务 8：更新升级指南

**文件：**
- 修改：`cocom_v0.0.57_to_a8d21.txt`

- [ ] **步骤 1：在升级指南末尾追加破坏性变更说明**

在文件末尾追加：

```text

七、v0.0.58+ 配置键迁移（破坏性变更）

┌──────────────────────────────┬────────────────────────────────────────┬─────────┐
│          旧键（已废弃）      │              新键                      │ 必须改? │
├──────────────────────────────┼────────────────────────────────────────┼─────────┤
│ archive.password             │ cocom.archive.password                │   是    │
├──────────────────────────────┼────────────────────────────────────────┼─────────┤
│ archive.cmd                  │ cocom.archive.cmd                     │   是    │
├──────────────────────────────┼────────────────────────────────────────┼─────────┤
│ archive.replicate            │ cocom.archive.replicate               │   是    │
├──────────────────────────────┼────────────────────────────────────────┼─────────┤
│ archive.algorithm.single.*   │ cocom.archive.algorithm.single.*      │   是    │
├──────────────────────────────┼────────────────────────────────────────┼─────────┤
│ archive.algorithm.double.*   │ cocom.archive.algorithm.double.*      │   是    │
├──────────────────────────────┼────────────────────────────────────────┼─────────┤
│ storage.backends             │ cocom.storage.backends                │   是    │
├──────────────────────────────┼────────────────────────────────────────┼─────────┤
│ admin.token                  │ server.admin.token                    │   是    │
├──────────────────────────────┼────────────────────────────────────────┼─────────┤
│ admin.allow_remote           │ server.admin.allow_remote             │   是    │
└──────────────────────────────┴────────────────────────────────────────┴─────────┘

以下旧键因无对应结构体字段，已被静默忽略，请直接删除：
- archive.path, archive.temp_path, archive.try_replicate
- arctl.*（整个块）

新配置文件参考：cocom-gen.yaml（完整）或 cocom-gen-dev.yaml（最小）。
```

- [ ] **步骤 2：Commit**

```bash
git add cocom_v0.0.57_to_a8d21.txt
git commit -m "docs: 更新升级指南，补充 v0.0.58+ 配置键迁移说明

新增第七节，列出 archive.*→cocom.archive.* 等破坏性配置键变更，
标注需删除的孤立键，引用新的 cocom-gen.yaml 参考文件。"
```

---

### 任务 9：全量构建 + 测试验证

- [ ] **步骤 1：全量编译**

```bash
cd D:\workdir\leon\cocomhub\cocom
go build ./cmd/... ./pkg/... ./internal/... ./tools/...
```

预期：全部编译成功，无错误。

- [ ] **步骤 2：golangci-lint**

```bash
golangci-lint run ./...
```

预期：无新增 lint 问题。

- [ ] **步骤 3：运行配置包测试**

```bash
go test -race -count=1 ./internal/config/...
```

预期：所有测试通过（包括新增的 6 个键）。

- [ ] **步骤 4：运行 mongowrap 包测试**

```bash
go test -race -count=1 ./pkg/mongowrap/...
```

预期：所有测试通过。

- [ ] **步骤 5：运行完整单元测试（排除需要 MongoDB 的测试）**

```bash
go test -race -tags=memory_storage_integration -count=1 ./cmd/... ./pkg/... ./internal/...
```

预期：通过率不低于修改前。

- [ ] **步骤 6：go vet**

```bash
go vet ./...
```

预期：无警告。

---

## 验证清单

完成所有任务后验证：

1. [ ] `go build ./cmd/... ./pkg/... ./internal/... ./tools/...` 全部通过
2. [ ] `golangci-lint run ./...` 无新增问题
3. [ ] `go test -race -tags=memory_storage_integration -count=1 ./cmd/... ./pkg/... ./internal/...` 通过
4. [ ] `go vet ./...` 无警告
5. [ ] `cocom-gen-dev.yaml` 和 `cocom-gen.yaml` 语法正确的 YAML（用 `yq` 或 Python `yaml.safe_load` 验证）
6. [ ] `internal/config/config_keys_test.go` 中 6 个新增键的测试通过

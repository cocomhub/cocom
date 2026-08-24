# 配置管理文档

## 配置文件位置

默认配置文件位于 `conf/cocom.yaml`。也可通过 `--config` 标志指定路径。

## 配置一览

| 配置路径 | 定义位置 | 文档章节 |
|----------|----------|----------|
| `log.*` | `pkg/logging/config.go` | 日志配置 |
| `cocom.storage.path` | `internal/config/config.go` | 存储配置 |
| `cocom.archive.*` | `internal/config/config.go` | 存档配置 |
| `archive.*` (旧版兼容，迁移期) | `internal/config/config.go` | 存档配置 |
| `mongo.*` | `pkg/mongowrap/mongo.go` | MongoDB 配置 |
| `download.*` | `pkg/download/downloader.go` | 下载配置 |
| `server.*` | `cmd/server/config.go` | 服务端配置 |
| `comic.verify.*` | `pkg/comic/config.go` | 漫画验证配置 |
| `comic.download.*` | `cmd/server/internal/comic/download.go` | 漫画下载配置 |
| `cocom.cache.*` | `cmd/server/internal/cache/cache.go` | 漫画缓存配置 |
| `comic.mongo.*` | `cmd/server/internal/mongo/mongo.go` | 漫画 MongoDB 配置 |
| `cocom.storage.backends` | `tools/arctl/main.go` / `tools/pixm/main.go` | 存储注册 |
| `archive.manager.*` | `pkg/archive/manager/config.go` | 归档管理器配置 |

## 配置项说明

### 服务监听

- `server.listen.http.addr`: HTTP 监听地址（`host:port`，默认 `127.0.0.1:8080`，仅本机监听；如需对外暴露请显式配置 `0.0.0.0:8080` 或具体 IP）。旧版顶层 `host`/`port` 已删除，可用 `cocom config migrate` 迁移。

### 日志配置 (log)

Viper 键以 `log.` 为前缀：

- `log.enableFile`: 是否启用文件日志
- `log.filename`: 日志文件路径
- `log.fileLevel`: 文件日志级别 (debug/info/warn/error)
- `log.fileEncoding`: 文件日志编码格式 (json/console)
- `log.maxSize`: 单个日志文件最大尺寸（MB）
- `log.maxAge`: 日志文件保留天数
- `log.maxBackups`: 最大保留日志文件数
- `log.localtime`: 是否使用本地时间
- `log.compress`: 是否压缩旧的日志文件
- `log.enableConsole`: 是否启用控制台日志
- `log.consoleLevel`: 控制台日志级别 (debug/info/warn/error)
- `log.consoleEncoding`: 控制台日志编码格式 (json/console)
- `log.enableCaller`: 是否记录调用位置
- `log.enableSourceIP`: 是否记录源 IP
- `log.enablePID`: 是否记录进程 PID
- `log.appName`: 应用名称
- `log.sourceEth`: 源 IP 所在的网卡名称
- `log.disableTraceID`: 是否禁用 TraceID

### 存储配置 (cocom.storage)

- `cocom.storage.path`: 画廊数据存储路径
- `cocom.archive.path`: 归档文件存储路径
- `cocom.archive.temp_path`: 归档临时文件路径

#### 存储注册（storage registry）

应用启动后可调用存储注册入口（见 pkg/storage/registry），根据配置注册全局存储，供各模块通过名称获取：

- 已知存储（当路径非空时自动注册）：
  - `gallery` ← `cocom.storage.path`（LocalFS）
  - `archive` ← `cocom.archive.path`（LocalFS）
  - `archive-temp` ← `cocom.archive.temp_path`（LocalFS）
- 可选扩展项：
  - `cocom.storage.backends`: 列表，支持通过统一结构注册额外后端（旧版顶层 `storage.backends` 已迁移到此处，可用 `cocom config migrate` 迁移）：
    - `type: localfs`
      - `metadata.root`: 本地根目录
    - `type: baidupcs`
      - `metadata.root`: 远端根目录
      - `metadata.temp_dir`: 下载/上传时使用的本地临时目录
      - `metadata.bduss` 或 `metadata.cookies`: 至少提供一项认证信息
      - `metadata.uid`: 可选，自定义 uid
      - `metadata.stoken`: 可选，补充认证信息
      - `metadata.sboxtkn`: 可选，补充认证信息
      - `metadata.app_id`: 可选，自定义 app id
      - `metadata.pcs_addr`: 可选，自定义 PCS 地址
      - `metadata.pcs_user_agent`: 可选，自定义 PCS User-Agent
      - `metadata.pan_user_agent`: 可选，自定义 Pan User-Agent
    ```yaml
    cocom:
      storage:
        backends:
          - name: extra1
            type: localfs
            metadata:
              root: /mnt/data/extra1
          - name: archive-baidu
            type: baidupcs
            metadata:
              root: /apps/cocom/archive
              temp_dir: /var/tmp/cocom-baidupcs
              bduss: ${BAIDU_BDUSS}
              stoken: ${BAIDU_STOKEN}
              sboxtkn: ${BAIDU_SBOXTKN}
              app_id: 266719
    ```
  - `baidupcs` 现在直接使用内置库，不再依赖宿主机安装 `BaiduPCS-Go` 可执行文件
  - 未提供 `bduss`/`cookies` 时，驱动初始化会失败
  - `metadata.root` 会作为逻辑 key 的远端根目录前缀，`../` 等越界 key 会在驱动层被拒绝

#### BaiduPCS BREAKING 迁移

- 旧配置中的 `metadata.command`、`metadata.commandPath`、`metadata.workDir`、`metadata.timeout`、`metadata.args`、`metadata.globalArgs` 已不再是主路径配置，迁移后应删除。
- 新配置需要改为显式提供认证参数，例如 `bduss` 或 `cookies`，以及可选的 `stoken`、`sboxtkn`、`app_id`。

### 存档配置 (cocom.archive)

规范键（新部署一律使用）：
- `cocom.archive.password`: 存档加密密码。**默认空** —— 为空时 `pack` / server 归档会明确报错；若命中公开默认口令 `archive@123456` 会输出告警。**注意**：已用旧口令归档的文件，迁移配置后不会自动可解，需显式确认口令与历史一致。
- `cocom.archive.cmd`: 7z 命令路径（默认 `"7z"`）
- `cocom.archive.replicate`: 是否默认复制到远端存储
- `cocom.archive.redact_cmd`: 归档错误/日志中是否对 7z 命令行做密码脱敏（默认 `true`，可置 `false` 便于调试）
- `cocom.archive.path`: 归档文件存储根目录
- `cocom.archive.temp_path`: 归档临时文件目录
- `cocom.archive.algorithm.single.concurrency`: 单线程算法并发数

**旧键兼容（`archive.*`）**：读取时优先 `cocom.archive.*`，命中旧键 `archive.*` 且新键为默认值时回退（并输出弃用告警，v0.0.59 移除）。**注意**：新键的「零值」无法覆盖旧键的非零值（例如新键 `cocom.archive.replicate: false` 不能覆盖旧键 `archive.replicate: true`）——如需显式覆盖，请移除旧键或运行 `cocom config migrate`。
- `cocom.archive.algorithm.double.concurrency`: 双线程算法并发数

**旧版兼容（迁移期，命中时输出弃用告警，计划 v0.0.59 移除）**：
- `archive.password` — 请迁移到 `cocom.archive.password`
- `archive.cmd` — 请迁移到 `cocom.archive.cmd`
- `archive.replicate` — 请迁移到 `cocom.archive.replicate`
- `archive.algorithm.*` — 请迁移到 `cocom.archive.algorithm.*`

运行 `cocom config migrate` 可一次性迁移以上旧键。

另注意：
- `archive.root_dir` 仍存活：是 CLI 工具（arctl/pixm/ar）源数据与归档的根目录，与 `cocom.archive.path`（server 归档存储根）语义不同。
- `archive.manager.*` 是归档管理器配置，见下节。

### 归档管理器配置 (archive.manager)

- `archive.manager.algorithm`: 存档算法类型（`"double"` / `"single"`）
- `archive.manager.meta_record_file_list`: 是否记录文件列表
- `archive.manager.replicates`: 副本存储后端名称列表
- `archive.manager.index.type`: 索引类型（`"memory"` / `"file"` / `"mongo"`）
- `archive.manager.index.file_store_name`: 文件存储后端名称
- `archive.manager.index.file_store_prefix`: 文件存储 key 前缀
- `archive.manager.index.mongo_database`: MongoDB 索引数据库名
- `archive.manager.index.mongo_collection`: MongoDB 索引集合名
- `archive.manager.index.mongo_prefix`: MongoDB key 前缀
- `archive.manager.index.mongo_id_field`: MongoDB ID 字段名
- `archive.manager.index.mongo_name_field`: MongoDB 名称字段名

### 客户端配置 (client)

- `client.server_addr`: 服务器地址

### MongoDB 配置 (mongo)

- `mongo.host`: MongoDB 服务器地址
- `mongo.user`: 用户名
- `mongo.password`: 密码
- `mongo.database`: 数据库名
- `mongo.authSource`: 认证数据库

### 下载配置 (download)

- `download.maxRunning`: 最大并发下载数
- `download.downloadDir`: 下载目录
- `download.enableProxy`: 是否启用 HTTP 代理下载（旧版 `http.enable_proxy` 已迁移到此处）
- `download.proxyURL`: HTTP 代理地址（旧版 `http.proxy` 已迁移到此处，仅 `enableProxy=true` 时生效）

### 服务端配置 (server)

#### 访问日志 (server.access_log)

- `server.access_log.patterns`: 记录访问日志的 URL 模式列表（默认 `["/debug", "/api", "/v1", "/v2"]`）

#### CORS (server.cors)

- `server.cors.expose_headers`: CORS 响应 `Access-Control-Expose-Headers` 值（可选；默认不设置）
- `server.cors.allow_origins`: 允许的源
- `server.cors.allow_methods`: 允许的 HTTP 方法
- `server.cors.allow_headers`: 允许的请求头

#### Gzip (server.gzip)

- `server.gzip.enabled`: 是否启用 Gzip 压缩
- `server.gzip.level`: 压缩级别

#### 限流 (server.ratelimit)

- `server.ratelimit.enabled`: 是否启用限流
- `server.ratelimit.rps`: 每秒请求数限制
- `server.ratelimit.burst`: 突发请求数

#### 调度器 (server.scheduler)

- `server.scheduler.enabled`: 是否启用调度器
- `server.scheduler.timezone`: 时区

**漫画探测调度 (probe_comic)：**

- `server.scheduler.probe_comic.enabled`: 是否启用
- `server.scheduler.probe_comic.name`: 任务名称
- `server.scheduler.probe_comic.cron`: Cron 表达式
- `server.scheduler.probe_comic.tags`: 标签列表

**存档状态检查调度 (archive_status_check)：**

- `server.scheduler.archive_status_check.enabled`: 是否启用
- `server.scheduler.archive_status_check.name`: 任务名称
- `server.scheduler.archive_status_check.cron`: Cron 表达式（默认每 30 分钟）
- `server.scheduler.archive_status_check.tags`: 标签列表
- `server.scheduler.archive_status_check.limit`: 每次检查数量上限
- `server.scheduler.archive_status_check.max_conn`: 最大并发连接数
- `server.scheduler.archive_status_check.backends`: 要检查的后端列表

**Cocoma 归档调度 (cocoma_archiver)：**

- `server.scheduler.cocoma_archiver.enabled`: 是否启用
- `server.scheduler.cocoma_archiver.cron`: Cron 表达式
- `server.scheduler.cocoma_archiver.limit`: 每次处理上限
- `server.scheduler.cocoma_archiver.cid_regex`: CID 匹配正则
- `server.scheduler.cocoma_archiver.scan_dir`: 扫描目录
- `server.scheduler.cocoma_archiver.archive_dir`: 归档输出目录
- `server.scheduler.cocoma_archiver.notmatch_dir`: 不匹配文件的移动目录

### 漫画相关配置 (comic)

#### MongoDB 集合配置 (comic.mongo)

- `comic.mongo.database`: 漫画 MongoDB 数据库名
- `comic.mongo.collections.comicInfo`: comicInfo 集合名
- `comic.mongo.collections.oneComicInfo`: oneComicInfo 集合名
- `comic.mongo.collections.videoInfo`: videoInfo 集合名
- `comic.mongo.collections.settings`: settings 集合名
- `comic.mongo.collections.custom`: custom 集合名
- `comic.mongo.collections.comicTag`: comicTag 集合名

#### 下载配置 (comic.download)

- `comic.download.maxDownloadSize`: 最大下载大小（单位：图片数，默认 5）

#### 验证配置 (comic.verify)

- `comic.verify.concurrent`: 验证并发协程数
- `comic.verify.task_buffer_size`: 任务缓冲区大小
- `comic.verify.autoFix`: 是否自动修复损坏的图片
- `comic.verify.retryInterval`: 重试间隔时间
- `comic.verify.maxRetries`: 最大重试次数
- `comic.verify.checkInterval`: 定期检查间隔
- `comic.verify.timeoutDuration`: 超时时间

### 缓存配置 (cocom.cache)

- `cocom.cache.cleanInterval`: 缓存清理间隔
- `cocom.cache.evictionInterval`: 缓存淘汰间隔

## 配置示例

```yaml
# 服务监听
server:
  listen:
    http:
      addr: "127.0.0.1:8080"

# 日志配置
log:
  enableFile: true
  filename: "/var/log/cocom/cocom.log"
  fileLevel: "info"
  enableConsole: true
  consoleLevel: "debug"
  appName: "cocom"

# 存储配置
cocom:
  storage:
    path: "/data/cocom"
    backends:
      - name: "backup"
        type: "localfs"
        metadata:
          root: "/data/backup"
      - name: "archive-baidu"
        type: "baidupcs"
        metadata:
          root: "/apps/cocom/archive"
          temp_dir: "/var/tmp/cocom-baidupcs"
          bduss: "${BAIDU_BDUSS}"
          stoken: "${BAIDU_STOKEN}"
          sboxtkn: "${BAIDU_SBOXTKN}"
          app_id: 266719
  archive:
    path: "/data/cocom/archive"
    temp_path: "/data/cocom/archive-temp"
    password: ""          # 默认空，pack 需显式配置
    cmd: "7z"
    replicate: false

# 客户端配置
client:
  server_addr: "http://127.0.0.1:8080"

# MongoDB配置
mongo:
  host: "localhost:27017"
  database: "cocom"
  user: "cocom"
  password: "cocom123"
  authSource: "cocom"

# 下载配置
download:
  maxRunning: 4
  downloadDir: "/data/cocom/downloads"
  enableProxy: false
  proxyURL: ""

# 漫画配置
comic:
  download:
    maxDownloadSize: 100  # 100 张图片
  verify:
    concurrent: 10
    autoFix: true
    retryInterval: "1h"
    maxRetries: 3
    checkInterval: "24h"
    timeoutDuration: "30s"
```

## 最佳实践

### 1. 配置文件管理

- 使用版本控制管理配置模板
- 不要将包含敏感信息的配置文件提交到代码库
- 为不同环境（开发、测试、生产）创建不同的配置文件

### 2. 安全性

- 敏感信息（如密码、密钥）使用环境变量或密钥管理系统
- MongoDB 建议启用认证
- 生产环境建议禁用调试日志

### 3. 性能优化

- 根据服务器资源调整并发数
- 合理设置缓存大小和过期时间
- 监控配置建议启用以便及时发现问题

### 4. 日志配置

- 生产环境建议使用文件日志
- 设置合适的日志轮转策略
- 根据磁盘空间调整日志级别

### 5. 监控告警

- 配置合适的告警阈值
- 设置正确的告警接收人
- 定期检查告警配置

## 配置热更新

配置文件支持热更新，修改配置文件后会自动重新加载。某些配置项的修改可能需要重启服务才能生效。

## 配置验证

所有配置项都会进行验证，确保：

1. 必填项不为空
2. 数值在有效范围内
3. 路径和URL格式正确
4. 时间格式正确

## 环境变量覆盖

可以使用环境变量覆盖配置文件中的设置，环境变量格式为：`COCOM_[配置路径]`
例如：

- `COCOM_SERVER_LISTEN_HTTP_ADDR=0.0.0.0:8080`（监听地址，顶层 `COCOM_PORT` 已移除）
- `COCOM_MONGO_HOST=localhost:27017`

## 故障排除

### 1. 配置加载失败

- 检查配置文件路径是否正确
- 验证配置文件格式是否符合 YAML 规范
- 查看日志中的具体错误信息

### 2. 配置验证失败

- 检查必填字段是否已填写
- 确认数值是否在有效范围内
- 验证路径和 URL 格式是否正确

### 3. 热更新不生效

- 检查文件权限是否正确
- 确认修改的配置项是否支持热更新
- 查看日志中是否有相关错误信息
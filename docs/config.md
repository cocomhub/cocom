# 配置管理文档

## 配置文件位置

默认配置文件位于 `conf/cocom.yaml`。也可通过 `--config` 标志指定路径。

## 配置一览

| 配置路径 | 定义位置 | 文档章节 |
|----------|----------|----------|
| `port` | — | 基础配置 |
| `log.*` | `pkg/logging/config.go` | 日志配置 |
| `cocom.storage.path` | `internal/config/config.go` | 存储配置 |
| `cocom.archive.*` (已废弃) | `internal/config/config.go` | 存档配置 |
| `archive.*` | `internal/config/config.go` | 存档配置 |
| `mongo.*` | `pkg/mongowrap/mongo.go` | MongoDB 配置 |
| `download.*` | `pkg/download/downloader.go` | 下载配置 |
| `server.*` | `cmd/server/config.go` | 服务端配置 |
| `comic.verify.*` | `pkg/comic/config.go` | 漫画验证配置 |
| `comic.download.*` | `cmd/server/internal/comic/download.go` | 漫画下载配置 |
| `comic.cache.*` | `cmd/server/internal/cache/cache.go` | 漫画缓存配置 |
| `comic.mongo.*` | `cmd/server/internal/mongo/mongo.go` | 漫画 MongoDB 配置 |
| `storage.backends` | `tools/arctl/main.go` / `tools/pixm/main.go` | 存储注册 |
| `archive.manager.*` | `pkg/archive/manager/config.go` | 归档管理器配置 |

## 配置项说明

### 基础配置

- `port`: 服务器监听端口 (1024-65535)

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
  - `storage.backends`: 列表，支持通过统一结构注册额外后端：
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

### 存档配置 (archive)

- `archive.password`: 存档加密密码
- `archive.cmd`: 7z 命令路径（默认 `"7z"`）
- `archive.replicate`: 是否默认复制到远端存储
- `archive.root_dir`: 归档根目录（可选，默认使用 `rootcli.DataDir()`）
- `archive.algorithm.single.concurrency`: 单线程算法并发数
- `archive.algorithm.double.concurrency`: 双线程算法并发数

**已废弃（legacy 路径）**：
- `cocom.archive.password` — 请改用 `archive.password`
- `cocom.archive.cmd` — 请改用 `archive.cmd`
- `cocom.archive.replicate` — 请改用 `archive.replicate`

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

### 服务端配置 (server)

#### 访问日志 (server.access_log)

- `server.access_log.patterns`: 记录访问日志的 URL 模式列表（默认 `["/debug", "/api", "/v1", "/v2"]`）

#### CORS (server.cors)

- `server.cors.enabled`: 是否启用 CORS
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
# 基础配置
port: 35456

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
  archive:
    path: "/data/cocom/archive"
    temp_path: "/data/cocom/archive-temp"

# 存档配置
archive:
  password: "archive@123456"
  cmd: "7z"
  replicate: false

storage:
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

# 客户端配置
client:
  server_addr: "http://localhost:35456"

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

- `COCOM_PORT=35456`
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
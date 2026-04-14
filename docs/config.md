# 配置管理文档

## 配置文件位置

默认配置文件位于 `conf/cocom.yaml`。

## 配置项说明

### 基础配置

- `port`: 服务器监听端口 (1024-65535)

### 日志配置 (logging)

- `enableFile`: 是否启用文件日志
- `filename`: 日志文件路径
- `fileLevel`: 文件日志级别 (debug/info/warn/error)
- `enableConsole`: 是否启用控制台日志
- `consoleLevel`: 控制台日志级别 (debug/info/warn/error)
- `appName`: 应用名称

### 存储配置 (cocom.storage)

- `path`: 数据存储路径

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

### 客户端配置 (client)

- `server_addr`: 服务器地址

### MongoDB配置 (mongo)

- `host`: MongoDB服务器地址
- `username`: 用户名（可选）
- `password`: 密码（可选）
- `database`: 数据库名

### 下载配置 (download)

- `maxRunning`: 最大并发下载数
- `downloadDir`: 下载目录

### 漫画相关配置 (comic)

#### 下载配置 (comic.download)

- `maxDownloadSize`: 最大下载大小

#### 验证配置 (comic.verify)

- `autoFix`: 是否自动修复损坏的图片
- `concurrent`: 验证并发数
- `retryInterval`: 重试间隔时间
- `maxRetries`: 最大重试次数
- `checkInterval`: 定期检查间隔
- `timeoutDuration`: 超时时间

#### 缓存配置 (comic.cache)

- `enabled`: 是否启用缓存
- `maxSize`: 最大缓存大小（字节）
- `expireTime`: 缓存过期时间

#### 监控配置 (comic.monitor)

- `enabled`: 是否启用监控
- `metricsPort`: 监控指标端口
- `alertTargets`: 告警接收邮箱列表

## 配置示例

```yaml
# 基础配置
port: 35456

# 日志配置
logging:
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

# 下载配置
download:
  maxRunning: 4
  downloadDir: "/data/cocom/downloads"

# 漫画配置
comic:
  download:
    maxDownloadSize: 104857600  # 100MB
  verify:
    autoFix: true
    concurrent: 4
    retryInterval: "1h"
    maxRetries: 3
    checkInterval: "24h"
    timeoutDuration: "30s"
  cache:
    enabled: true
    maxSize: 1073741824  # 1GB
    expireTime: "72h"
  monitor:
    enabled: true
    metricsPort: 35457
    alertTargets:
      - "admin@example.com"
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


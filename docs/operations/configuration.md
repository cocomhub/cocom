# 配置管理文档

- 配置文件位置：`conf/cocom.yaml` 示例 `conf/cocom.yaml:1-28`
- 运行时默认值：下载器 `pkg/download/downloader.go:41-43`，Mongo 集合名 `cmd/server/internal/mongo/mongo.go:48-54`
- 环境变量覆盖：viper 自动加载 `cmd/root.go:93-100`

## 主要配置项

- 基础：`port` 监听端口
- logging：文件/控制台级别与文件名
- 存储：`cocom.storage.path`
- 归档存储（新增）：`archive.storage.backend`（默认 `localfs`）、`archive.storage.localfs.root`
- 客户端：`client.server_addr`
- Mongo：`mongo.host` 等
- 下载：`download.maxRunning`、`download.downloadDir`
- 漫画：`comic.download.maxDownloadSize`

## 最佳实践

- 区分环境配置，敏感信息不入库；启用 Mongo 认证；生产环境减少调试日志
- 归档存储后端的凭据与路径不应在日志中输出，避免泄露；默认使用 LocalFS 保持行为一致

## 环境变量覆盖示例

- `COCOM_PORT=35456`
- `COCOM_MONGO_HOST=localhost:27017`

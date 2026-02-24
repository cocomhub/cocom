# 故障排查

## 下载器相关
- 未安装 wget 导致启动修复时 panic：`pkg/comic/comic.go:441-464`
- 网络/服务器错误的重试策略：`pkg/comic/comic.go:291-304`

## 数据库连接
- 连接失败与集合名配置：`cmd/verify.go:235-250`、`cmd/server/internal/mongo/mongo.go:47-96`

## 图片验证
- 文件不存在/权限不足/解码失败：`pkg/comic/verify.go:561-570`

## 代理与下载失败
- 代理配置与写入失败日志：`pkg/download/downloader.go:160-169,268-276`

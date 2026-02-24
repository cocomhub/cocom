# Storage 模块

## 抽象接口与过滤器
- 接口：`pkg/comic/storage.go:12-24`
- 过滤器字段与构造器：`pkg/comic/storage.go:31-56,96-110`

## Mongo 集成
- 集合名默认值：`cmd/server/internal/mongo/mongo.go:47-54`
- 数据库与集合：`cmd/server/internal/mongo/mongo.go:56-96`
- 独立 Mongo 存储实现：`pkg/comic/storage/mongo.go:14-127`

## 服务器内部存储
- Nhcomic 与 Onecomic：`cmd/server/internal/{comic,onecomic}/storage.go`
- 过滤器映射到 Mongo 查询：`cmd/server/internal/comic/storage.go:95-135`、`cmd/server/internal/onecomic/storage.go:87-111`

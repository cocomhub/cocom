# Legacy `/api/*` 接口

- 路由注册：`cmd/server/handler/comic.go:34-38`
- 响应包裹格式：`pkg/httpwrap/http.go:27-37,39-59`
- 主要接口：
  - 保存漫画信息：`POST /api/comic/saveComicInfo` `cmd/server/handler/comic.go:40-96`
  - 获取漫画信息：`GET /api/comic/getComicInfo?cid=<cid>` `cmd/server/handler/comic.go:98-136`
  - 触发下载：`POST /api/comic/download`（支持异步与并发控制）`cmd/server/handler/comic.go:138-184`


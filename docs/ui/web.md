# Web 页面与路由

- 静态资源与模板注册：`cmd/server/view/init.go:43-61`
- 页面路由：
  - 首页：`GET /` `cmd/server/view/init.go:50-51`
  - 画廊详情：`GET /g/:cid` `cmd/server/view/init.go:54-55`
  - 单页浏览：`GET /g/:cid/:no` `cmd/server/view/init.go:56-57`
  - 标签检索：`GET /tag/:tag/:name`、`GET /list/:tagType` `cmd/server/view/init.go:52-53,60-61`
  - 图片访问：`GET/HEAD /galleries/:cid/:name` `cmd/server/view/init.go:48-49`
- 模板函数：`Add`、`TitlePretty` 等 `cmd/server/view/init.go:64-76,92-104,111-123`

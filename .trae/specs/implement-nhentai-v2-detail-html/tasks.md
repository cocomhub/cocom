# Tasks
- [x] 任务1：实现 HTML 预取脚本解析器
  - [x] 在 probe 包新增 `parseComicPageV2FromHTML(html string, id int) (map[string]any, error)`
  - [x] 定位 `script[type="application/json"][data-sveltekit-fetched][data-url^="/api/v2/galleries/"]`
  - [x] 解码脚本 JSON，再对 `body` 二次 JSON 解码，映射字段
  - [x] 补充 `comic_id` 与 `comic_url`
- [x] 任务2：整合到 v2 详情解析策略
  - [x] 在现有 v2 分支中先抓取详情页 HTML 并尝试 HTML 解析
  - [x] 若失败则回退到 `/api/gallery/{id}` 的实现
  - [x] 记录解析来源（html/api）日志
- [x] 任务3：添加基于样例的单测
  - [x] 新增 `v2_detail_html_test.go`，读取 `/pkg/comic/probe/640503.html`
  - [x] 断言能解析到 `media_id`、`images.pages`（或等效）等关键字段
  - [x] 验证能序列化为 `api.ComicInfo` 并与 `genDownList` 兼容

# Task Dependencies
- [任务2] 依赖 [任务1]
- [任务3] 可与 [任务1] 并行编写，依赖样例文件

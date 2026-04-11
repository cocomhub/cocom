# Tasks
- [x] 任务1：引入模式参数与策略选择骨架（默认 v1）
  - [x] 添加 `--nhentai_mode` flag（v1|v2，默认 v1）
  - [x] 在探测流程中读取参数并分派到 v1/v2 实现
- [x] 任务2：实现 v2 列表页解析（基于 SvelteKit 数据脚本）
  - [x] 从 HTML 中定位 `script[type="application/json"][data-sveltekit-fetched]`
  - [x] 选择 `data-url` 以 `/api/v2/galleries?page=` 开头的脚本
  - [x] 解析脚本 JSON，取 `body` 字段再 JSON 解码，提取 `result[].id`、`tag_ids`
  - [x] 依据标签（6346、29963）与 `lastComic` 规则过滤与截断，返回 ID 列表
- [x] 任务3：实现 v2 详情页解析（基于官方 JSON）
  - [x] 请求 JSON 接口获取详情（如 `/api/gallery/{id}` 或等价 v2 接口）
  - [x] 映射到现有结构：`comic_id`、`comic_url`、`media_id`、`images.pages` 等
  - [x] 确保 `saveComicInfo` 与 `genDownList` 可直接复用
- [x] 任务4：接入 v2 到探测流程
  - [x] 在 `probeComic` 分支调用 v2 列表与详情
  - [x] 记录关键日志，保证异常回退与错误输出
- [x] 任务5：增加基础测试与本地验证
  - [x] 使用 `pkg/comic/probe/index.html` 快照编写 v2 列表解析单测
  - [x] 构造或引入示例详情 JSON，验证字段映射
  - [x] 本地跑通探测前几页的 dry-run（仅日志）验证

# Task Dependencies
- [任务2] 依赖 [任务1]
- [任务3] 依赖 [任务1]
- [任务4] 依赖 [任务2] 与 [任务3]
- [任务5] 可与 [任务2]/[任务3] 并行，完成后再执行 [任务4] 的联调

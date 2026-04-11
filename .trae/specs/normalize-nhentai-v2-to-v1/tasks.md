# Tasks
- [x] 任务1：实现归一化函数
  - [x] 在 probe 包新增 `normalizeV2ToV1(info map[string]any) map[string]any`
  - [x] 将 pages 的 path 扩展名映射为 t（j/p/w/g），并复制 width/height 到 w/h
  - [x] 将 cover/thumbnail 的 path 扩展名映射为 t，复制 width/height 到 w/h
  - [x] 移除 images 中无用的 path 字段
- [x] 任务2：接入 v2 解析路径
  - [x] 在 `parseComicPageV2FromHTML` 返回前调用归一化
  - [x] 在 `parseComicPageV2`（API 兜底）返回前调用归一化
  - [x] 记录日志表明已归一
- [x] 任务3：添加单元测试
  - [x] 新增 `normalize_v2_to_v1_test.go`，读取 `comicInfo.v2.json`，归一化后断言 images.pages/cover/thumbnail 为 t/w/h 形态
  - [x] 断言可序列化为 `api.ComicInfo` 并使用 `PageOriginUrlByIndex` 生成 URL

# Task Dependencies
- [任务2] 依赖 [任务1]
- [任务3] 可与 [任务1] 并行编写，依赖样例文件

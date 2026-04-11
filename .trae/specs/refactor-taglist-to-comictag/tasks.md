# Tasks
- [x] 任务1：comicTag 提供分段与列表聚合
  - [x] 子任务1.1：实现 AggregateTagSectionIndices（支持 liked 过滤）
  - [x] 子任务1.2：实现 AggregateTagList（支持类型/分页/排序与 liked 过滤）
- [x] 任务2：缓存调整为单 ID
  - [x] 子任务2.1：实现 GetTagByID 缓存键 comicTag:id:{type}:{id}
  - [x] 子任务2.2：替换调用点，移除批量 ids 键的缓存
- [x] 任务3：页面与接口支持 liked 筛选
  - [x] 子任务3.1：TagListPage 接受 liked 参数并应用到分段与列表
  - [x] 子任务3.2：GET /api/comic/tags 支持 liked=true 过滤
- [x] 任务4：验证与构建
  - [x] 子任务4.1：go build 验证
  - [x] 子任务4.2：页面与接口手动验证

# Task Dependencies
- [任务2] 依赖 [任务1]
- [任务3] 依赖 [任务1]

# Tasks
- [x] 任务1：实现标签聚合与存储到 comicTag
  - [x] 子任务1.1：从 comicInfo 聚合 tag（type/id/name/url/count）
  - [x] 子任务1.2：写入/更新 comicTag 集合，补充 like=false 与 updated_at
  - [x] 子任务1.3：提供读取接口（按类型/分页/排序），增加缓存
  - [x] 子任务1.4: TagListResultPage 直接使用聚合数据，避免实时查询 comicInfo 统计
- [x] 任务2：TagResultPage 喜欢/取消喜欢交互（侧边栏形式）
  - [x] 子任务2.1：新增后端接口设置/取消 like（更新 comicTag）
  - [x] 子任务2.2：页面按钮与样式切换（btn-secondary/btn-primary）
- [x] 任务3：新增样式 tag-like 并应用
  - [x] 子任务3.1：CSS 增加 tag-like 红色样式
  - [x] 子任务3.2：模板中根据 like 状态应用 tag-like
- [x] 任务4：展示层读取 comicTag 聚合数据
  - [x] 子任务4.1：替换从 comicInfo 读取的 count 为 comicTag
  - [x] 子任务4.2：缓存命中与回填；无数据时降级
- [x] 任务5：测试与验证
  - [x] 子任务5.1：聚合与读取接口的单元/集成测试
  - [x] 子任务5.2：TagResultPage 喜欢交互与样式验证

# Task Dependencies
- [任务2] 依赖 [任务1]
- [任务4] 依赖 [任务1]

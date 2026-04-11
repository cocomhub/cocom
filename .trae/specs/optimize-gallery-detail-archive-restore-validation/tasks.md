# Tasks
- [x] 任务1：后端归档前校验与强制归档
  - [x] 子任务1.1：ArchiveComic 增加校验逻辑，返回 ResponseInfo（code=-1001，body.invalid_images）
  - [x] 子任务1.2：支持 force=true 强制归档，并落库 ArchiveInfo.by_force=true
  - [ ] 子任务1.3：完善单元测试：校验通过/失败、强制归档路径
- [ ] 任务2：后端恢复前 MD5 校验
  - [x] 子任务2.1：RestoreComic 增加 MD5 比对，不一致返回 code=-2001，并附 expected/actual
  - [ ] 子任务2.2：完善单元测试：匹配/不匹配路径
- [x] 任务3：前端交互与可视化
  - [x] 子任务3.1：统一按 httpwrap.ResponseInfo 解析接口；失败 Toast「[$code] $msg」
  - [x] 子任务3.2：归档失败接收 invalid_images，高亮对应图片并展示引导
  - [x] 子任务3.3：在异常场景显示「强制归档」按钮并调用 ?force=true
  - [x] 子任务3.4：文案与提示统一：修复漫画状态入口复用既有接口
- [ ] 任务4：验证与文档
  - [ ] 子任务4.1：手动验证 6 个路径：归档成功/失败/强制；恢复成功/MD5 不匹配；消息格式
  - [ ] 子任务4.2：补充 README/开发文档，声明错误码与交互规范

# Task Dependencies
- [任务3] 依赖 [任务1][任务2]
- [任务4] 依赖 [任务1][任务2][任务3]

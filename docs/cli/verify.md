# verify 命令

- 用途：校验漫画图片完整性，支持自动修复与定时任务 `cmd/verify.go:30-64`
- 常用参数：
  - `--pattern` 标题匹配正则 `cmd/verify.go:83-88`
  - `--auto-fix` 自动修复
  - `--workers` 并发数
  - `status <taskID>` 查看任务进度 `cmd/verify.go:137-166`
  - `cancel <taskID>` 取消任务 `cmd/verify.go:168-197`
  - `schedule` 启动定时任务 `cmd/verify.go:199-232`
- 任务输出：处理总数/损坏数/修复数 `cmd/verify.go:127-133`

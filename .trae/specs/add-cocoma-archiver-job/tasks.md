# Tasks

 - [x] 任务 1：新增配置与默认值
  - [x] 注入 viper 默认：enabled=false、cron="* * * * *"、limit=10000、cid_regex="^(\\d+)\\.cocoma$"
  - [x] 新增必填项：scan_dir、archive_dir、notmatch_dir（无默认，缺失时任务记录错误并不运行）

 - [x] 任务 2：实现归档作业库 pkg/cocomaarchiver
  - [x] 提供 RunOnce(ctx, opts) (processed, matched, mismatched int, err) 执行一次扫描处理（受 limit 约束）
  - [x] 递归枚举 scan_dir 下 *.cocoma 常规文件（忽略符号链接/隐藏目录可按需跳过）
  - [x] 解析 cid：优先使用 cid_regex 从文件名提取，失败则跳过并记日志
  - [x] 计算文件 MD5 并对比 comicInfo(md5)（通过内部 service/dao 查询）
  - [x] 移动文件：一致→archive_dir，失败→notmatch_dir；实现跨分区安全移动（rename 或 copy+fsync+remove）
  - [x] 返回统计信息，便于 UI 或日志展示

 - [x] 任务 3：在 scheduler 中注册 CocomaArchiver
  - [x] 新增 internal/scheduler/cocoma_archiver.go，读取配置并注册到 gocron
  - [x] 名称/标签：name=CocomaArchiver；tags=["archive","cocoma"]
  - [x] 使用 CronJob 表达式（自动判断是否含秒）；每次触发调用 RunOnce，避免重入

 - [x] 任务 4：验证与可视化
  - [x] 在 /admin/cron 可见任务并可 Run 一次
  - [x] 本地构建通过并进行一次手动验证（dry run 或小样本目录）

# Task Dependencies
- [任务 3] 依赖 [任务 1][任务 2]
- [任务 4] 依赖 [任务 3]

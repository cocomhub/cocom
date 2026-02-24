## 文档目标
- 全量梳理当前代码实现的功能并形成可查阅的文档
- 将文档按“用户指南 / API 参考 / 模块参考 / 运维与配置 / 示例与故障排查”分层分类
- 与现有 docs 内容整合，统一入口与导航，避免与代码不一致

## 文档目录结构
- docs/README.md（统一索引与导航）
- docs/overview/
  - architecture.md（整体架构与流程）
- docs/operations/
  - configuration.md（配置项与来源，整合现有 config.md）
  - deployment.md（Docker、运行、端口、环境变量）
- docs/cli/
  - server.md（启动服务、端口参数）
  - verify.md（漫画校验与定时任务）
  - image.md（图片处理子命令）
  - install.md（依赖安装，如 webp）
  - version.md（版本信息与 dirty-info）
- docs/api/
  - legacy.md（/api/* 旧接口与响应包裹格式）
  - v2.md（/v2/api/{onecomic|nhcomic} 校验与检索接口）
  - debug.md（pprof 与管理接口）
- docs/ui/
  - web.md（页面路由与模板片段）
- docs/modules/
  - comic.md（领域模型、校验器、下载器、存储、指标）
  - imaging.md（图像处理、批处理、验证）
  - download.md（抓取下载器、批量任务）
  - logging.md（日志与 Trace ID）
  - storage.md（Mongo 集合、Builder）
- docs/troubleshooting.md（常见问题与解决）
- docs/examples.md（端到端使用示例）

## 具体文档与内容要点
- docs/overview/architecture.md
  - CLI 入口与整体结构：`main.go:23-25`、`cmd/root.go:57-72`
  - Web 服务初始化与路由：`cmd/server/server.go:60-101`
  - 视图层注册与静态资源：`cmd/server/view/init.go:43-61`
  - 领域服务与存储抽象：`pkg/comic/service.go:10-21`、`pkg/comic/storage.go:12-24`
  - Mongo 构建与集合：`cmd/server/internal/mongo/mongo.go:47-96`

- docs/operations/configuration.md
  - 配置文件示例与释义：`conf/cocom.yaml:1-28`
  - 运行时默认值来源：下载默认值 `pkg/download/downloader.go:41-43`，Mongo 默认集合名 `cmd/server/internal/mongo/mongo.go:48-54`
  - 环境变量覆盖（viper）：`cmd/root.go:93-100`

- docs/cli/server.md
  - 启动命令与端口参数：`cmd/server.go:60-65`
  - 运行时管理接口（关闭）：`cmd/server/server.go:72-81`

- docs/cli/verify.md
  - 命令、标志与示例：`cmd/verify.go:30-64`、标志 `cmd/verify.go:83-88`
  - 启动任务与进度查看：`cmd/verify.go:90-135`、`cmd/verify.go:137-166`
  - 取消与定时任务：`cmd/verify.go:168-197`、`cmd/verify.go:199-232`

- docs/cli/image.md
  - 子命令与参数：`cmd/image.go:26-110` 索引，具体子命令实现见各段（如 `resizeCmd` 等）
  - 批处理与结果输出：`cmd/image.go:433-494`
  - WebP 工具检测：`cmd/image.go:392-399`

- docs/cli/install.md
  - webp 安装：`cmd/install.go:20-31`、平台实现 `cmd/install.go:60-103`

- docs/cli/version.md
  - 版本与 dirty-info：`cmd/version.go:36-79`

- docs/api/legacy.md
  - 旧版接口路由：`cmd/server/handler/comic.go:34-38`
  - 请求/响应格式（统一包裹）：`pkg/httpwrap/http.go:27-37,39-59`
  - 示例接口：保存与获取漫画信息、发起下载：`cmd/server/handler/comic.go:40-96,98-136,138-184`

- docs/api/v2.md
  - 路由分组：`cmd/server/server.go:90-91`
  - 处理器注册：`pkg/comic/handler.go:26-39`
  - 校验任务接口：创建/查询/进度/取消/列表/定时：`pkg/comic/handler.go:41-161`
  - 漫画检索与无效漫画：`pkg/comic/handler.go:163-203`
  - 漫画信息与封面路径：`pkg/comic/handler.go:205-233`

- docs/api/debug.md
  - pprof：`cmd/server/handler/mux.go:41-46`
  - 关闭服务：`cmd/server/server.go:72-81`

- docs/ui/web.md
  - 页面路由：`cmd/server/view/init.go:47-61`
  - 模板函数：`cmd/server/view/init.go:64-76,84-104,111-123`

- docs/modules/comic.md
  - 领域模型与校验结果：`pkg/comic/comic.go:22-39,48-61`、`pkg/comic/storage.go:112-123`
  - 校验任务/进度/状态：`pkg/comic/verify.go:26-66,153-166,245-254,261-270`
  - 校验流程与修复：`pkg/comic/verify.go:405-521,533-577,579-601`
  - 生成下载列表：`pkg/comic/verify.go:471-486`
  - 定时任务：`pkg/comic/verify.go:661-732`
  - 下载器（HTTP/Wget）：`pkg/comic/comic.go:168-289,364-439`
  - 指标收集：`pkg/comic/metrics.go:8-19,21-75`
  - 存储实现（Memory/Mongo/Server内部）：`pkg/comic/storage.go:125-200`、`pkg/comic/storage/mongo.go:14-127`、`cmd/server/internal/{comic,onecomic}/storage.go`

- docs/modules/imaging.md
  - 验证与批处理：`pkg/imaging/verify.go:12-39`
  - 单图处理 API 与批处理框架（从 `cmd/image.go` 归纳）

- docs/modules/download.md
  - 抽象与默认配置：`pkg/download/downloader.go:34-54,97-125`
  - 运行与并发模型：`pkg/download/downloader.go:188-217`
  - 批量任务接口：`pkg/download/downloader.go:244-250`、`pkg/download/task.go:22-32`

- docs/modules/logging.md
  - clog 使用与 Trace ID 贯穿：`pkg/clog/*` 引入点 `cmd/root.go:100`、请求追踪 `cmd/server/handler/mux.go:34-37`

- docs/modules/storage.md
  - Mongo 连接与集合名配置：`cmd/server/internal/mongo/mongo.go:47-96`
  - Builder：`pkg/mongowrap/builder.go`

- docs/troubleshooting.md
  - wget 未安装导致 panic：`pkg/comic/comic.go:441-464`
  - Mongo 连接失败与集合名：`cmd/verify.go:235-250`、`cmd/server/internal/mongo/mongo.go`
  - 图片验证失败分类与处理：`pkg/comic/verify.go:561-575`
  - 代理与下载失败：`pkg/download/downloader.go:160-169,268-276`

- docs/examples.md
  - 启动服务与访问 UI
  - 触发校验（自动修复/生成下载列表）与查看进度
  - 使用 image 子命令进行批量处理与结果输出
  - 使用 genwget 生成脚本：`cmd/genwget/genwget.go:57-89,125-151`

## 实施步骤
- 建立上述目录结构，迁移并整合现有 docs（保留原文件作为重定向或章节）
- 从代码提取 API 与 CLI 的参数、示例与响应模型，补充示例请求/响应
- 将配置文档与 conf/cocom.yaml 对齐，标注默认值来源与可覆盖方式
- 为模块文档添加跨引用（API/CLI/配置），形成闭环
- 完成后运行实际验证：
  - 启动本地服务，验证路由与 UI 页面可访问
  - 运行 CLI 示例验证输出与行为
  - 以示例配置验证连接与默认值

## 校验方式
- 通过实际运行与最小示例请求校验文档可操作性
- 所有文档附带代码引用位置，便于查证与更新

## 维护策略
- 新增功能合并时同步更新对应文档章节
- 为 PR 引入“文档检查清单”：是否影响 API/CLI/配置/模块
- 定期对照代码进行文档一致性检查（含默认值与参数）
# Download 模块

## 配置与默认值
- 默认最大并发与目录：`pkg/download/downloader.go:41-43`
- 初始化替换默认下载器：`pkg/download/downloader.go:45-55`

## 运行模型
- 启动：`Start()` 创建工作协程 `pkg/download/downloader.go:188-217`
- 关闭与等待：`Close()`、`Wait()` `pkg/download/downloader.go:219-242`

## 批量任务
- 提交批量下载：`DoBatch(workers, tasks...)` `pkg/download/downloader.go:244-250`
- 任务/结果结构：`pkg/download/task.go:22-32`

## 代理支持
- 通过 viper 配置代理并注入 HTTP 客户端 `pkg/download/downloader.go:160-169`

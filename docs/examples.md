# 使用示例

## 启动与访问
- 启动服务：`cocom server -p 15456`
- 访问 UI：浏览器打开 `http://localhost:15456/`

## 校验与修复
- 启动校验并自动修复：`cocom verify --pattern ".*" --auto-fix --workers 4`
- 查看进度：`cocom verify status <taskID>`
- 取消任务：`cocom verify cancel <taskID>`

## 生成下载列表
- 启动校验但仅生成下载列表（不修复）：通过接口选项 `GenDownList` `pkg/comic/verify.go:471-486`

## 批量图片处理
- 批量调整大小：`cocom image resize "./src/*.jpg" ./output/ 800 600 --batch`
- 批量格式转换（WebP）：`cocom image convert "./src/*.*" ./output/ webp --batch`

## 生成 wget 脚本
- 使用 `cmd/genwget` 根据 CID 批量生成脚本：`cmd/genwget/genwget.go:57-89,125-151`

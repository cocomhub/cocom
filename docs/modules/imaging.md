# Imaging 模块

## 功能
- 图片验证：`pkg/imaging/verify.go:22-32` 返回 `ImageInfo` 并标识无效
- 批量验证：`pkg/imaging/verify.go:34-39` 与 CLI `image verify` 配合使用

## 与 CLI 的协作
- 单图处理入口由 `cmd/image.go` 统一封装（处理器函数形态）
- 批处理框架提供并发与进度显示：`cmd/image.go:433-494`

## 图片信息结构
- `ImageInfo` 字段：`pkg/imaging/verify.go:12-20`

## 支持格式
- 通过注册解码器支持 GIF/JPEG/PNG/WebP `pkg/imaging/verify.go:5-10`

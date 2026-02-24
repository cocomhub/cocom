# image 命令

- 功能：调整大小、裁剪、旋转、亮度对比、模糊/锐化、翻转、格式转换、验证
- 全局参数：`--format`、`--workers`、`--batch`、`--output` `cmd/image.go:426-431`
- 子命令：
  - `resize` `cmd/image.go:112-145`
  - `crop` `cmd/image.go:147-190`
  - `rotate` `cmd/image.go:192-219`
  - `adjust` `cmd/image.go:222-254`
  - `blur` `cmd/image.go:257-285`
  - `sharpen` `cmd/image.go:287-315`
  - `flip` `cmd/image.go:317-338`
  - `flop` `cmd/image.go:340-361`
  - `verify` `cmd/image.go:363-380`
  - `convert`（WebP 工具校验） `cmd/image.go:382-411,392-399`
- 批处理框架与进度显示：`cmd/image.go:433-494`

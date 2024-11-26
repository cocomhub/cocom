# 图片处理功能说明文档

## 依赖要求

1. Go 1.23+
2. WebP 工具（可选，用于 WebP 格式支持）：
   ```bash
   # 使用 cocom 安装（推荐）
   cocom install webp

   # 或手动安装：
   # Ubuntu/Debian
   sudo apt-get install webp

   # macOS
   brew install webp

   # Windows
   # 从 https://storage.googleapis.com/downloads.webmproject.org/releases/webp/index.html 下载
   ```

## 格式支持情况

| 功能 | JPEG | PNG | GIF | BMP | TIFF | WebP |
|------|------|-----|-----|-----|------|------|
| 读取 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| 写入 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓* |
| 调整大小 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| 裁剪 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| 旋转 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| 亮度/对比度 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| 翻转 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| 模糊/锐化 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| 格式转换 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓* |
| 完整性验证 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |

注意事项：
1. GIF 动画暂不支持，仅处理第一帧
2. WebP 格式需要安装 WebP 工具（cwebp、dwebp）
3. TIFF 多页面暂不支持，仅处理第一页
4. 格式转换时保持原图质量

## 命令行参数说明

全局参数：
- `-f, --format`: 指定输出格式 (jpg,png,gif,tiff,bmp,webp)
- `-b, --batch`: 启用批量处理模式
- `-n, --workers`: 指定并发工作协程数 (默认: 4)

### 图片处理示例

格式转换：
```bash
# 转换单个文件格式
cocom image convert input.jpg output.png png

# 批量转换格式
cocom image convert "./src/*.jpg" ./output/ webp --batch
# 将生成: ./output/input_convert.webp
```

调整图片大小：
```bash
# 输出到文件
cocom image resize input.jpg output.jpg 800 600

# 输出到目录（自动命名）
cocom image resize input.jpg ./output/ 800 600
# 生成: ./output/input_resize_w800h600.jpg
```

裁剪图片：
```bash
cocom image crop input.jpg ./output/ 100 100 400 300
# 生成: ./output/input_crop_x100y100w400h300.jpg
```

旋转图片：
```bash
cocom image rotate input.jpg ./output/ 90
# 生成: ./output/input_rotate_angle90.jpg
```

调整亮度和对比度：
```bash
cocom image adjust input.jpg ./output/ 0.5 1.5
# 生成: ./output/input_adjust_brightness0.5contrast1.5.jpg
```

批量处理图片：
```bash
# 处理单个目录
cocom image resize "./src/*.jpg" ./output/ 800 600 --batch

# 处理多个源
cocom image resize "./photos/*.jpg" "./images/*.png" ./output/ 800 600 --batch

# 批量转换格式
cocom image convert "./src/*.*" ./output/ webp --batch
```

验证图片完整性：
```bash
# 验证单个文件
cocom image verify input.jpg

# 批量验证
cocom image verify "./src/*.jpg" --batch
```

## 输出文件命名规则

- resize: {name}_resize_w{width}h{height}.{ext}
- crop: {name}_crop_x{x}y{y}w{width}h{height}.{ext}
- rotate: {name}_rotate_angle{angle}.{ext}
- adjust: {name}_adjust_brightness{b}contrast{c}.{ext}
- blur: {name}_blur_sigma{sigma}.{ext}
- sharpen: {name}_sharpen_sigma{sigma}.{ext}
- flip: {name}_flip.{ext}
- flop: {name}_flop.{ext}
- convert: {name}_convert.{new_ext}

## 错误处理

- 无效参数: 返回参数错误信息
- 文件不存在: 返回文件打开错误
- 格式不支持: 返回格式错误信息
- 处理失败: 返回具体错误原因
- 批处理错误: 返回所有失败文件的错误信息

## 性能优化

- 使用并发处理提高批处理性能
- 自动管理内存，避免内存泄漏
- 支持调整并发数控制资源使用
- 支持进度日志输出

## WebP 工具安装

1. 通过命令行安装（推荐）：
```bash
cocom install webp
```

2. 通过 HTTP API 获取安装脚本：
```bash
# Linux (Ubuntu)
curl http://localhost:8080/api/webp/install | bash

# Windows
curl http://localhost:8080/api/webp/install?os=windows | powershell

# macOS
curl http://localhost:8080/api/webp/install?os=darwin | bash
```

3. 手动安装：
- Ubuntu/Debian: `sudo apt-get install webp`
- CentOS/RHEL: `sudo yum install libwebp-tools`
- macOS: `brew install webp`
- Windows: 从 https://storage.googleapis.com/downloads.webmproject.org/releases/webp/index.html 下载

## 批量处理说明

所有图片处理命令都支持批量模式，使用 `--batch` 标志启用：

```bash
# 批量调整大小
cocom image resize "./src/*.jpg" ./output/ 800 600 --batch

# 批量格式转换
cocom image convert "./src/*.*" ./output/ webp --batch

# 批量验证（支持保存结果）
cocom image verify "./src/*.jpg" --batch -o results.json
```

批量处理特性：
- 支持多源输入（可使用多个 glob 模式）
- 自动创建输出目录
- 并发处理（默认 4 个工作协程）
- 保持原有的文件命名规则
- 支持格式转换
- 错误收集和报告
- 支持保存验证结果
- 支持进度日志输出

验证结果格式：
```json
[
   {
    "path": "1.webp",
    "image": {
      "path": "1.webp",
      "format": "webp",
      "width": 1024,
      "height": 1456,
      "size": 229394,
      "invalid": false
    },
    "timestamp": "2024-11-24T22:28:16.530112+08:00"
  }
]
```
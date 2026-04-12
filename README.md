# COCOM

一个基于 Go 语言开发的命令行工具和 API 服务器。

## 功能特性

### 漫画验证
- [漫画验证](docs/comic-verifier.md)
  - 支持图片完整性检查
  - 支持自动修复损坏图片
  - 支持断点续传
  - 支持优先级队列
  - 支持定时检查
  - 支持性能监控
  - 支持结果汇总和报告

### 图片处理
- [图片处理](docs/image-processor.md)
  - 支持多种图片格式
  - 支持批量处理
  - 支持格式转换
  - 支持完整性验证
  - 支持 WebP 格式
  - 支持性能监控
  - 支持结果收集

### 日志系统
- 完整的日志系统
  - 支持文件和控制台输出
  - 支持日志级别控制
  - 支持日志轮转
  - 支持 JSON 和控制台格式
  - 支持追踪 ID
  - 支持调用者信息
  - 支持进程 ID 和源 IP 记录

## 快速开始

### 安装 

```bash
make install
```

### 使用示例

```bash
# 启动服务器
cocom run server

# 打包单个漫画归档
cocom ar pack --cid 1001 --src-dir /data/gallery/[1001]\ Demo

# 查询归档记录
cocom ar query --cid 1001 --output json

# 验证漫画
cocom verify --pattern ".*" --auto-fix

# 查看验证进度
cocom verify status

# 查看验证报告
cocom verify report

# 定时检查
cocom verify schedule --pattern ".*" --interval 24h
```

## 开发

### 环境要求

- Go 1.26+
- Make
- Docker (可选)
- MongoDB
- WebP 工具 (可选，用于 WebP 格式支持)

### 开发命令

```bash
# 运行测试
make test

# 代码格式化
make fmt

# 静态检查
make lint

# 构建
make build

# 生成覆盖率报告
make cover
```

## 许可证

Apache License 2.0

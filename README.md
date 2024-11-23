# COCOM

一个基于 Go 语言开发的命令行工具和 API 服务器。

## 功能特性

- 完整的日志系统
  - 支持文件和控制台输出
  - 支持日志级别控制
  - 支持日志轮转
  - 支持 JSON 和控制台格式
  - 支持追踪 ID
  - 支持调用者信息
  - 支持进程 ID 和源 IP 记录
  
- 配置管理
  - 基于 Viper 的配置系统
  - 支持多种配置格式
  - 支持环境变量覆盖

- 构建系统
  - 支持多架构构建
  - 支持 Docker 镜像构建
  - 支持版本信息注入
  - 支持代码格式化和静态检查

## 快速开始

### 安装 

```bash
make install
```

### 使用示例

```bash
cocom run server
```

## 开发

### 环境要求

- Go 1.23+
- Make
- Docker (可选)

### 开发命令

运行测试
```bash
make test
```

代码格式化
```bash
make fmt
```

静态检查
```bash
make lint
```

构建
```bash
make build
```

生成覆盖率报告
```bash
make cover
```

## 许可证

Apache License 2.0


# arctl

arctl 是围绕 pkg/archive/manager 的命令行工具，用于执行存档的打包、解包、查询、多存储备份与一致性检查。

## 安装与构建

```bash
go build ./tools/arctl
```

## 全局参数
- --config 配置文件路径（可选，支持 viper）
- --output 输出格式：text|json，默认 text
- --verbose 详细日志

## 子命令

### pack

打包目录并注册索引。

```bash
arctl --config ./config.yaml --index-root ./data pack --src ./src/a --dest ./archives/a.7z --id 1001
```

### unpack

解包归档到目标目录（支持按 ID 或直接指定归档路径）。

```bash
arctl --config ./config.yaml unpack --id 1001 --out ./restore/a
arctl unpack --src ./archives/a.7z --out ./restore/a
```

### query

查询索引元数据。

```bash
arctl query --id 1001
arctl query --name a --limit 10
```

### backup

复制到目标存储并更新索引位置（基础版支持 LocalFS→LocalFS）。

```bash
arctl backup --id 1001 --to-root D:/backup --backend backupfs --prefix archives/data
```

### check

校验归档一致性并更新索引健康状态。

```bash
arctl check --id 1001
```

## 配置示例

```yaml
cocom:
  archive:
    cmd: "7z"
    password: "your-password"
    temp_path: "D:/temp/cocom-restore"
arctl:
  output: "text"
  verbose: false
  archive:
    manager:
      root: "./data"
      algorithm: double
      index:
        type: "file"
        fileStoreName: arctl-archive-manager-index
        path: "archive/index"
```

## 注意事项

- 归档 ID 为业务主键（int），需由调用方提供
- 归档/解包依赖 7z，可通过 cocom.archive.cmd 指定二进制路径
- 基础版本仅支持本地文件系统存储；云端后端可通过 Storage 接口后续扩展


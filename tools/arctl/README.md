# arctl

arctl 是围绕 pkg/archive/manager 的命令行工具，用于执行存档的打包、解包、查询、多存储备份与一致性检查。

`arctl` 与 `cocom ar` 现在复用同一套 archive manager 命令执行层；推荐优先使用 `cocom ar`，当你需要脱离主 CLI 单独调试 archive manager 时再使用 `arctl`。

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
arctl --config ./config.yaml pack --cid 1001 --src-dir ./src/a --dest-path ./archives/1001.cocoma
```

### unpack

解包归档到目标目录（支持按 ID 或直接指定归档路径）。

```bash
arctl --config ./config.yaml unpack --cid 1001 --out ./restore/a
arctl unpack --src ./archives/a.7z --out ./restore/a
```

### query

查询索引元数据。

```bash
arctl query --cid 1001
arctl query --name a --limit 10
```

### backup

复制到目标存储并更新索引位置（基础版支持 LocalFS→LocalFS）。

```bash
arctl backup --cid 1001 --backend backupfs --prefix archives/data
```

### check

校验归档一致性并更新索引健康状态。

```bash
arctl check --cid 1001
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
      rootDir: "./data"
      algorithm: double
      index:
        type: "file"
        fileStoreName: arctl-archive-manager-index
        fileStorePrefix: "archive/index"
```

## 注意事项

- `--cid` 与 `--id` 都会映射到 archive manager 的记录 ID；在 comicInfo 场景推荐直接使用 `--cid`
- 归档/解包依赖 7z，可通过 cocom.archive.cmd 指定二进制路径
- 支持 `archive.manager.index.type=mongo`，会写入 `comicInfo.archive` 并保留兼容字段
- `cocom ar` 复用主配置链路；`arctl` 继续保留为独立 archive manager 调试入口

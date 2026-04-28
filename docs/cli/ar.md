# cocom ar

`cocom ar` 是集成在主 CLI 中的单记录归档入口，适合针对单个 `cid` 执行打包、解包、查询、备份与校验，而不需要先启动 server。

## 全局参数

- `--config`：复用 `cocom` 主配置文件
- `--output`：输出格式，支持 `text` 与 `json`

## 子命令

### pack

针对单个 `cid` 打包目录并注册到 archive manager。

```bash
cocom ar pack --cid 1001 --src-dir /data/gallery/[1001]\ Demo
cocom ar pack --cid 1001 --src-dir /data/gallery/[1001]\ Demo --dest-path /data/archive/00/10/1001.cocoma
```

- 未显式指定 `--dest-path` 时，会按 `cocom.archive.path` 与 `cid` 自动推导归档路径
- 未显式指定 `--src-dir` 时，会尝试从 `comicInfo` 读取该 `cid` 的保存目录

### unpack

按 `cid`、归档 ID 或直接路径解包归档。

```bash
cocom ar unpack --cid 1001 --out /data/restore
cocom ar unpack --id 1001 --out /data/restore
cocom ar unpack --src /data/archive/00/10/1001.cocoma --out /data/restore
```

- `--src` 为空时，会优先从 archive manager 记录中解析主归档路径
- `--out` 为空且提供 `--cid` 时，会尝试从 `comicInfo` 推导默认保存目录

### query

查询单个归档记录或按名称过滤。

```bash
cocom ar query --cid 1001
cocom ar query --id 1001 --output json
cocom ar query --name "Demo" --limit 10
```

输出包含 archive ID、路径、校验摘要、位置列表与健康状态。

### backup

将单个归档复制到目标存储后端，并更新位置列表。

```bash
cocom ar backup --cid 1001 --backend archive-backup --prefix replicas
```

### check

校验主归档与副本状态，并把结果回写到 archive manager。

```bash
cocom ar check --cid 1001
```

## 配置示例

### file index

```yaml
cocom:
  storage:
    path: /opt/cocom/data/gallery
  archive:
    path: /opt/cocom/data/archive
    temp_path: /opt/cocom/data/archive-temp
    password: cocom
    cmd: 7z
    algorithm: single

archive:
  manager:
    algorithm: single
    replicates:
      - archive-backup
    index:
      type: file
      file_store_name: archive-index
      file_store_prefix: archive/index

storage:
  backends:
    - name: archive-index
      type: localfs
      metadata:
        root: /opt/cocom/data/archive-index
    - name: archive-backup
      type: localfs
      metadata:
        root: /opt/cocom/data/archive-backup
```

### mongo index

```yaml
archive:
  manager:
    index:
      type: mongo

mongo:
  host: 127.0.0.1:27017
  database: cocom

comic:
  mongo:
    database: cocom
    collections:
      comicInfo: comicInfo
```

- `type=mongo` 时，archive manager 会把索引写入 `comicInfo.archive`
- 回写只更新 `archive` 子树，同时保留兼容字段 `archive.path`、`archive.size`、`archive.algorithm`、`archive.md5` 与 `archive.manager`

## 与 arctl 的边界

- `cocom ar` 适合复用主配置、直接联动 `cocom.archive.*`、`storage.backends` 与 Mongo comicInfo
- `arctl` 适合独立调试 archive manager，本次也复用了同一套执行层与输出格式

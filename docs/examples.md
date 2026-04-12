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

## 单记录归档
- 直接使用主 CLI 打包单个 CID：`cocom ar pack --cid 1001 --src-dir /data/gallery/[1001]\ Demo`
- 查询该 CID 的归档定位：`cocom ar query --cid 1001 --output json`
- 校验并刷新主副本健康状态：`cocom ar check --cid 1001`
- 复制单条记录到备份后端：`cocom ar backup --cid 1001 --backend archive-backup --prefix replicas`
- 按 CID 解包到目标目录：`cocom ar unpack --cid 1001 --out /data/restore`

## 使用 BaiduPCS 归档后端
- 将 `storage.backends` 中的 `archive-baidu` 配置为 `type: baidupcs`，并为 `metadata.command` 指定 `BaiduPCS-Go` 路径
- 将 `archive.manager.index.type` 设为 `file`，并把 `archive.manager.index.fileStoreName` 指向 `archive-baidu`
- `cocom ar` 与 `arctl` 都可复用同一后端：

```yaml
storage:
  backends:
    - name: archive-baidu
      type: baidupcs
      metadata:
        command: /usr/local/bin/BaiduPCS-Go
        root: /apps/cocom/archive
        tempDir: /var/tmp/cocom-baidupcs
        timeout: 45s

archive:
  manager:
    index:
      type: file
      fileStoreName: archive-baidu
```

- 复制归档副本到百度网盘：`cocom ar backup --cid 1001 --backend archive-baidu --prefix rep`
- 独立调试 archive manager 时也可以继续使用：`arctl backup --cid 1001 --backend archive-baidu --prefix rep`

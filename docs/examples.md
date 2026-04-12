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

## 使用 BaiduPCS 归档后端
- 将 `storage.backends` 中的 `archive-baidu` 配置为 `type: baidupcs`，并为 `metadata.command` 指定 `BaiduPCS-Go` 路径
- 将 `archive.manager.index.type` 设为 `file`，并把 `archive.manager.index.fileStoreName` 指向 `archive-baidu`
- 使用 `arctl` 时可复用同一后端：

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

arctl:
  archive:
    manager:
      index:
        type: file
        fileStoreName: archive-baidu
```

- 复制归档副本到百度网盘：`arctl archive replicate --backend archive-baidu --prefix rep`

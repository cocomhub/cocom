# Archive Manager 快速上手

本模块提供统一的归档元数据管理能力：索引存储、归档与注册、校验与更新、复制到对象存储、保留策略等。下面示例演示从零到一的常见流程。

## 1. 创建索引存储
- 内存索引（适合测试或临时运行）：

```go
import (
    "context"
    "github.com/cocomhub/cocom/pkg/archive/manager"
)

ctx := context.Background()
index := manager.NewMemoryIndexStore()
```

- 文件存储索引（持久化到任意 Storage 后端，如本地文件系统）：

```go
import (
    "context"
    "github.com/cocomhub/cocom/pkg/archive/manager"
    "github.com/cocomhub/cocom/pkg/storage/localfs"
)

ctx := context.Background()
fs := localfs.New("D:/data")         // Storage 根目录
index := manager.NewIndexStoreFS(fs, "archives/index") // 索引前缀，JSON 元数据将落在此路径
```

索引接口定义与实现参考：[indexstore.go](file:///d:/workdir/leon/cocom/pkg/archive/manager/indexstore.go).

## 2. 初始化 Manager

```go
import "github.com/cocomhub/cocom/pkg/archive/manager"

m := manager.New(index) // 可选传入 Config，默认从 viper 读取
manager.Set(m)          // 全局设置 Manager，后续调用 Get() 即可获取
```

构造函数与核心方法参考：[manager.go](file:///d:/workdir/leon/cocom/pkg/archive/manager/manager.go).

## 3. 归档并注册

将源目录压缩到目标路径，并将生成的归档文件注册到索引中：

```go
import (
    "github.com/cocomhub/cocom/pkg/archive"
    "github.com/cocomhub/cocom/pkg/archive/manager"
)

srcDir   := "D:/source/comic-001"
destPath := "D:/archive/comic-001.7z"
replicate := true
replicatePrefix := "rep"

acfg := archive.Config{
    ID:       1001,        // 业务主键
    Password: "your-7z-password",
    // CmdPath/TempDir/ModTime 可按需设置
}

if err := manager.Archive(ctx, srcDir, destPath, replicate, replicatePrefix, acfg); err != nil {
    panic(err)
}
```

归档算法类型由 Manager 配置控制，默认 single，详见：[archiver.go](file:///d:/workdir/leon/cocom/pkg/archive/archiver.go) 与配置结构：[config.go](file:///d:/workdir/leon/cocom/pkg/archive/config.go).

执行函数参考：[executor.go](file:///d:/workdir/leon/cocom/pkg/archive/manager/executor.go).

## 4. 校验并更新

对已注册的归档执行一致性校验（例如 MD5）并将健康状态写回索引：

```go
import (
    "github.com/cocomhub/cocom/pkg/archive/manager"
)

report, err := manager.CheckAndUpdate(ctx, 1001)
if err != nil {
    panic(err)
}
// report.Healthy / report.Expected / report.Actual 等字段可用于观测与审计
```

实现参考：[checker.go](file:///d:/workdir/leon/cocom/pkg/archive/manager/checker.go).

## 5. 复制到对象存储

将归档文件复制到目标 Storage（如本地文件系统或其他对象存储），并在索引中记录副本位置与健康状态：

```go
import (
    "github.com/cocomhub/cocom/pkg/archive/manager"
    "github.com/cocomhub/cocom/pkg/storage/localfs"
)


dst := localfs.New("D:/backup-root")
backend := "backupfs"       // 自定义后端名，将写入 Locators 和 Health
prefix  := "archives"       // 目标存储中的前缀路径

n, err := manager.Replicate(ctx, dst, prefix, manager.IndexFilter{})
if err != nil {
    panic(err)
}
// n 为成功复制并更新索引的条数
```

方法参考：[replicate.go](file:///d:/workdir/leon/cocom/pkg/archive/manager/replicate.go).

## 6. 应用保留策略

当某后端副本健康后，可将本地原始归档文件删除，仅保留受信任后端的副本，并更新索引：

```go
import (
    "github.com/cocomhub/cocom/pkg/archive/manager"
)

removed, err := manager.ApplyRetention(ctx, "backupfs", manager.IndexFilter{})
if err != nil {
    panic(err)
}
// removed 为完成保留动作的条数
```

实现参考：[retention.go](file:///d:/workdir/leon/cocom/pkg/archive/manager/retention.go).

## 过滤器

在 List/Replicate/ApplyRetention 等场景中可通过过滤器选择对象：

```go
f := manager.IndexFilter{
    ID:   1001,
    Name: "comic-001",
    // Before/After 支持按时间筛选
}
```

结构定义参考：[types.go](file:///d:/workdir/leon/cocom/pkg/archive/manager/types.go).

## 其他能力
- 索引复制（跨索引后端同步元数据）：`m.Replicate(ctx, destIndex, filter)`，参考 [manager.go](file:///d:/workdir/leon/cocom/pkg/archive/manager/manager.go#L89-L109)。

## 最佳实践提示
- 在生产环境中建议使用文件存储索引并定期运行校验与复制任务。
- 保留策略会删除本地归档文件，请确保目标后端副本已健康并可恢复。

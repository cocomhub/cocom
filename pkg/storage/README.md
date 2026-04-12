# 存储子系统

本文档说明通用 URI 约定、LocalFS 与 BaiduPCS 后端的行为差异，以及归档场景下的使用注意事项。

## URI 约定
- 统一格式：`<type>://<name>/<key>`
- 字段含义：
  - `type`：存储后端类型，例如 `localfs`、`baidupcs`。
  - `name`：后端的人类可读名称；对 LocalFS 来说，优先使用构造时传入的 `name`，否则回退为根目录路径。
  - `key`：对象键，使用当前系统分隔符进行规范化后再转换为 `/` 分隔。
- 规范化规则：
  - 使用 `filepath.Clean` 归一化输入键，移除冗余 `.`、多余分隔等。
  - 对于清理后的 `.`，表示空键，URI 末尾不追加多余路径。
- 例子：
  - `localfs://testfs/a/b.txt`
  - 经过规范化后，`./a/../a//b.txt` 也会得到 `localfs://testfs/a/b.txt`

## LocalFS 后端
LocalFS 使用操作系统提供的“根目录句柄”进行沙箱化，确保所有文件操作都被限制在配置的根目录之内。

- 构造：`localfs.New(name, root)`
- 类型与名称：
  - `Type()` 返回 `localfs`
  - `Name()` 返回构造时的 `name` 或 `root` 路径
- 沙箱与安全：
  - 所有路径在调用前都会通过 `filepath.Clean` 规范化。
  - 使用根目录句柄打开/遍历文件，阻止 `..` 越权访问根目录之外的路径。
  - 测试覆盖了目录穿越防护.
- Put 写入行为：
  - 默认采用“新建且必须不存在”（`O_CREATE|O_WRONLY|O_EXCL`），可通过 `storage.WithOverwrite(true)` 允许覆盖（改为截断）。
  - 可选计算哈希作为 ETag：
    - `storage.WithSHA256()` 计算 SHA-256
    - `storage.WithMD5()` 计算 MD5
  - 返回的 `ObjectMeta` 包含 `Key/Size/ETag/ModTime`。
- 读取与元数据：
  - `Get` 返回 `io.ReadCloser` 与对象元数据。
  - `Stat` 返回对象元数据（大小与修改时间）。
- 列表：
  - `List(prefix)` 对目录进行递归遍历并返回对象列表；当 `prefix` 指向文件时返回单项。
- 复制与移动：
  - `Copy` 通过读写流实现在沙箱内复制，支持可选哈希与覆盖选项。
  - `Move` 在沙箱内重命名文件，必要时创建目标目录。

## BaiduPCS 后端
BaiduPCS 后端通过外部 `BaiduPCS-Go` 命令实现对象操作，适合把归档副本或文件型索引放到百度网盘。

- 构造：`baidupcs.New(name, baidupcs.Config{...})`
- 配置字段：
  - `Command`：`BaiduPCS-Go` 可执行文件路径
  - `Root`：远端根目录，所有逻辑 key 都会映射到该目录之下
  - `TempDir`：`Put`/`Get` 的本地受控临时目录
  - `WorkDir`：命令工作目录
  - `Timeout`：单次命令执行超时
  - `Args`：命令全局参数，例如 `--profile=default`
- 路径与安全：
  - 驱动复用 `storage.Path` 进行 key 规范化，保持 `<type>://<name>/<key>` URI 约定不变。
  - `../`、绝对越界等输入会被拒绝，确保逻辑 key 不能逃逸出配置的远端根目录。
- 对象读写：
  - `Put` 先把输入流写入本地临时文件，再调用命令上传，并在成功后回查远端元数据。
  - `Get` 先把远端对象下载到本地临时文件，再返回带自动清理能力的读取流。
  - `Copy` 与 `Move` 优先走远端命令，避免强制回源到本地。
- 诊断与错误语义：
  - 命令执行统一捕获标准输出、标准错误、退出码与超时。
  - “文件不存在”类诊断会映射为 `storage.ErrNotFound`，超时会映射为 `storage.ErrTransient`，权限类错误会映射为 `storage.ErrPermissionDenied`。
- 适用边界：
  - 依赖外部命令已正确安装并完成账号登录。
  - 大文件上传/下载会额外占用 `TempDir` 本地磁盘空间，建议预留与单文件峰值接近的空间。
  - 输出解析默认兼容 JSON 或制表符行格式；如自定义包装脚本，请保持其中一种稳定格式。

## 与归档管理的集成
- 在归档复制流程中，成功写入目标存储后会记录对象 URI，便于审计与排错。
- `archive.manager.index.type=file` 时，只要 `fileStoreName` 指向已注册的 `baidupcs` 实例，索引文件读写逻辑无需额外改造。

## 注意事项
- 请勿在 `key` 中使用绝对路径；一切路径均相对于配置的根目录解析。
- 即使传入了包含 `..` 的路径，也会被沙箱机制拦截，防止越权。
- 在生产中建议开启哈希计算用于校验与跨后端一致性检查。
- 对于 BaiduPCS 后端，建议把 `tempDir` 放在独立磁盘或容量充足的目录，并为命令配置合理的 `timeout`。

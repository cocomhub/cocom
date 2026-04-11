# 新增定时任务：Cocoma 文件归档作业 Spec

## Why
- 需要将离线生成的 .cocoma 文件自动归档到预期存档目录，形成标准化的入库流程与校验闭环。
- 通过调度与 UI 管控，确保高频（每分钟）增量处理，限制单次工作量上限，避免长期占用资源。

## What Changes
- 新增作业：CocomaArchiver（每分钟执行一次）。
  - 每次最多处理 10000 个文件；递归扫描配置目录下所有后缀为 .cocoma 的文件。
  - 解析文件对应的 cid（默认从文件名按正则提取，可配置）。
  - 计算文件 MD5，与 comicInfo 中记录的 md5 比对。
    - 一致：移动到“存档目录”的规范路径下。
    - 不一致：移动到“notmatch 目录”，避免后续扫描。
- 调度集成：使用已接入的 gocron/v2 与 gocron-ui，在 /admin/cron 中可见并可「Run」。
- 可配置项（viper）新增：
  - server.scheduler.cocoma_archiver.enabled（bool，默认 false）
  - server.scheduler.cocoma_archiver.cron（string，默认 "* * * * *" 每分钟；支持含/不含秒）
  - server.scheduler.cocoma_archiver.limit（int，默认 10000）
  - server.scheduler.cocoma_archiver.scan_dir（string，必填，扫描根目录）
  - server.scheduler.cocoma_archiver.archive_dir（string，必填，归档根目录）
  - server.scheduler.cocoma_archiver.notmatch_dir（string，必填，不匹配根目录）
  - server.scheduler.cocoma_archiver.cid_regex（string，默认 "^(\\d+)\\.cocoma$"）
- 安全与健壮性：
  - 仅处理位于 scan_dir 子树且扩展名为 .cocoma 的常规文件；忽略符号链接与非常规节点。
  - 移动采用 os.Rename；跨分区失败时回退为 copy+fsync+remove，确保原子性与数据完整性。
  - 归档路径按 cid 规则组织，确保幂等（目标存在则跳过或覆盖策略可配置，默认跳过并移动到 notmatch/exist 目录下）。
- 依赖接口：通过内部 service/dao 查询 comicInfo 的 md5（按 cid）。若 md5 字段不存在，作业记录警告并将文件移动到 notmatch。

## Impact
- Affected specs:
  - 定时归档能力：新增作业在 UI 管控下工作，满足分钟级自动入库与校验。
- Affected code:
  - 新增 pkg/cocomaarchiver 包：扫描、MD5 校验、移动、统计。
  - 在 internal/scheduler 注册 CocomaArchiver 任务并读取配置。
  - 配置默认值注入与文档说明。

## ADDED Requirements
### Requirement: Cocoma 归档作业
系统应提供每分钟执行一次的 Cocoma 文件归档作业，每次最多处理 10000 个文件。

#### Scenario: 成功归档
- WHEN 扫描到文件名符合 cid_regex 的 foo.cocoma，且其 MD5 与 comicInfo(cid) 的 md5 一致
- THEN 将文件移动至 archive_dir 的目标路径，并记录成功日志

#### Scenario: 校验失败
- WHEN 计算出的 MD5 与 comicInfo(md5) 不一致或查询不到 md5
- THEN 将文件移动至 notmatch_dir 对应路径，并记录原因，后续扫描不再处理该文件

#### Scenario: 限流与递归
- WHEN 单次运行发现超过 limit 的候选文件
- THEN 仅处理最多 limit 个，剩余待下次调度处理；扫描递归覆盖 scan_dir 全部子目录

#### Scenario: UI 管控
- WHEN 访问 /admin/cron
- THEN 能看到 CocomaArchiver 任务（名称/标签：archive,cocoma），可点击「Run」手动执行一次

## MODIFIED Requirements
无

## REMOVED Requirements
无


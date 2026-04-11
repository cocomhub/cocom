# 归档/恢复幂等与在线校验增强 Spec

## Why
避免重复归档与无意义的 I/O；确保归档前的有效性校验基于实时数据而非依赖落库的 VerifyInfo；通过接口抽象替代类型断言，提升代码健壮性与可维护性。

## What Changes
- 归档幂等：若目标归档文件已存在且 MD5 一致，则直接返回成功并提示“已存在”，不重复归档；若存在但 MD5 不一致，则继续归档（覆盖或按策略替换）
- 在线校验：归档时同步执行 verifyComic（实时校验图片有效性），不依赖既有 VerifyInfo 字段
- 接口替代断言：归档/恢复流程中的校验、元数据访问、图片迭代基于接口实现，避免 map[string]any 等类型断言
- 仍使用 httpwrap.ResponseInfo 统一返回；错误提示格式保持为「[$code] $msg」

## Impact
- Affected specs: GalleryDetail 归档/恢复流程、错误规范
- Affected code:
  - 后端：cmd/server/internal/comic/archive.go、storage.go、pkg/comic/handler.go、pkg/comic 服务/校验模块
  - 前端：无需变更交互文案，仅复用现有「[$code] $msg」提示

## ADDED Requirements
### Requirement: 归档幂等检查（存在即验）
系统 SHALL 在归档前执行目标文件存在性与 MD5 一致性检查：
- 若目标归档文件已存在，且其 MD5 与记录一致（或与实时计算的期望一致）
  - THEN 不再执行归档步骤，直接返回 head.code=0，head.msg 包含“已存在”
- 若目标归档文件已存在但 MD5 不一致
  - THEN 继续归档（覆盖或按实现策略替换），最终以新文件为准，并更新归档元信息

#### Scenario: 文件已存在且一致
- WHEN 调用归档接口且目标文件已存在，MD5 一致
- THEN 返回成功，不进行 7z 打包

### Requirement: 归档同步在线校验（不依赖 VerifyInfo）
系统 SHALL 在归档前调用在线校验组件对漫画图片进行实时校验：
- 使用 Verifier 接口：Verify(ctx, comic) -> VerifyResult（含 invalid 列表）
- 不读取/依赖 ComicInfo.VerifyInfo 字段
- 若存在异常：
  - 非 force：返回 head.code=-1001，body.invalid_images
  - force：继续归档并将 by_force=true 落库

#### Scenario: 在线校验失败
- WHEN 在线校验检测到异常图片
- THEN 非 force 返回 -1001；force 仍归档并记录 by_force=true

### Requirement: 接口化替代类型断言
系统 SHALL 在归档/恢复与其校验路径中，通过接口访问漫画信息与图片数据：
- 归档/恢复/校验流程禁止对 ComicInfo 做 map 或 JSON 的类型断言
- 使用已存在的 comic.Comic（GetID/GetImages/Object）与新增的 Verifier 接口协调完成校验与归档决策

#### Scenario: 代码检查
- WHEN 代码审查归档/恢复路径
- THEN 不出现对 map[string]any 的类型断言读取 verify/images/archive 字段

## MODIFIED Requirements
### Requirement: 归档接口行为
- 增加幂等：若目标文件存在且 MD5 一致，返回成功并提示“已存在”
- 增加在线校验：归档时强制执行 Verifier.Verify，同步判定是否允许继续
- 强制归档语义维持：?force=true 时跳过失败阻断并落库 by_force=true

## REMOVED Requirements
无


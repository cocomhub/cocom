# Mongo 版 IndexStore 规格

## Why
为存档索引提供基于 MongoDB 的持久化实现，满足两类场景：
- pkg 通用实现：独立集合管理 ArchiveMeta，具备通用 CRUD 与筛选能力
- cocom 特化实现：将索引数据嵌入现有 comicInfo 文档的 archive 字段，通过可注入处理函数实现字段映射与兼容

## What Changes
- 在 pkg/archive/manager 新增 MongoIndexStore，实现 IndexStore 接口（Create/Get/Update/Delete/List）
- 新增可注入策略（Option/Hook）：
  - 字段前缀与键映射（如 id/name/modTime… 对应 BSON 键，支持统一前缀 archive.）
  - 过滤器构建器：IndexFilter → bson.M
  - 编解码器：ArchiveMeta ↔ 子文档/整文档
- 提供便捷构造：NewMongoIndexStore 与 NewComicInfoArchiveIndexStore
- 在 manager 包内新增可注册的 IndexStore 工厂表：按配置 IndexConfig.Type 选择实现（memory/file/mongo…）
- 扩展 IndexConfig：Type 支持 “mongo”（不在 pkg 内耦合连接信息，由上层注入 collection）

## Impact
- 受影响规格：IndexStore 的新增实现；Manager 根据配置选择实现的装配方式
- 受影响代码：
  - [indexstore.go](file:///d:/workdir/leon/cocom/pkg/archive/manager/indexstore.go)
  - [manager.go](file:///d:/workdir/leon/cocom/pkg/archive/manager/manager.go)
  - [config.go](file:///d:/workdir/leon/cocom/pkg/archive/manager/config.go)
  - 新增 mongostore.go（MongoIndexStore）

## ADDED Requirements
### Requirement: Mongo 索引存储（通用）
系统 SHALL 提供通用 MongoIndexStore，通过注入的 collection 与选项完成 CRUD/List。

#### Scenario: 成功写入独立集合
- WHEN 使用 NewMongoIndexStore(coll) 创建存储，并传入默认映射（id→id，name→name，modTime→modTime）
- THEN Create 插入唯一 id 文档；Get/Update/Delete/List 正常工作；对 name/时间范围支持服务端过滤

### Requirement: ComicInfo 嵌入式存储（特化）
系统 SHALL 提供便捷构造 NewComicInfoArchiveIndexStore(coll)，把 ArchiveMeta 储存在 comicInfo 文档的 archive 字段；id 对应 cid。

#### Scenario: 成功写入 archive 子文档
- WHEN 使用 NewComicInfoArchiveIndexStore 并对 id 使用 cid，字段统一前缀为 archive.
- THEN Create 对 {cid} 的文档执行 $setOnInsert/$set 以初始化/写入 archive 子文档；Get 从 archive 解码；List 使用 archive.* 字段过滤

### Requirement: 可注入处理函数
系统 SHALL 允许通过 Option 指定：
- WithPrefix/WithIDField/WithNameField/WithModTimeField
- WithFilterBuilder(func(IndexFilter) bson.M)
- WithEncoder(func(ArchiveMeta) (any, error)) 与 WithDecoder(func(any) (ArchiveMeta, error))

#### Scenario: 覆盖默认过滤器
- WHEN 为 ComicInfo 场景注入 FilterBuilder 将 IndexFilter 映射为 {"cid":…, "archive.name":…, "archive.modTime":{$gt/lt…}}
- THEN List 能在服务端生效筛选并按 id 升序返回

## MODIFIED Requirements
### Requirement: Manager 构造与配置
补充 Manager 的装配逻辑：
- 若 IndexConfig.Type == "mongo"，则通过注册表查找并构建 IndexStore
- 若未注册对应类型，抛出明确错误信息（与 file 类型缺失时 panic 的行为一致）

## REMOVED Requirements
无

## Notes
- 推荐索引（独立集合场景）：
  - 唯一索引：{id: 1}
  - 常用查询：{name: 1}、{modTime: -1}
- ComicInfo 场景：建议在 comicInfo 上建立 {cid: 1} 唯一索引与 archive.name/modTime 的辅助索引（按需）
- 并发与幂等：Update/Set 使用原子操作；Create 在独立集合使用 InsertOne + 唯一索引；嵌入式可用 upsert 与匹配计数判断冲突


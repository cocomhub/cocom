# Tasks
- [x] 任务 1：定义可注入选项与构造器
  - [x] 子任务 1.1：定义 MongoOption 与内部 mapping（前缀/字段名/过滤器/编解码）
  - [x] 子任务 1.2：实现 NewMongoIndexStore(coll *mongo.Collection, opts ...MongoOption)
  - [x] 子任务 1.3：实现 NewComicInfoArchiveIndexStore(coll *mongo.Collection)（默认 cid 与 archive. 前缀）

- [x] 任务 2：实现 CRUD 与 List
  - [x] 子任务 2.1：Create（独立集合 InsertOne；嵌入式使用 $setOnInsert 或先查重后 $set）
  - [x] 子任务 2.2：Get（FindOne + Decoder；嵌入式使用投影/子文档提取）
  - [x] 子任务 2.3：Update（$set；不存在返回 ErrNotFound）
  - [x] 子任务 2.4：Delete（独立集合 DeleteOne；嵌入式 $unset archive 子文档）
  - [x] 子任务 2.5：List（服务端过滤 name/时间范围；按 id 升序）

- [x] 任务 3：Manager 工厂注册
  - [x] 子任务 3.1：新增注册表（map[string]func(IndexConfig) IndexStore）与 RegisterIndexStoreFactory
  - [x] 子任务 3.2：在 New() 中按 IndexConfig.Type 查找并构建（支持 memory/file/mongo）
  - [x] 子任务 3.3：mongo 未注册时报错信息清晰（与 file 缺失时一致的失败风格）

- [x] 任务 4：测试与验证
  - [x] 子任务 4.1：纯单元测试（编码/解码/过滤器映射）
  - [x] 子任务 4.2：可选集成测试（设置 MONGO_TEST=1，使用 mongowrap 默认连接，独立临时集合验证 CRUD/List）
  - [x] 子任务 4.3：ComicInfo 嵌入式用例（archive 子文档的 Create/Get/Update/List）

- [x] 任务 5：文档与示例
  - [x] 子任务 5.1：在 pkg/archive/manager README/注释中说明如何注册工厂并在 cocom 中对接 comicInfo
  - [x] 子任务 5.2：示例代码片段：
        RegisterIndexStoreFactory("mongo", func(cfg IndexConfig) IndexStore { return NewComicInfoArchiveIndexStore(mongo.ComicInfo()) })

- [x] 任务 6：编译修复与完整性补齐
  - [x] 子任务 6.1：按 `pkg/storage/types.go` 的真实结构修复 `Checksum` 与 `StorageLocator` 的编解码
  - [x] 子任务 6.2：补齐对字段映射能力的最小验证，确认通用模式与 `comicInfo.archive` 模式均能通过构建
  - [x] 子任务 6.3：重新评估本次 Mongo IndexStore 是否已按规格完整实现，若发现缺口则补测并修正

# Task Dependencies
- [任务 2] 依赖 [任务 1]
- [任务 3] 可与 [任务 1/2] 并行但需在集成验证前完成
- [任务 4] 依赖 [任务 1/2/3]
- [任务 5] 最后完成
- [任务 6] 依赖 [任务 1/2/3/4/5]

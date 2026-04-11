* [x] 提供通用 NewMongoIndexStore 并通过可注入选项完成字段映射/过滤/编解码

* [x] 提供 NewComicInfoArchiveIndexStore，默认使用 cid 与 archive. 前缀

* [x] CRUD 在独立集合与嵌入式两种模式下均可工作，错误语义符合 ErrNotFound/ErrAlreadyExists

* [x] List 支持 name/时间范围过滤并按 id 升序返回

* [x] Manager 能通过注册表按 IndexConfig.Type=="mongo" 装配实现，未注册时报错明确

* [x] 单元测试通过；当 MONGO\_TEST=1 时，集成测试通过（独立集合与 comicInfo 场景）

* [x] `MongoIndexStore` 与 `pkg/storage/types.go` 的真实结构保持一致，并通过包级构建与回归验证

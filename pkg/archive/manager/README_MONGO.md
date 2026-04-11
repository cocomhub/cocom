Mongo IndexStore 使用说明

- 注册工厂：

```go
// 在应用启动时注册 mongo 类型的 IndexStore
manager.RegisterIndexStoreFactory("mongo", func(cfg manager.IndexConfig) manager.IndexStore {
    return manager.NewComicInfoArchiveIndexStore(internalmongo.ComicInfo())
})
```

- 通用集合：

```go
st := manager.NewMongoIndexStore(db.Collection("archiveIndex"))
_ = st.Create(ctx, meta)
```

- 嵌入 comicInfo.archive：

```go
st := manager.NewComicInfoArchiveIndexStore(internalmongo.ComicInfo())
_ = st.Create(ctx, meta)
```


// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"
	"fmt"

	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/cmd/server/internal/onecomic"
	"github.com/cocomhub/cocom/cmd/server/internal/tag"
	comicpkg "github.com/cocomhub/cocom/pkg/comic"

	"github.com/gin-gonic/gin"
)

// InitE2EStorage 初始化 E2E 测试需要的内存存储并注入到各包默认存储中。
// E2E 测试（独立 module）无法直接导入 internal 包，所以通过 handler 包间接完成初始化。
func InitE2EStorage() *comicpkg.MemoryStorage {
	store := comicpkg.NewMemoryStorage()
	comic.SetDefaultStorage(store)
	tag.SetDefaultLikeStore(tag.NewMemoryLikeStore())
	tag.SetDefaultComicStore(store)
	tag.SetDefaultRelationStore(tag.NewMemoryRelationStore())
	return store
}

// RegisterE2ERoutesWithStore 使用已有 store 注册 E2E 路由，复用生产代码路由。
// 该函数会重新注入 store 到各包默认存储，确保路由 handler 使用正确的内存存储实例。
func RegisterE2ERoutesWithStore(ctx context.Context, r *gin.Engine, store *comicpkg.MemoryStorage) {
	comic.SetDefaultStorage(store)
	tag.SetDefaultLikeStore(tag.NewMemoryLikeStore())
	tag.SetDefaultComicStore(store)
	tag.SetDefaultRelationStore(tag.NewMemoryRelationStore())

	// API 路由 — 与生产代码共用（路由路径已包含 /api/ 前缀）
	registerAPIRoutes(r)

	// v2 API 路由 — 复用 pkg/comic.Handler.RegisterRoutes
	nhSrv, err := comicpkg.NewService(ctx, comic.NewTestStorage(store), "")
	if err != nil {
		panic(fmt.Errorf("new nhcomic service for e2e failed: %w", err))
	}
	comicpkg.NewHandler(ctx, nhSrv).RegisterRoutes(r.Group("/v2/api/nhcomic"))

	ocSrv, err := comicpkg.NewService(ctx, onecomic.NewTestStorage(store), "")
	if err != nil {
		panic(fmt.Errorf("new onecomic service for e2e failed: %w", err))
	}
	comicpkg.NewHandler(ctx, ocSrv).RegisterRoutes(r.Group("/v2/api/onecomic"))

	// g/api 路由 — gallery_detail 前端调用的 like/archive/restore
	galleryGroup := r.Group("/g")
	galleryGroup.POST("/api/like", gin.WrapF(LikeTag))
	galleryGroup.POST("/api/archive", gin.WrapF(AddLikeGroup))
	galleryGroup.POST("/api/restore", gin.WrapF(RestoreComic))
}

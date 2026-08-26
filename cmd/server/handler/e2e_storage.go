// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/cache"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/cmd/server/internal/custom"
	"github.com/cocomhub/cocom/cmd/server/internal/onecomic"
	"github.com/cocomhub/cocom/cmd/server/internal/setting"
	"github.com/cocomhub/cocom/cmd/server/internal/tag"
	"github.com/cocomhub/cocom/cmd/server/internal/video"
	comicpkg "github.com/cocomhub/cocom/pkg/comic"

	"github.com/gin-gonic/gin"
)

// Package handler 内部文件。下述函数仅供 E2E 测试（独立 module tests/e2e）通过桥接调用，
// 生产代码禁止调用：它们会以内存存储替换各包的默认存储（comic/tag/setting/video/custom），
// 并可能反复 Init 全局缓存，若在生产链路误用会破坏真实数据读写与缓存状态。
//
// 本文件不改变构建隔离（handler 包仍随主包构建），隔离仅靠文档约定与函数命名约束；
// 如未来需要在构建期隔离，可将其移到独立 _test 包或 build-tag 文件（当前不做）。

// InitE2EStorage 初始化 E2E 测试需要的内存存储并注入到各包默认存储中。
// E2E 测试（独立 module）无法直接导入 internal 包，所以通过 handler 包间接完成初始化。
func InitE2EStorage() *comicpkg.MemoryStorage {
	store := comicpkg.NewMemoryStorage()
	comic.SetDefaultStorage(store)
	tag.SetDefaultLikeStore(tag.NewMemoryLikeStore())
	tag.SetDefaultComicStore(store)
	tag.SetDefaultRelationStore(tag.NewMemoryRelationStore())
	tag.SetDefaultTagStore(tag.NewMemoryTagStore())
	setting.SetDefaultSettingsStore(setting.NewMemorySettingsStore())
	video.SetDefaultVideoStore(video.NewMemoryVideoStore())
	custom.SetDefaultCustomStore(custom.NewMemoryCustomStore())
	return store
}

// RegisterE2ERoutesWithStore 使用已有 store 注册 E2E 路由，复用生产代码路由。
// 该函数会重新注入 store 到各包默认存储，确保路由 handler 使用正确的内存存储实例。
// 根路径沿用当前全局 DefaultRootPaths（/data/cocom/...）——E2E 中图片读取路径不对齐时
// 请使用 RegisterE2ERoutesWithRoot 显式设置与 seed 一致的根路径。
func RegisterE2ERoutesWithStore(ctx context.Context, r *gin.Engine, store *comicpkg.MemoryStorage) {
	RegisterE2ERoutesWithRoot(ctx, r, store, rootPathsFromEnv())
}

// RegisterE2ERoutesWithRoot 使用已有 store 注册 E2E 路由，并显式设置漫画存储根路径。
// rootPaths 会被写入 api.DefaultRootPaths，与 fixtures.SeedE2EData 生成 mock 图片的
// 目录保持一致，使 view 层的 SaveDir/PageSavePath 正确解析到临时 gallery 根路径。
func RegisterE2ERoutesWithRoot(ctx context.Context, r *gin.Engine, store *comicpkg.MemoryStorage, rootPaths api.RootPaths) {
	comic.SetDefaultStorage(store)
	tag.SetDefaultLikeStore(tag.NewMemoryLikeStore())
	tag.SetDefaultComicStore(store)
	tag.SetDefaultRelationStore(tag.NewMemoryRelationStore())
	tag.SetDefaultTagStore(tag.NewMemoryTagStore())
	setting.SetDefaultSettingsStore(setting.NewMemorySettingsStore())
	video.SetDefaultVideoStore(video.NewMemoryVideoStore())
	custom.SetDefaultCustomStore(custom.NewMemoryCustomStore())

	// 图片根路径与 seed 对齐：E2E 不经过 server.Run()，隧道不存在时 DefaultRootPaths
	// 保持默认 /data/cocom/... 值，会导致图片服务 404。这里显式注入临时目录根路径。
	api.SetRootPaths(rootPaths)

	// API 路由 — 与生产代码共用（路由路径已包含 /api/ 前缀）。
	// E2E 测试由本地浏览器驱动，走 loopback，因此管理端传 allowRemote=false（默认仅 loopback）。
	registerAPIRoutes(r, false, "")

	cache.Init(ctx, 10*time.Minute, 1*time.Minute)

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

// rootPathsFromEnv 从 COCOM_STORAGE_* 环境变量（main_test.go TestMain 设置）构造 RootPaths，
// 缺省时回退到 api.DefaultRootPaths，保证旧调用方行为不变。
func rootPathsFromEnv() api.RootPaths {
	return api.RootPaths{
		SaveRoot:    envOr("COCOM_STORAGE_GALLERY", api.DefaultRootPaths.SaveRoot),
		ArchiveRoot: envOr("COCOM_STORAGE_ARCHIVE", api.DefaultRootPaths.ArchiveRoot),
		ArchiveTemp: envOr("COCOM_STORAGE_ARCHIVE_TEMP", api.DefaultRootPaths.ArchiveTemp),
	}
}

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

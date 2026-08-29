// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/cocomhub/cocom/cmd/server/internal/cache"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/pkg/download"
	"github.com/cocomhub/cocom/pkg/imaging/webp"
	"github.com/cocomhub/cocom/pkg/mongowrap"

	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/pkg/middlewares"

	"github.com/gin-gonic/gin"
)

func Init(ctx context.Context, r *gin.Engine) {
	cfg := config.Get()
	comic.Init(ctx, cfg.Comic.Download.MaxDownloadSize)
	// cocom.cache.* 使用 Go duration 字符串（如 "1m"），此处统一解析。
	// 裸数字会被拒绝（Validate 已拦截），避免 viper 把数字解释为纳秒。
	evictionInterval, err := time.ParseDuration(cfg.Cocom.Cache.EvictionInterval)
	if err != nil {
		panic(fmt.Errorf("cache evictionInterval invalid: %w", err))
	}
	cleanInterval, err := time.ParseDuration(cfg.Cocom.Cache.CleanInterval)
	if err != nil {
		panic(fmt.Errorf("cache cleanInterval invalid: %w", err))
	}
	cache.Init(ctx, evictionInterval, cleanInterval)
	download.Init(download.Config{
		DownloadDir: cfg.Download.DownloadDir,
		MaxRunning:  cfg.Download.MaxRunning,
		EnableProxy: cfg.Download.EnableProxy,
		ProxyURL:    cfg.Download.ProxyURL,
	})
	if err := mongowrap.Init(ctx, cfg.Mongo); err != nil {
		panic(fmt.Errorf("mongowrap init: %w", err))
	}
	registerAPIRoutes(r, cfg.Server.Admin.AllowRemote, cfg.Server.Admin.Token)
}

// registerAPIRoutes 注册 API 路由（生产和 E2E 共用）。
// allowRemote/adminToken 用于管理端（/api/admin）鉴权：默认仅 loopback；
// allow_remote=true 且配置 token 时校验 X-Admin-Token。E2E 传入 allowRemote=false。
func registerAPIRoutes(r gin.IRouter, allowRemote bool, adminToken string) {
	r.POST(webp.InstallScriptEndpoint, gin.WrapF(webp.HandleWebPInstall))
	r.GET(webp.InstallScriptEndpoint, gin.WrapF(webp.HandleWebPInstall))

	r.POST("/api/comic/addLikeGroup", gin.WrapF(AddLikeGroup))

	r.POST("/api/comic/saveComicInfo", gin.WrapF(SaveComicInfo))
	r.POST("/api/comic/getComicInfo", gin.WrapF(GetComicInfo))
	r.GET("/api/comic/getComicInfo", gin.WrapF(GetComicInfo))
	r.POST("/api/comic/tags/like", gin.WrapF(AddLikeTag))
	r.DELETE("/api/comic/tags/like", gin.WrapF(RemoveLikeTag))
	r.POST("/api/comic/download", gin.WrapF(DownloadComic))
	r.POST("/api/comic/restore", gin.WrapF(RestoreComic))

	r.POST("/api/comic/tags/aggregate", gin.WrapF(AggregateTags))
	r.GET("/api/comic/tags", gin.WrapF(GetTags))
	r.GET("/api/comic/tags/search", gin.WrapF(SearchTags))
	r.POST("/api/comic/tags/likeTag", gin.WrapF(LikeTag))
	r.DELETE("/api/comic/tags/likeTag", gin.WrapF(UnlikeTag))
	r.POST("/api/comic/tags/update", gin.WrapF(UpdateComicTags))
	r.GET("/api/comic/tags/search-unique", gin.WrapF(GetSearchUniqueTags))
	r.POST("/api/comic/tags/batch-add", gin.WrapF(BatchAddTagToComics))
	r.GET("/api/comic/tags/related", gin.WrapF(GetRelatedTags))
	r.GET("/api/search/autocomplete", gin.WrapF(SearchAutocomplete))
	r.GET("/api/comic/recommendations", GetRecommendations)
	r.POST("/api/comic/getComicPages", gin.WrapF(GetComicPages))
	r.POST("/api/comic/savePages", gin.WrapF(SavePages))
	r.POST("/api/comic/tags/relation", gin.WrapF(CreateTagRelation))
	r.DELETE("/api/comic/tags/relation", gin.WrapF(DeleteTagRelation))
	r.GET("/api/comic/tags/relation", gin.WrapF(GetTagRelations))

	// Admin 管理端：挂 AdminGuard（默认仅 loopback，allow_remote=true 且配置 token 时校验 X-Admin-Token）。
	// 该组内的写操作（如删除漫画）不可逆，禁止在无鉴权的情况下暴露给远程客户端。
	adminGroup := r.Group("/api/admin", middlewares.AdminGuard(allowRemote, adminToken))
	{
		adminGroup.POST("/comic/compare", gin.WrapF(CompareComics))
		adminGroup.POST("/comic/link", gin.WrapF(LinkComics))
		adminGroup.POST("/comic/unlink", gin.WrapF(UnlinkComics))
		adminGroup.GET("/comic/links", gin.WrapF(GetLinks))
		adminGroup.POST("/comic/delete", gin.WrapF(DeleteComic))
		adminGroup.POST("/cache/reset", gin.WrapF(ResetCache))
	}

	r.POST("/api/onecomic/saveComicInfo", gin.WrapF(SaveOneComicInfo))
	r.POST("/api/onecomic/getComicInfo", gin.WrapF(GetOneComicInfo))
	r.GET("/api/onecomic/getComicInfo", gin.WrapF(GetOneComicInfo))

	r.GET("/api/settings", gin.WrapF(GetSetting))
	r.POST("/api/settings", gin.WrapF(SetSetting))
	r.DELETE("/api/settings", gin.WrapF(DelSetting))

	r.POST("/api/video/saveVideoInfo", gin.WrapF(SaveVideoInfo))
	r.POST("/api/video/getVideoInfo", gin.WrapF(GetVideoInfo))
	r.GET("/api/video/getVideoInfo", gin.WrapF(GetVideoInfo))

	r.POST("/api/migrate/customLikeToTag", gin.WrapF(CustomLikeToTag))
}

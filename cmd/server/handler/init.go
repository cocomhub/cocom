// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"
	"fmt"

	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/pkg/download"
	"github.com/cocomhub/cocom/pkg/imaging/webp"
	"github.com/cocomhub/cocom/pkg/mongowrap"

	"github.com/gin-gonic/gin"
)

func Init(ctx context.Context, r *gin.Engine) {
	comic.Init(ctx)
	download.Init()
	if err := mongowrap.Init(); err != nil {
		panic(fmt.Errorf("mongowrap init: %w", err))
	}

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
	r.POST("/api/comic/tags/relation", gin.WrapF(CreateTagRelation))
	r.DELETE("/api/comic/tags/relation", gin.WrapF(DeleteTagRelation))
	r.GET("/api/comic/tags/relation", gin.WrapF(GetTagRelations))

	// Admin 漫画对比工具
	r.POST("/api/admin/comic/compare", gin.WrapF(CompareComics))
	r.POST("/api/admin/comic/link", gin.WrapF(LinkComics))
	r.POST("/api/admin/comic/unlink", gin.WrapF(UnlinkComics))
	r.GET("/api/admin/comic/links", gin.WrapF(GetLinks))
	r.POST("/api/admin/comic/delete", gin.WrapF(DeleteComic))

	r.POST("/api/cache/reset", gin.WrapF(ResetCache))

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

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"

	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/pkg/download"
	"github.com/cocomhub/cocom/pkg/imaging/webp"
	"github.com/cocomhub/cocom/pkg/mongowrap"

	"github.com/gin-gonic/gin"
)

func Init(ctx context.Context, r *gin.Engine) {
	comic.Init(ctx)
	download.Init()
	mongowrap.Init()

	r.POST(webp.InstallScriptEndpoint, gin.WrapF(webp.HandleWebPInstall))
	r.GET(webp.InstallScriptEndpoint, gin.WrapF(webp.HandleWebPInstall))

	r.POST("/api/comic/addLikeGroup", gin.WrapF(AddLikeGroup))

	r.POST("/api/comic/saveComicInfo", gin.WrapF(SaveComicInfo))
	r.POST("/api/comic/getComicInfo", gin.WrapF(GetComicInfo))
	r.GET("/api/comic/getComicInfo", gin.WrapF(GetComicInfo))
	r.POST("/api/comic/download", gin.WrapF(DownloadComic))
	r.POST("/api/comic/restore", gin.WrapF(RestoreComic))

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
}

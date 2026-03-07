/*
Copyright © 2023 suixibing <suixibing@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package handler

import (
	"context"

	"github.com/suixibing/cocom/cmd/server/internal/comic"
	"github.com/suixibing/cocom/pkg/download"
	"github.com/suixibing/cocom/pkg/imaging/webp"
	"github.com/suixibing/cocom/pkg/mongowrap"

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

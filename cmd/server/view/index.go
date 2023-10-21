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
package view

import (
	"fmt"

	"github.com/suixibing/cocom/cmd/server/internal/comic"
	"github.com/suixibing/cocom/pkg/clog"

	"github.com/gin-gonic/gin"
)

type GalleryIndexPage struct {
	PopularNow []*GalleryDetail
	NewUpdates []*GalleryDetail
}

func IndexPage(c *gin.Context) {
	infos, err := comic.GetLatestComicInfos(c, 20, 0)
	if err != nil {
		clog.Errorf(c, "get latest comic info failed. errmsg: %s", err)
		c.AbortWithError(-1, fmt.Errorf("get latest comic info failed. errmsg: %s", err))
		return
	}
	clog.Debugf(c, "get latest comic infos length[%d]", len(infos))

	indexInfo := &GalleryIndexPage{}
	limit := 5
	if len(infos) < limit {
		limit = len(infos)
	}
	for _, info := range infos[:limit] {
		indexInfo.PopularNow = append(indexInfo.PopularNow, &GalleryDetail{ComicInfo: *info})
	}

	limit = 20
	if len(infos) < limit {
		limit = len(infos)
	}
	for _, info := range infos[:limit] {
		indexInfo.NewUpdates = append(indexInfo.NewUpdates, &GalleryDetail{ComicInfo: *info})
	}

	c.HTML(200, "index.tpl", indexInfo)
}

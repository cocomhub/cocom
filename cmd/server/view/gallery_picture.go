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
	"net/http"
	"strconv"

	"github.com/suixibing/cocom/cmd/server/api"
	"github.com/suixibing/cocom/cmd/server/internal/comic"
	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/errwrap"

	"github.com/gin-gonic/gin"
)

func parseGalleryPicturePageArgs(c *gin.Context) (cid int, no int, err error) {
	cid, err = strconv.Atoi(c.Param("cid"))
	if err != nil {
		err = errwrap.ErrInvalidArgs.SetIErrF("parse cid failed: %s", err)
		return
	}

	no, err = strconv.Atoi(c.Param("no"))
	if err != nil {
		err = errwrap.ErrInvalidArgs.SetIErrF("parse pic no failed: %s", err)
		return
	}

	return
}

func GalleryPicturePage(c *gin.Context) {
	cid, no, err := parseGalleryPicturePageArgs(c)
	if err != nil {
		clog.Errorf(c, "parseGalleryPicturePageArgs failed: %#v", err)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	info := api.ComicInfo{}
	err = comic.GetComicInfo(c, cid, &info)
	if err != nil {
		clog.Errorf(c, "comic.GetComicInfo failed: %#v", err)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	c.File(info.PageSavePath(no))
}

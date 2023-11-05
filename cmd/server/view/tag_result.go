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
	"net/http"
	"strconv"

	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/errwrap"

	"github.com/gin-gonic/gin"
)

func parseTagResultPageArgs(c *gin.Context) (page int, tag string, url string, err error) {
	if len(c.Query("page")) != 0 {
		page, err = strconv.Atoi(c.Query("page"))
		if err != nil {
			err = errwrap.ErrInvalidArgs.SetIErrF("parse page failed: %s", err)
			return
		}
	}

	if page < 1 {
		page = 1
	}

	tag = c.Param("tag")
	if len(tag) == 0 {
		err = errwrap.ErrInvalidArgs.SetIErrF("tag type not found")
		return
	}

	name := c.Param("name")
	if len(name) == 0 {
		err = errwrap.ErrInvalidArgs.SetIErrF("tag name not found")
		return
	}
	url = fmt.Sprintf("/%s/%s/", tag, name)
	return
}
func TagResultPage(c *gin.Context) {
	page, tag, url, err := parseTagResultPageArgs(c)
	if err != nil {
		clog.Errorf(c, "parseTagResultPageArgs failed: %#v", err)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	indexInfo, err := NewGalleryIndexPage(c, c.Request.URL.Path, page, "tags.type", tag, "tags.url", url)
	if err != nil {
		clog.Errorf(c, "NewGalleryIndexPage failed: %#v", err)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	c.HTML(http.StatusOK, "index.tpl", indexInfo)
}

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

	"github.com/gin-gonic/gin"
	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/errwrap"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func parseSearchResultPageArgs(c *gin.Context) (page int, query string, err error) {
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

	query = c.Query("q")
	if len(query) == 0 {
		err = errwrap.ErrInvalidArgs.SetIErrF("query not found")
		return
	}
	clog.Infof(c, "search query: %s", query)
	return
}

func SearchResultPage(c *gin.Context) {
	page, query, err := parseSearchResultPageArgs(c)
	if err != nil {
		clog.Errorf(c, "parseSearchResultPageArgs failed: %#v", err)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	indexInfo, err := NewGalleryIndexPage(c, c.Request.URL.Path, page, "$or", []bson.M{
		{"title.english": bson.M{"$regex": primitive.Regex{Pattern: query, Options: "i"}}},
		{"title.japanese": bson.M{"$regex": primitive.Regex{Pattern: query, Options: "i"}}},
		{"title.pretty": bson.M{"$regex": primitive.Regex{Pattern: query, Options: "i"}}},
	})
	if err != nil {
		clog.Errorf(c, "NewGalleryIndexPage failed: %#v", err)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	indexInfo.SearchQuery = query

	c.HTML(http.StatusOK, "index.tpl", indexInfo)
}

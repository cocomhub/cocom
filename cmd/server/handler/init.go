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
	"net/http"

	"github.com/suixibing/cocom/cmd/server/internal/comic"
	"github.com/suixibing/cocom/pkg/download"
	"github.com/suixibing/cocom/pkg/mongowrap"

	"github.com/gin-gonic/gin"
)

func Init(ctx context.Context) {
	comic.Init(ctx)
	download.Init()
	mongowrap.Init()
}

func Register(ctx context.Context, r *gin.Engine) {
	Init(ctx)
	r.Group("/api").Handle(http.MethodPost, "*filepath", func(c *gin.Context) {
		Mux().ServeHTTP(c.Writer, c.Request)
	})
	r.Group("/api").Handle(http.MethodGet, "*filepath", func(c *gin.Context) {
		Mux().ServeHTTP(c.Writer, c.Request)
	})
	r.Group("/debug").Handle(http.MethodGet, "*filepath", func(c *gin.Context) {
		Mux().ServeHTTP(c.Writer, c.Request)
	})
}

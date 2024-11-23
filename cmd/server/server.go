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
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/suixibing/cocom/cmd/server/handler"
	"github.com/suixibing/cocom/cmd/server/view"
	"github.com/suixibing/cocom/pkg/clog"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

var server http.Server

func Run() {
	ctx := clog.NewTraceCtx("server")

	shutdownCh := make(chan context.Context)
	wg := sync.WaitGroup{}

	go func() {
		wg.Add(1)
		defer wg.Done()

		select {
		case <-shutdownCh:
			clog.Infof(ctx, "server shutdown start...")
			err := server.Shutdown(ctx)
			if err != nil {
				clog.Errorf(ctx, "server shutdown failed: %s", err)
				return
			}
			clog.Infof(ctx, "server shutdown")
		}
	}()

	r := gin.Default()
	view.Register(r)
	handler.Register(ctx, r)
	r.POST("/admin/server/shutdown", func(c *gin.Context) {
		ctx := c.Request.Context()
		select {
		case shutdownCh <- ctx:
			close(shutdownCh)
			c.JSON(0, "server shutdown start")
		default:
			c.AbortWithError(-1, errors.New("server shutdown failed"))
		}
	})

	addr := fmt.Sprintf("%s:%d", viper.GetString("host"), viper.GetInt32("port"))
	clog.Infof(ctx, "cocom server listening and serving HTTP on [%s]", addr)
	server = http.Server{Addr: addr, Handler: r}

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		clog.Fatalf(ctx, "server listen and serve failed: %s", err)
	}
	clog.Infof(ctx, "server stop listen")
	wg.Wait()
}

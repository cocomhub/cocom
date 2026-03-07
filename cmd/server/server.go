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
	"strings"
	"sync"
	"time"

	"github.com/suixibing/cocom/cmd/server/handler"
	"github.com/suixibing/cocom/cmd/server/internal/comic"
	"github.com/suixibing/cocom/cmd/server/internal/onecomic"
	"github.com/suixibing/cocom/cmd/server/view"
	"github.com/suixibing/cocom/pkg/clog"
	comicpkg "github.com/suixibing/cocom/pkg/comic"
	"github.com/suixibing/cocom/pkg/middlewares"

	"github.com/gin-contrib/graceful"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// BuildEngine 构建并返回 Gin 引擎（注册通用中间件、视图、旧版 API 桥接与健康探针）
func BuildEngine(ctx context.Context, shutdownCh chan context.Context) *gin.Engine {
	r := gin.Default()
	r.Use(middlewares.RequestID())
	r.Use(middlewares.AccessLog(viper.GetStringSlice("server.access_log.patterns")...))
	if viper.GetBool("server.cors.enabled") {
		r.Use(middlewares.CORS())
	}
	if viper.GetBool("server.gzip.enabled") {
		r.Use(gzip.Gzip(viper.GetInt("server.gzip.level")))
	}
	if viper.GetBool("server.ratelimit.enabled") {
		rps := viper.GetInt("server.ratelimit.rps")
		burst := viper.GetInt("server.ratelimit.burst")
		r.Use(middlewares.RateLimit(rps, burst))
	}
	// 页面与静态资源
	view.Register(r)
	pprofGroup := r.Group("/debug", middlewares.LocalGuard("debug.allow_remote"))
	pprof.RouteRegister(pprofGroup, "pprof")
	// 旧版 /api 与 /debug 转发到 net/http Mux
	handler.Init(ctx, r)
	// 健康/就绪探针
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/readyz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	// 管理端点：触发优雅关闭（可选）
	if shutdownCh != nil {
		r.POST("/admin/server/shutdown", func(c *gin.Context) {
			rc := c.Request.Context()
			token := viper.GetString("admin.token")
			if token != "" {
				if c.GetHeader("X-Admin-Token") != token {
					c.AbortWithStatus(http.StatusUnauthorized)
					return
				}
			} else {
				ip := c.ClientIP()
				if ip != "127.0.0.1" && ip != "::1" {
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
			}
			select {
			case shutdownCh <- rc:
				close(shutdownCh)
				c.JSON(0, "server shutdown start")
			default:
				c.AbortWithError(-1, errors.New("server shutdown failed"))
			}
		})
	}
	return r
}

func Run() {
	ctx := clog.NewTraceCtx("server")

	shutdownCh := make(chan context.Context)
	wg := sync.WaitGroup{}

	r := BuildEngine(ctx, shutdownCh)

	var err1, err2 error
	comic.NhcomicSrv, err1 = comicpkg.NewService(ctx, onecomic.NewStorage())
	comic.OnecomicSrv, err2 = comicpkg.NewService(ctx, comic.NewStorage())
	if err1 != nil || err2 != nil {
		clog.Fatalf(ctx, "new comic service failed. onecomic err(%v) nhcomic err(%v)", err1, err2)
	}

	comicpkg.NewHandler(context.Background(), comic.NhcomicSrv).RegisterRoutes(r.Group("/v2/api/onecomic"))
	comicpkg.NewHandler(context.Background(), comic.OnecomicSrv).RegisterRoutes(r.Group("/v2/api/nhcomic"))

	// graceful 多路监听
	opts := []graceful.Option{}
	timeout := viper.GetDuration("server.shutdown_timeout")
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	opts = append(opts, graceful.WithShutdownTimeout(timeout))

	httpAddr := viper.GetString("server.listen.http.addr")
	tlsCert := viper.GetString("server.listen.tls.cert")
	tlsKey := viper.GetString("server.listen.tls.key")
	unixPath := viper.GetString("server.listen.unix.path")

	if strings.TrimSpace(httpAddr) == "" {
		httpAddr = fmt.Sprintf("%s:%d", viper.GetString("host"), viper.GetInt32("port"))
	}

	if strings.TrimSpace(unixPath) != "" {
		opts = append(opts, graceful.WithUnix(unixPath))
	}

	if strings.TrimSpace(tlsCert) != "" && strings.TrimSpace(tlsKey) != "" {
		opts = append(opts, graceful.WithTLS(httpAddr, tlsCert, tlsKey))
		clog.Infof(ctx, "cocom server will serve HTTPS on [%s]", httpAddr)
	} else if strings.TrimSpace(httpAddr) != "" {
		opts = append(opts, graceful.WithAddr(httpAddr))
		clog.Infof(ctx, "cocom server will serve HTTP on [%s]", httpAddr)
	}
	if strings.TrimSpace(unixPath) != "" {
		clog.Infof(ctx, "cocom server will also serve on unix socket [%s]", unixPath)
	}

	gr, err := graceful.New(r, opts...)
	if err != nil {
		clog.Fatalf(ctx, "create graceful server failed: %v", err)
	}
	defer gr.Close()

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		wg.Add(1)
		defer wg.Done()
		select {
		case <-shutdownCh:
			clog.Infof(ctx, "server shutdown start...")
			cancel()
		}
	}()

	if err := gr.RunWithContext(runCtx); err != nil && err != context.Canceled && err != http.ErrServerClosed {
		clog.Fatalf(ctx, "server run failed: %v", err)
	}
	clog.Infof(ctx, "server stop listen")
	wg.Wait()
}

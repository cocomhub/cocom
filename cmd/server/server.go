// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cocomhub/cocom/cmd/server/handler"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/cmd/server/internal/onecomic"
	"github.com/cocomhub/cocom/cmd/server/internal/scheduler"
	"github.com/cocomhub/cocom/cmd/server/view"
	comicpkg "github.com/cocomhub/cocom/pkg/comic"
	"github.com/cocomhub/cocom/pkg/logging"
	"github.com/cocomhub/cocom/pkg/middlewares"
	ui "github.com/go-co-op/gocron-ui/server"

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
	r.Use(middlewares.AccessLog(ctx, viper.GetStringSlice("server.access_log.patterns")...))
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

func mountSchedulerAdminUI(r *gin.Engine, sched *scheduler.Scheduler) {
	if r == nil || sched == nil || sched.Core() == nil {
		return
	}
	port := viper.GetInt("port")
	u := ui.NewServer(sched.Core(), port)
	group := r.Group("/admin/cron", middlewares.LocalGuard("admin.allow_remote"))
	h := gin.WrapH(http.StripPrefix("/admin/cron", u.Router))
	group.Any("/*path", h)
}

func Run() {
	ctx := logging.NewTraceCtx("server")

	shutdownCh := make(chan context.Context)
	wg := sync.WaitGroup{}

	r := BuildEngine(ctx, shutdownCh)

	// 初始化并启动调度器（可选）
	var sched *scheduler.Scheduler
	if viper.GetBool("server.scheduler.enabled") {
		if s, err := scheduler.New(ctx); err != nil {
			slog.WarnContext(ctx, "init scheduler failed", slog.String("err", err.Error()))
		} else {
			if err := s.Start(ctx); err != nil {
				slog.WarnContext(ctx, "start scheduler failed", slog.String("err", err.Error()))
			} else {
				sched = s
				slog.InfoContext(ctx, "server scheduler started")
				scheduler.RegisterProbeComic(ctx, sched)
				scheduler.RegisterArchiveStatusChecker(ctx, sched)
				scheduler.RegisterCocomaArchiver(ctx, sched)
				mountSchedulerAdminUI(r, sched)
			}
		}
	}

	var err1, err2 error
	comic.NhcomicSrv, err1 = comicpkg.NewService(ctx, onecomic.NewStorage())
	comic.OnecomicSrv, err2 = comicpkg.NewService(ctx, comic.NewStorage())
	if err1 != nil || err2 != nil {
		slog.ErrorContext(ctx, "new comic service failed",
			slog.Any("onecomic_err", err1),
			slog.Any("nhcomic_err", err2))
		panic(fmt.Errorf("new comic service failed: NhcomicSrv=[%w] OnecomicSrv=[%w]", err1, err2))
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
		slog.InfoContext(ctx, "cocom server will serve HTTPS", slog.String("addr", httpAddr))
	} else if strings.TrimSpace(httpAddr) != "" {
		opts = append(opts, graceful.WithAddr(httpAddr))
		slog.InfoContext(ctx, "cocom server will serve HTTP", slog.String("addr", httpAddr))
	}
	if strings.TrimSpace(unixPath) != "" {
		slog.InfoContext(ctx, "cocom server will also serve on unix socket", slog.String("path", unixPath))
	}

	gr, err := graceful.New(r, opts...)
	if err != nil {
		slog.ErrorContext(ctx, "create graceful server failed", slog.String("err", err.Error()))
		panic(fmt.Errorf("create graceful server failed: %w", err))
	}
	defer gr.Close()

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		wg.Add(1)
		defer wg.Done()
		select {
		case <-shutdownCh:
			slog.InfoContext(ctx, "server shutdown start...")
			cancel()
		}
	}()

	if err := gr.RunWithContext(runCtx); err != nil && err != context.Canceled && err != http.ErrServerClosed {
		slog.ErrorContext(ctx, "server run failed", slog.String("err", err.Error()))
		panic(fmt.Errorf("server run failed: %w", err))
	}
	// 服务器关闭后停止调度器
	if sched != nil {
		if err := sched.Stop(ctx); err != nil {
			slog.WarnContext(ctx, "stop scheduler failed", slog.String("err", err.Error()))
		} else {
			slog.InfoContext(ctx, "server scheduler stopped")
		}
	}
	slog.InfoContext(ctx, "server stop listen")
	wg.Wait()
}

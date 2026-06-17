// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cocomhub/cocom/cmd/server/handler"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/cmd/server/internal/onecomic"
	"github.com/cocomhub/cocom/cmd/server/internal/scheduler"
	"github.com/cocomhub/cocom/cmd/server/view"
	"github.com/cocomhub/cocom/internal/config"
	comicpkg "github.com/cocomhub/cocom/pkg/comic"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	"github.com/cocomhub/cocom/pkg/logging"
	"github.com/cocomhub/cocom/pkg/middlewares"
	ui "github.com/go-co-op/gocron-ui/server"

	"github.com/gin-contrib/graceful"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

// BuildEngine 构建并返回 Gin 引擎（注册通用中间件、视图、旧版 API 桥接与健康探针）
func BuildEngine(ctx context.Context, cfg *config.Server, shutdownCh chan context.Context) *gin.Engine {
	r := gin.Default()
	r.MaxMultipartMemory = 10 << 20 // 10MB
	r.Use(middlewares.RequestID())
	r.Use(middlewares.MaxBodySize(10 << 20)) // 10MB
	r.Use(middlewares.AccessLog(ctx, cfg.AccessLog.Patterns...))
	if cfg.CORS.Enabled {
		r.Use(middlewares.CORS())
	}
	if cfg.Gzip.Enabled {
		r.Use(gzip.Gzip(cfg.Gzip.Level))
	}
	if cfg.RateLimit.Enabled {
		r.Use(middlewares.RateLimit(cfg.RateLimit.RPS, cfg.RateLimit.Burst))
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
			token := cfg.Admin.Token
			if token != "" {
				if c.GetHeader("X-Admin-Token") != token {
					httpwrap.GinRespondError(c, http.StatusUnauthorized, httpwrap.ErrCodeForbidden, "admin token mismatch")
					c.Abort()
					return
				}
			} else {
				ip := c.ClientIP()
				if ip != "127.0.0.1" && ip != "::1" {
					httpwrap.GinRespondError(c, http.StatusForbidden, httpwrap.ErrCodeForbidden, "only loopback allowed")
					c.Abort()
					return
				}
			}
			select {
			case shutdownCh <- rc:
				c.JSON(http.StatusOK, gin.H{"message": "server shutdown start"})
			default:
				httpwrap.GinRespondError(c, http.StatusConflict, httpwrap.ErrCodeInternal, "server shutdown already started")
				c.Abort()
				return
			}
		})
	}
	return r
}

func mountSchedulerAdminUI(r *gin.Engine, sched *scheduler.Scheduler) {
	if r == nil || sched == nil || sched.Core() == nil {
		return
	}
	cfg := config.Get()
	svrCfg := &cfg.Server

	// 从 Listen.HTTP.Addr 提取端口（唯一入口）
	port := 8080
	if _, portStr, err := net.SplitHostPort(svrCfg.Listen.HTTP.Addr); err == nil {
		if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
			port = p
		}
	}

	u := ui.NewServer(sched.Core(), port)
	group := r.Group("/admin/cron", middlewares.LocalGuard("admin.allow_remote"))
	h := gin.WrapH(http.StripPrefix("/admin/cron", u.Router))
	group.Any("/*path", h)
}

func Run() {
	ctx := logging.NewTraceCtx("server")

	shutdownCh := make(chan context.Context, 1)
	wg := sync.WaitGroup{}

	cfg := config.Get()
	r := BuildEngine(ctx, &cfg.Server, shutdownCh)

	// 初始化并启动调度器（可选）
	var sched *scheduler.Scheduler
	if cfg.Server.Scheduler.Enabled {
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
	comic.NhcomicSrv, err1 = comicpkg.NewService(ctx, comic.NewStorage())
	comic.OnecomicSrv, err2 = comicpkg.NewService(ctx, onecomic.NewStorage())
	if err1 != nil || err2 != nil {
		slog.ErrorContext(ctx, "new comic service failed",
			slog.Any("onecomic_err", err1),
			slog.Any("nhcomic_err", err2))
		panic(fmt.Errorf("new comic service failed: NhcomicSrv=[%w] OnecomicSrv=[%w]", err1, err2))
	}

	comicpkg.NewHandler(context.Background(), comic.NhcomicSrv).RegisterRoutes(r.Group("/v2/api/nhcomic"))
	comicpkg.NewHandler(context.Background(), comic.OnecomicSrv).RegisterRoutes(r.Group("/v2/api/onecomic"))

	// graceful 多路监听
	opts := []graceful.Option{}
	svrCfg := &cfg.Server
	timeout := svrCfg.ShutdownTimeout
	if timeout == "" || timeout <= "0" {
		timeout = "5s"
	}
	parsedTimeout, err := time.ParseDuration(timeout)
	if err != nil {
		parsedTimeout = 5 * time.Second
	}
	opts = append(opts, graceful.WithShutdownTimeout(parsedTimeout))

	httpAddr := svrCfg.Listen.HTTP.Addr
	tlsCert := svrCfg.Listen.TLS.Cert
	tlsKey := svrCfg.Listen.TLS.Key
	unixPath := svrCfg.Listen.Unix.Path

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

	wg.Go(func() {
		<-shutdownCh
		slog.InfoContext(ctx, "server shutdown start...")
		cancel()
	})

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

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cocomhub/cocom/cmd/server/api"
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
	// 不信任任何反向代理的 X-Forwarded-For / X-Real-IP 头，避免 ClientIP() 被伪造绕过 LocalGuard
	// （仅当部署在可信反代之后时才需按需添加其 CIDR）。
	if err := r.SetTrustedProxies(nil); err != nil {
		slog.WarnContext(ctx, "SetTrustedProxies failed", slog.String("err", err.Error()))
	}
	r.MaxMultipartMemory = 10 << 20 // 10MB
	r.Use(middlewares.RequestID())
	r.Use(middlewares.MaxBodySize(10 << 20)) // 10MB
	r.Use(middlewares.AccessLog(ctx, cfg.AccessLog.Patterns...))
	if cfg.CORS.Enabled {
		r.Use(middlewares.CORS(cfg.CORS))
	}
	if cfg.Gzip.Enabled {
		r.Use(gzip.Gzip(cfg.Gzip.Level))
	}
	if cfg.RateLimit.Enabled {
		r.Use(middlewares.RateLimit(cfg.RateLimit.RPS))
	}
	// 页面与静态资源
	view.SetAdminAllowRemote(cfg.Admin.AllowRemote)
	view.Register(r)
	// pprof 属管理面：统一 AdminGuard（allowRemote=false 或 token 为空时自动降级仅 loopback）。
	// 与 /api/admin、/admin/cron 一致语义：allow_remote=true 且配 token 才放行远程。
	pprofGroup := r.Group("/debug", middlewares.AdminGuard(cfg.Admin.AllowRemote, cfg.Admin.Token))
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
		var shutdownOnce sync.Once
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
				// netip 判定 loopback，覆盖 IPv4-mapped（::ffff:127.0.0.1）与 ::1/127.0.0.1。
				// BuildEngine SetTrustedProxies(nil)，ClientIP 即真实对端，非伪造头。
				if !isOurLoopback(ip) {
					httpwrap.GinRespondError(c, http.StatusForbidden, httpwrap.ErrCodeForbidden, "only loopback allowed")
					c.Abort()
					return
				}
			}
			var sent bool
			shutdownOnce.Do(func() {
				select {
				case shutdownCh <- rc:
					sent = true
				default:
				}
			})
			if sent {
				c.JSON(http.StatusOK, gin.H{"message": "server shutdown start"})
			} else {
				httpwrap.GinRespondError(c, http.StatusConflict, httpwrap.ErrCodeInternal, "server shutdown already started")
				c.Abort()
			}
		})
	}
	return r
}

// isOurLoopback 判断字符串 IP 是否为 loopback（netip.ParseAddr + IsLoopback）。
// 与 pkg/middlewares.isLoopback 语义一致；server 包内用于 shutdown 端点的 ClientIP 判定。
func isOurLoopback(ip string) bool {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	return addr.IsLoopback()
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
	// 调度器管理 UI 属管理面：统一 AdminGuard，与 /api/admin 及 /admin 一致语义。
	// allowRemote=false 或 token 为空时自动降级仅 loopback，避免无凭据远程裸奔。
	group := r.Group("/admin/cron", middlewares.AdminGuard(svrCfg.Admin.AllowRemote, svrCfg.Admin.Token))
	h := gin.WrapH(http.StripPrefix("/admin/cron", u.Router))
	group.Any("/*path", h)
}

func Run() error {
	ctx := logging.NewTraceCtx("server")

	// 监听中断/终止信号，使其经 ctx 触发 graceful shutdown，而非被信号直接杀进程。
	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	// 二次 Ctrl+C 说明：NotifyContext 在首次信号后已停止继续监听（其内部对信号注册执行 Stop），
	// 因此首次信号之后再次收到 SIGINT/SIGTERM 会被吞掉，不会强杀进程。
	// 若用户在关闭窗口时触发二次信号，需等待 shutdown_timeout 到期后进程自然退出；
	// 本项为纯注释说明（B 方案），刻意不实现“二次 Ctrl+C 强杀”，后续如需可再评估。
	defer stop()

	shutdownCh := make(chan context.Context, 1)
	wg := sync.WaitGroup{}

	cfg := config.Get()
	// 注入漫画存储根路径
	api.SetRootPaths(api.RootPaths{
		SaveRoot:    cfg.Cocom.Storage.Path,
		ArchiveRoot: cfg.Cocom.Archive.Path,
		ArchiveTemp: cfg.Cocom.Archive.TempPath,
	})
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
	comic.NhcomicSrv, err1 = comicpkg.NewService(ctx, comic.NewStorage(), config.Get().Download.DownloadDir)
	comic.OnecomicSrv, err2 = comicpkg.NewService(ctx, onecomic.NewStorage(), config.Get().Download.DownloadDir)
	if err1 != nil || err2 != nil {
		slog.ErrorContext(ctx, "new comic service failed",
			slog.Any("nhcomic_err", err1),
			slog.Any("onecomic_err", err2))
		return fmt.Errorf("new comic service failed: NhcomicSrv=[%w] OnecomicSrv=[%w]", err1, err2)
	}

	comicpkg.NewHandler(context.Background(), comic.NhcomicSrv).RegisterRoutes(r.Group("/v2/api/nhcomic"))
	comicpkg.NewHandler(context.Background(), comic.OnecomicSrv).RegisterRoutes(r.Group("/v2/api/onecomic"))

	// graceful 多路监听
	opts := []graceful.Option{}
	svrCfg := &cfg.Server
	parsedTimeout, err := time.ParseDuration(svrCfg.ShutdownTimeout)
	if err != nil || parsedTimeout <= 0 {
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
	// 三空兜底：addr 为空且无 TLS 且无 unix_path → 无任何监听地址，直接 fail-fast。
	// （config.Validate 的 addr 校验为启动前置；此处为 Run 组装期第二道防线，
	//  避免 graceful 无监听地址启动后进入静默不可用状态。unix_path 存在时允许 addr 为空。）
	hasListen := strings.TrimSpace(unixPath) != "" ||
		(strings.TrimSpace(tlsCert) != "" && strings.TrimSpace(tlsKey) != "") ||
		strings.TrimSpace(httpAddr) != ""
	if !hasListen {
		slog.ErrorContext(ctx, "listen config empty: set a non-empty server.listen.http.addr or unix_path/TLS")
		slog.InfoContext(ctx, "addr 为空但允许有 unix_path/TLS 场景：addr 为空 + unix_path 存在时 HTTP 将回退 :8080，建议显式配置 server.listen.http.addr")
		return fmt.Errorf("server.listen.http.addr 未配置（需显式监听地址或 unix_path/TLS）")
	}

	if strings.TrimSpace(unixPath) != "" {
		slog.InfoContext(ctx, "cocom server will also serve on unix socket", slog.String("path", unixPath))
	}

	gr, err := graceful.New(r, opts...)
	if err != nil {
		slog.ErrorContext(ctx, "create graceful server failed", slog.String("err", err.Error()))
		return fmt.Errorf("create graceful server failed: %w", err)
	}
	defer gr.Close()

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg.Go(func() {
		select {
		case <-shutdownCh:
			slog.InfoContext(ctx, "server shutdown start (admin endpoint)")
		case <-sigCtx.Done():
			slog.InfoContext(ctx, "server shutdown start (signal)")
		}
		cancel()
	})

	if err := gr.RunWithContext(runCtx); err != nil && err != context.Canceled && err != http.ErrServerClosed {
		slog.ErrorContext(ctx, "server run failed", slog.String("err", err.Error()))
		return fmt.Errorf("server run failed: %w", err)
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
	return nil
}

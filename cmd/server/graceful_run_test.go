// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/cocomhub/cocom/internal/config"
	"github.com/gin-contrib/graceful"
)

func testCfgGrace() *config.Server {
	cfg := config.Get()
	return &cfg.Server
}

func TestHTTPStartAndGracefulShutdown(t *testing.T) {
	skipIfNoMongo(t)

	cfg := config.Get()
	cfg.Server.Listen.HTTP.Addr = "127.0.0.1:0"
	cfg.Server.ShutdownTimeout = "500ms"

	shutdownCh := make(chan context.Context, 1)
	r := BuildEngine(context.Background(), testCfgGrace(), shutdownCh)

	// 预创建 listener：绑定 127.0.0.1:0 由 OS 分配端口，拿到实际地址用于就绪轮询
	//（替代原 time.Sleep 的不确定同步；WithListener 让服务器用该 listener 服务）。
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	gr, err := graceful.New(
		r,
		graceful.WithListener(ln),
		graceful.WithShutdownTimeout(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("graceful.New error: %v", err)
	}
	defer gr.Close()

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 收到 shutdown 信号后取消 runCtx（等价生产：shutdownCh 驱动优雅停机）
	go func() {
		select {
		case <-shutdownCh:
			cancel()
		case <-runCtx.Done():
		}
	}()

	errCh := make(chan error, 1)
	go func() {
		errCh <- gr.RunWithContext(runCtx)
	}()

	// 轮询就绪：healthz 可达即服务器已监听（有界，不走固定 Sleep）
	up := false
	for i := 0; i < 50 && !up; i++ {
		resp, rErr := http.Get("http://" + ln.Addr().String() + "/healthz")
		if rErr == nil {
			_ = resp.Body.Close()
			up = resp.StatusCode == http.StatusOK
		}
		if !up {
			time.Sleep(20 * time.Millisecond)
		}
	}
	if !up {
		t.Fatalf("server did not become ready on %s", ln.Addr())
	}

	shutdownCh <- context.Background()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("server exit error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("server did not shutdown in time")
	}
}

//go:build memory_storage_integration

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/cocomhub/cocom/cmd/server/internal/testutil"
	"github.com/cocomhub/cocom/internal/config"
	"github.com/gin-contrib/graceful"
)

func testCfgGrace() *config.ServerConfig {
	return testutil.TestServerConfig()
}

func TestHTTPStartAndGracefulShutdown(t *testing.T) {
	cfg := config.Get()
	cfg.Server.Listen.HTTP.Addr = "127.0.0.1:0"
	cfg.Server.ShutdownTimeout = "500ms"

	shutdownCh := make(chan context.Context)
	r := BuildEngine(context.Background(), testCfgGrace(), shutdownCh)

	gr, err := graceful.New(
		r,
		graceful.WithAddr(cfg.Server.Listen.HTTP.Addr),
		graceful.WithShutdownTimeout(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("graceful.New error: %v", err)
	}
	defer gr.Close()

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-shutdownCh
		cancel()
	}()

	errCh := make(chan error, 1)
	go func() {
		errCh <- gr.RunWithContext(runCtx)
	}()

	time.Sleep(100 * time.Millisecond)
	select {
	case shutdownCh <- context.Background():
	default:
		t.Fatalf("failed to send shutdown signal")
	}

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("server exit error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("server did not shutdown in time")
	}
}

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cocomhub/cocom/cmd/server/internal/scheduler"
	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/pkg/middlewares"
	"github.com/go-co-op/gocron/v2"
)

func TestHealthzReadyz(t *testing.T) {
	skipIfNoMongo(t)
	cfg := config.Get()
	// config.Get() 返回缓存单例：本用例不改全局值，build engine 仅读 Server；
	// healthz/readyz 不依赖任何状态，无需恢复。
	r := BuildEngine(context.Background(), &cfg.Server, nil)
	s := httptest.NewServer(r)
	defer s.Close()

	resp, err := http.Get(s.URL + "/healthz")
	if err != nil {
		t.Fatalf("healthz request error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz status = %d", resp.StatusCode)
	}
	if resp.Header.Get(middlewares.HeaderXRequestID) == "" {
		t.Fatalf("healthz missing X-Request-ID header")
	}

	resp2, err := http.Get(s.URL + "/readyz")
	if err != nil {
		t.Fatalf("readyz request error: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("readyz status = %d", resp2.StatusCode)
	}
	if resp2.Header.Get(middlewares.HeaderXRequestID) == "" {
		t.Fatalf("readyz missing X-Request-ID header")
	}
}

func TestAdminCronShowsArchiveStatusCheckerAndCanRun(t *testing.T) {
	skipIfNoMongo(t)
	cfg := config.Get()
	// 本用例会写 cfg.Server.Admin.AllowRemote/Token 到缓存单例：先备份，退出恢复
	oldAllow := cfg.Server.Admin.AllowRemote
	t.Cleanup(func() { cfg.Server.Admin.AllowRemote = oldAllow })
	r := BuildEngine(context.Background(), &cfg.Server, nil)

	sc, err := scheduler.New(context.Background())
	if err != nil {
		t.Fatalf("new scheduler err: %v", err)
	}
	defer func() { _ = sc.Stop(context.Background()) }()

	runCh := make(chan struct{}, 1)
	_, err = sc.Core().NewJob(
		gocron.CronJob("*/5 * * * * *", true),
		gocron.NewTask(func() {
			runCh <- struct{}{}
		}),
		gocron.WithName("ArchiveStatusChecker"),
		gocron.WithTags("archive", "check"),
	)
	if err != nil {
		t.Fatalf("new job err: %v", err)
	}

	if err := sc.Start(context.Background()); err != nil {
		t.Fatalf("start scheduler err: %v", err)
	}
	mountSchedulerAdminUI(r, sc)
	cfg.Server.Admin.AllowRemote = false

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/cron/api/jobs", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /admin/cron/api/jobs status = %d, want 200, body=%s", w.Code, w.Body.String())
	}

	var jobs []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &jobs); err != nil {
		t.Fatalf("decode jobs err: %v body=%s", err, w.Body.String())
	}
	var jobID string
	for _, job := range jobs {
		if job.Name == "ArchiveStatusChecker" {
			jobID = job.ID
			break
		}
	}
	if jobID == "" {
		t.Fatalf("ArchiveStatusChecker not found in jobs: %s", w.Body.String())
	}

	wRun := httptest.NewRecorder()
	reqRun := httptest.NewRequest(http.MethodPost, "/admin/cron/api/jobs/"+jobID+"/run", nil)
	reqRun.RemoteAddr = "127.0.0.1:12345"
	r.ServeHTTP(wRun, reqRun)
	if wRun.Code != http.StatusOK {
		t.Fatalf("POST /admin/cron/api/jobs/{id}/run status = %d, want 200, body=%s", wRun.Code, wRun.Body.String())
	}

	select {
	case <-runCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("ArchiveStatusChecker job was not triggered from /admin/cron")
	}
}

func TestAdminShutdownIsIdempotentAndReturnsValidStatus(t *testing.T) {
	skipIfNoMongo(t)
	cfg := config.Get()
	// 本用例写 cfg.Server.Admin.Token/AllowRemote 到缓存单例：备份并在测试后恢复，
	// 顺序化后不影响其他 BuildEngine 用例（Token 被清空、AllowRemote 被改 false）。
	oldToken := cfg.Server.Admin.Token
	oldAllow := cfg.Server.Admin.AllowRemote
	t.Cleanup(func() {
		cfg.Server.Admin.Token = oldToken
		cfg.Server.Admin.AllowRemote = oldAllow
	})
	shutdownCh := make(chan context.Context, 1)
	r := BuildEngine(context.Background(), &cfg.Server, shutdownCh)
	s := httptest.NewServer(r)
	defer s.Close()

	cfg.Server.Admin.Token = ""
	cfg.Server.Admin.AllowRemote = false

	resp, err := http.Post(s.URL+"/admin/server/shutdown", "application/json", nil)
	if err != nil {
		t.Fatalf("shutdown request error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("shutdown status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	select {
	case <-shutdownCh:
	default:
		t.Fatalf("shutdown signal was not sent to channel")
	}

	resp2, err := http.Post(s.URL+"/admin/server/shutdown", "application/json", nil)
	if err != nil {
		t.Fatalf("shutdown request error: %v", err)
	}
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("shutdown second status = %d, want %d", resp2.StatusCode, http.StatusConflict)
	}
}

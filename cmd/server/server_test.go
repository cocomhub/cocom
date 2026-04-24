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
	"github.com/cocomhub/cocom/pkg/middlewares"
	"github.com/go-co-op/gocron/v2"
	"github.com/spf13/viper"
)

func TestHealthzReadyz(t *testing.T) {
	r := BuildEngine(context.Background(), nil)
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
	r := BuildEngine(context.Background(), nil)

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
	viper.Set("admin.allow_remote", false)

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

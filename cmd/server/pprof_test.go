// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cocomhub/cocom/internal/config"
)

func TestPprofLocalAndRemote(t *testing.T) {
	skipIfNoMongo(t)
	cfg := config.Get()
	// 本用例会把 cfg.Server.Admin.AllowRemote 切成 true，退出恢复原值避免影响其他 BuildEngine 用例
	orig := cfg.Server.Admin.AllowRemote
	t.Cleanup(func() { cfg.Server.Admin.AllowRemote = orig })
	cfg.Server.Admin.AllowRemote = false
	r := BuildEngine(context.Background(), &cfg.Server, nil)

	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	req1.RemoteAddr = "127.0.0.1:12345"
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("local /debug/pprof/ status = %d, want 200", w1.Code)
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	req2.RemoteAddr = "8.8.8.8:12345"
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Fatalf("remote /debug/pprof/ status = %d, want 403", w2.Code)
	}

	// enable remote：注意 token 为空时 AdminGuard 仍会降级为「仅 loopback」（这是
	// AGENTS 明确的语义：allowRemote=false **或** token 为空 ⇒ 仅 loopback），
	// 所以这里必须同时给出 token 才能验证远程放行；仅切 AllowRemote=true 仍应 403。
	cfg.Server.Admin.AllowRemote = true
	r2 := BuildEngine(context.Background(), &cfg.Server, nil)

	// 2a) allow_remote=true 但 token 为空 ⇒ 仍降级仅 loopback ⇒ 远程 403。
	origToken := cfg.Server.Admin.Token
	cfg.Server.Admin.Token = ""
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	req3.RemoteAddr = "8.8.8.8:12345"
	r2.ServeHTTP(w3, req3)
	if w3.Code != http.StatusForbidden {
		t.Fatalf("allow_remote=true 且 token 为空时远程应降级 403，实际 = %d", w3.Code)
	}

	// 2b) allow_remote=true 且配置了 token、请求带正确 token ⇒ 200。
	const wantToken = "pprof-test-token"
	cfg.Server.Admin.Token = wantToken
	r3 := BuildEngine(context.Background(), &cfg.Server, nil)
	w4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	req4.RemoteAddr = "8.8.8.8:12345"
	req4.Header.Set("X-Admin-Token", wantToken)
	r3.ServeHTTP(w4, req4)
	if w4.Code != http.StatusOK {
		t.Fatalf("allow_remote=true + 正确 token 时远程应 200，实际 = %d", w4.Code)
	}
	cfg.Server.Admin.Token = origToken
}

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestRegisterShutdownRoute_AdminGuard 校验优雅关闭端点的鉴权与其余管理端点一致：
// allow_remote=false（或 token 为空）时仅 loopback 可访问；allow_remote=true 且配置 token
// 时远程必须携带正确的 X-Admin-Token。
//
// 直接调用 registerShutdownRoute（不经过 BuildEngine），因此不依赖 MongoDB——
// 避免落入 cmd/server 集成测试「无 Mongo 就静默 skip」的零覆盖陷阱。
func TestRegisterShutdownRoute_AdminGuard(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name        string
		allowRemote bool
		token       string
		remoteAddr  string
		headerToken string
		wantStatus  int
	}{
		{"allow_remote=false/token 空 + 远程 → 403", false, "", "203.0.113.9:5555", "", http.StatusForbidden},
		{"allow_remote=false/token 空 + loopback → 200", false, "", "127.0.0.1:5555", "", http.StatusOK},
		{"allow_remote=false/已配 token + 远程（即使带对 token）→ 403", false, "s3cret", "203.0.113.9:5555", "s3cret", http.StatusForbidden},
		{"allow_remote=false/已配 token + loopback → 200", false, "s3cret", "127.0.0.1:5555", "", http.StatusOK},
		{"allow_remote=true/已配 token + 远程错 token → 401", true, "s3cret", "203.0.113.9:5555", "wrong", http.StatusUnauthorized},
		{"allow_remote=true/已配 token + 远程缺 token → 401", true, "s3cret", "203.0.113.9:5555", "", http.StatusUnauthorized},
		{"allow_remote=true/已配 token + 远程对 token → 200", true, "s3cret", "203.0.113.9:5555", "s3cret", http.StatusOK},
		{"allow_remote=true/token 空 + 远程 → 403（无凭据不裸奔）", true, "", "203.0.113.9:5555", "", http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			shutdownCh := make(chan context.Context, 1)
			registerShutdownRoute(r, tc.allowRemote, tc.token, shutdownCh)

			req := httptest.NewRequest(http.MethodPost, "/admin/server/shutdown", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.headerToken != "" {
				req.Header.Set("X-Admin-Token", tc.headerToken)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body=%s)", w.Code, tc.wantStatus, w.Body.String())
			}

			// 放行时应真的投递了关闭信号；被拒时不得投递。
			delivered := len(shutdownCh) == 1
			if want := tc.wantStatus == http.StatusOK; delivered != want {
				t.Errorf("shutdown 信号投递 = %v, want %v", delivered, want)
			}
		})
	}
}

func TestRegisterShutdownRoute_SecondCallConflicts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	shutdownCh := make(chan context.Context, 1)
	registerShutdownRoute(r, false, "", shutdownCh)

	call := func() int {
		req := httptest.NewRequest(http.MethodPost, "/admin/server/shutdown", nil)
		req.RemoteAddr = "127.0.0.1:5555"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	if got := call(); got != http.StatusOK {
		t.Fatalf("first call status = %d, want 200", got)
	}
	if got := call(); got != http.StatusConflict {
		t.Errorf("second call status = %d, want 409", got)
	}
}

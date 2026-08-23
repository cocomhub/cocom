// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMiddlewares_RequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		// RequestID middleware uses requestid library with custom header key
		// The request ID may be set in the header value
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			t.Log("RequestID header not set (may need different key)")
		}
		c.JSON(http.StatusOK, gin.H{"id": rid})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMiddlewares_LocalGuard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := LocalGuard(false)
	if handler == nil {
		t.Error("LocalGuard should return a handler")
	}
}

// TestMiddlewares_LocalGuard_ForgedHeader 验证 I7 回归：
// 即使伪造 X-Real-IP / X-Forwarded-For: 127.0.0.1，guard 也必须拒绝（用 RemoteIP 判定真实对端）。
func TestMiddlewares_LocalGuard_ForgedHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(LocalGuard(false))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// 模拟远程客户端伪造 loopback 头 —— RemoteIP 只看 RemoteAddr（非 loopback），应被 403 拒绝。
	// 用 httptest.NewRequest 使 RemoteAddr 为 "192.0.2.1:1234"（非 loopback）。
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("X-Real-IP", "127.0.0.1")
	req.Header.Set("X-Forwarded-For", "127.0.0.1")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("forged loopback header: status = %d, want 403 (RemoteIP must win)", w.Code)
	}
}

// TestMiddlewares_AdminGuard 验证管理端鉴权中间件的四种语义：
// allowRemote=false 仅 loopback；allowRemote=true+token 校验 X-Admin-Token；
// allowRemote=true+token 为空退化为 loopback-only，避免无凭据裸奔。
func TestMiddlewares_AdminGuard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setup := func(guard gin.HandlerFunc) *gin.Engine {
		r := gin.New()
		r.Use(guard)
		r.GET("/admin/probe", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})
		return r
	}
	do := func(r *gin.Engine, remoteAddr, token string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/admin/probe", nil)
		req.RemoteAddr = remoteAddr
		if token != "" {
			req.Header.Set("X-Admin-Token", token)
		}
		r.ServeHTTP(w, req)
		return w.Code
	}

	t.Run("allowRemote false blocks remote even with token", func(t *testing.T) {
		r := setup(AdminGuard(false, "secret"))
		if got := do(r, "192.0.2.1:1234", "secret"); got != http.StatusForbidden {
			t.Errorf("remote + allowRemote=false: status = %d, want 403", got)
		}
	})
	t.Run("allowRemote false allows loopback", func(t *testing.T) {
		r := setup(AdminGuard(false, ""))
		if got := do(r, "127.0.0.1:8080", ""); got != http.StatusOK {
			t.Errorf("loopback + allowRemote=false: status = %d, want 200", got)
		}
	})
	t.Run("allowRemote true token empty falls back to loopback", func(t *testing.T) {
		r := setup(AdminGuard(true, ""))
		if got := do(r, "192.0.2.1:1234", ""); got != http.StatusForbidden {
			t.Errorf("remote + token empty: status = %d, want 403", got)
		}
		if got := do(r, "127.0.0.1:8080", ""); got != http.StatusOK {
			t.Errorf("loopback + token empty: status = %d, want 200", got)
		}
	})
	t.Run("allowRemote true token mismatch returns 401", func(t *testing.T) {
		r := setup(AdminGuard(true, "secret"))
		if got := do(r, "192.0.2.1:1234", "wrong"); got != http.StatusUnauthorized {
			t.Errorf("wrong token: status = %d, want 401", got)
		}
	})
	t.Run("allowRemote true token match returns 200", func(t *testing.T) {
		r := setup(AdminGuard(true, "secret"))
		if got := do(r, "192.0.2.1:1234", "secret"); got != http.StatusOK {
			t.Errorf("correct token: status = %d, want 200", got)
		}
	})
}

// TestMiddlewares_MaxBodySize_Chunked413 验证 S3 回归：
// 无 Content-Length 的请求（chunked/未知长度）超限时必须返回 413。
// handler 需将 MaxBytesReader 触发的错误经 c.Error() 上报，由中间件统一映射 413。
func TestMiddlewares_MaxBodySize_Chunked413(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MaxBodySize(1024)) // 1KB
	r.POST("/upload", func(c *gin.Context) {
		buf := make([]byte, 2048)
		_, readErr := c.Request.Body.Read(buf)
		if readErr != nil {
			_ = c.Error(readErr) // 上报给中间件链
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// 构造超 1KB 但无 Content-Length 的请求（模拟 chunked）
	body := strings.Repeat("x", 2048)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/upload", strings.NewReader(body))
	req.ContentLength = -1 // 强制未知长度路径
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("chunked oversized: status = %d, want 413", w.Code)
	}
}

// TestMiddlewares_MaxBodySize_ExplicitLength413 验证显式 Content-Length 超限被预检拦截（既有防护回归）。
func TestMiddlewares_MaxBodySize_ExplicitLength413(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MaxBodySize(1024))
	r.POST("/upload", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	body := strings.Repeat("x", 2048)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/upload", strings.NewReader(body))
	r.ServeHTTP(w, req) // ContentLength 由 strings.NewReader 自动设为 2048

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("explicit oversized: status = %d, want 413", w.Code)
	}
}

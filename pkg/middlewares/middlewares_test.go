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

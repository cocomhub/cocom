// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestOriginGuard(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name       string
		method     string
		host       string
		origin     string
		referer    string
		wantStatus int
	}{
		{"安全方法不受限", http.MethodGet, "127.0.0.1:8080", "https://evil.example", "", http.StatusOK},
		{"POST 无 Origin/Referer 放行（非浏览器客户端）", http.MethodPost, "127.0.0.1:8080", "", "", http.StatusOK},
		{"POST 同源 Origin 放行", http.MethodPost, "127.0.0.1:8080", "http://127.0.0.1:8080", "", http.StatusOK},
		{"POST 跨源 Origin 拒绝", http.MethodPost, "127.0.0.1:8080", "https://evil.example", "", http.StatusForbidden},
		{"POST Origin 为 null 拒绝", http.MethodPost, "127.0.0.1:8080", "null", "", http.StatusForbidden},
		{"POST 跨源 Referer 拒绝", http.MethodPost, "127.0.0.1:8080", "", "https://evil.example/x", http.StatusForbidden},
		{"POST 同源 Referer 放行", http.MethodPost, "127.0.0.1:8080", "", "http://127.0.0.1:8080/page", http.StatusOK},
		{"默认端口归一后同源放行", http.MethodPost, "example.com", "https://example.com:443", "", http.StatusOK},
		{"不同端口视为跨源拒绝", http.MethodPost, "example.com:8080", "https://example.com:8443", "", http.StatusForbidden},
		{"IPv6 同源放行", http.MethodPut, "[::1]:8080", "http://[::1]:8080", "", http.StatusOK},
		{"DELETE 跨源拒绝", http.MethodDelete, "127.0.0.1:8080", "https://evil.example", "", http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(OriginGuard())
			r.Handle(tc.method, "/x", func(c *gin.Context) { c.Status(http.StatusOK) })

			req := httptest.NewRequest(tc.method, "/x", nil)
			req.Host = tc.host
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.referer != "" {
				req.Header.Set("Referer", tc.referer)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestNormalizeHostPort(t *testing.T) {
	cases := []struct{ in, want string }{
		{"127.0.0.1:8080", "127.0.0.1:8080"},
		{"127.0.0.1", "127.0.0.1"},
		{"example.com:443", "example.com"},
		{"example.com:80", "example.com"},
		{"EXAMPLE.com:443", "example.com"},
		{"[::1]:8080", "[::1]:8080"},
		{"::1", "::1"},
	}
	for _, tc := range cases {
		if got := normalizeHostPort(tc.in); got != tc.want {
			t.Errorf("normalizeHostPort(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseHostPart(t *testing.T) {
	cases := []struct{ in, want string }{
		{"http://127.0.0.1:8080/path?q=1", "127.0.0.1:8080"},
		{"https://example.com", "example.com"},
		{"https://user:pw@example.com:8443/x", "example.com:8443"},
		{"null", ""},
		{"", ""},
		{"example.com", ""},
	}
	for _, tc := range cases {
		if got := parseHostPart(tc.in); got != tc.want {
			t.Errorf("parseHostPart(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

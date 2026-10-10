// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cocomhub/cocom/internal/config"
	"github.com/gin-gonic/gin"
)

// TestCORS_ThreeBranches 覆盖 CORS 中间件的三个分支：
// 1. 整体 *（含空值默认）→ AllowAllOrigins
// 2. 精确域名列表 → 仅反射允许的来源
// 3. 含 * 中缀（如 https://*.example.com）→ 显式 500，而非悄悄放行任意来源
func TestCORS_ThreeBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		cfg          config.CORS
		reqOrigin    string
		wantStatus   int
		wantAllowOri string // 期望的 Access-Control-Allow-Origin（空表示不反射）
	}{
		{
			name:         "空值默认 * → AllowAllOrigins",
			cfg:          config.CORS{Enabled: true, AllowOrigins: ""},
			reqOrigin:    "https://evil.example.com",
			wantStatus:   http.StatusOK,
			wantAllowOri: "*",
		},
		{
			name:         "整体 *（含空格）→ AllowAllOrigins",
			cfg:          config.CORS{Enabled: true, AllowOrigins: " * "},
			reqOrigin:    "https://any.example.com",
			wantStatus:   http.StatusOK,
			wantAllowOri: "*",
		},
		{
			name:         "精确来源 → 仅反射允许的来源",
			cfg:          config.CORS{Enabled: true, AllowOrigins: "https://a.example.com, https://b.example.com"},
			reqOrigin:    "https://a.example.com",
			wantStatus:   http.StatusOK,
			wantAllowOri: "https://a.example.com",
		},
		{
			name:         "精确来源列表拒绝未允许 origin → 403",
			cfg:          config.CORS{Enabled: true, AllowOrigins: "https://a.example.com"},
			reqOrigin:    "https://evil.example.com",
			wantStatus:   http.StatusForbidden, // gin-contrib/cors 对未允许 origin 直接 403
			wantAllowOri: "",
		},
		{
			name:         "* 中缀子域 → 显式 500 提示（不静默放行）",
			cfg:          config.CORS{Enabled: true, AllowOrigins: "https://*.example.com"},
			reqOrigin:    "https://sub.example.com",
			wantStatus:   http.StatusInternalServerError,
			wantAllowOri: "",
		},
		{
			name:         "含 * 的混合列表 → 显式 500",
			cfg:          config.CORS{Enabled: true, AllowOrigins: "https://a.com,*"},
			reqOrigin:    "https://a.com",
			wantStatus:   http.StatusInternalServerError,
			wantAllowOri: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(CORS(tt.cfg))
			r.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", tt.reqOrigin)
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body=%s)", w.Code, tt.wantStatus, w.Body.String())
			}
			gotAllowOri := w.Header().Get("Access-Control-Allow-Origin")
			if tt.wantAllowOri != "" {
				if gotAllowOri != tt.wantAllowOri {
					t.Errorf("Access-Control-Allow-Origin = %q, want %q", gotAllowOri, tt.wantAllowOri)
				}
			} else if gotAllowOri != "" {
				t.Errorf("unexpected Access-Control-Allow-Origin = %q", gotAllowOri)
			}
		})
	}
}

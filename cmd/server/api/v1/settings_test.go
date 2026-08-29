// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	"github.com/cocomhub/cocom/pkg/middlewares"
	"github.com/gin-gonic/gin"
)

// TestSettingsBindJSON_413 回归 maxbody 413：手写 handler 场景下 BindJSON 必须将
// MaxBytesError 上报至 c.Errors，使 MaxBodySize 中间件能映射 413（若用 ShouldBindJSON 会
// 立刻写 400 而覆盖 413）。此处直接用路由注册一个等价于 SetSettings 的 BindJSON handler，
// 验证 chunked/超大请求在 MaxBytesReader 触发后最终返回 413 而非 400。
func TestSettingsBindJSON_413(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middlewares.MaxBodySize(1024)) // 1KB 上限，强制触发读取超限
	r.POST("/api/settings", func(c *gin.Context) {
		var req api.SetSettingsRequest
		if err := c.BindJSON(&req); err != nil {
			httpwrap.GinRespondError(c, http.StatusBadRequest, -1, err.Error())
			return
		}
		httpwrap.GinRespondOK(c, "")
	})

	// 结构完全合法、仅体积超限的 JSON body（无 Content-Length → 走 MaxBytesReader 触发）
	body := `{"type":"t","settings":{"k":"` + strings.Repeat("v", 4096) + `"}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/settings", bytes.NewReader([]byte(body)))
	req.ContentLength = -1 // 无 Content-Length（chunked 语义）→ 走 MaxBytesReader 触发
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("chunked oversized + BindJSON: status = %d, want 413", w.Code)
	}
}

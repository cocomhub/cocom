// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/cocomhub/cocom/pkg/httpwrap"
)

// AdminGuard 管理端点鉴权中间件，语义与 /admin/server/shutdown 端点一致：
//   - allowRemote=false（默认）：仅放行 loopback（等价 LocalGuard(false)）；
//   - allowRemote=true 且 token 非空：校验 X-Admin-Token，不匹配返回 401；
//   - allowRemote=true 但 token 为空：退化为仅放行 loopback，避免无凭据裸奔。
//
// 与 LocalGuard 一样使用 RemoteIP 判定 loopback，避免伪造的 X-Real-IP/X-Forwarded-For 绕过。
func AdminGuard(allowRemote bool, token string) gin.HandlerFunc {
	if allowRemote && token != "" {
		want := token
		return func(c *gin.Context) {
			got := c.GetHeader("X-Admin-Token")
			// 恒定时间比较：长度不等直接失败（subtle.ConstantTimeCompare 返回 0），不会早退泄露长度差异。
			same := subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
			if !same {
				httpwrap.GinRespondError(c, http.StatusUnauthorized, httpwrap.ErrCodeForbidden, "admin token mismatch")
				c.Abort()
				return
			}
			c.Next()
		}
	}
	// token 为空（无论 allowRemote）→ 仅 loopback；allowRemote=false 时同样仅 loopback。
	return LocalGuard(false)
}

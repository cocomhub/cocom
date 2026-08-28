// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"net/http"
	"net/netip"

	"github.com/gin-gonic/gin"
)

// isLoopback 判断 IP 是否为 loopback 地址。
func isLoopback(ipStr string) bool {
	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return false
	}
	return addr.IsLoopback()
}

func LocalGuard(allowRemote bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if allowRemote {
			c.Next()
			return
		}
		// 使用 RemoteIP 而非 ClientIP，避免伪造的 X-Real-IP / X-Forwarded-For 头绕过 loopback 校验。
		// BuildEngine 已 SetTrustedProxies(nil)，RemoteIP 即真实对端地址。
		ip := c.RemoteIP()
		if !isLoopback(ip) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}

// LocalGuardFunc 与 LocalGuard 等价，但 allowRemote 为请求时读取的闭包（便于热改/多引擎）。
func LocalGuardFunc(allowRemote func() bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if allowRemote() {
			c.Next()
			return
		}
		// 语义与 LocalGuard 一致：使用 RemoteIP 判定 loopback。
		ip := c.RemoteIP()
		if !isLoopback(ip) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}

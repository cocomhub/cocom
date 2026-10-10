// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// OriginGuard 对「非安全方法」（POST/PUT/PATCH/DELETE）做同源校验，缓解 CSRF。
//
// 背景：写端点此前仅依赖 loopback 守卫，浏览器里的恶意站点仍可向 127.0.0.1:8080
// 发起跨站 POST（简单请求不会被 CORS 预检拦截），构成 CSRF。项目此前在模板里透出
// 但从不生成的 CSRF token 并不能提供任何保护。
//
// 规则：
//   - GET/HEAD/OPTIONS 直接放行（安全方法，不改变状态）；
//   - Origin 存在且为 "null"（沙箱 iframe / data: URL 等不透明源）→ 拒绝；
//   - Origin/Referer 存在但 host 与请求 Host 不一致 → 403；
//   - 两者都不存在 → 放行（非浏览器客户端，如 CLI/curl，不携带环境凭据，不构成 CSRF 载体）。
func OriginGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isSafeMethod(c.Request.Method) {
			c.Next()
			return
		}

		if origin := c.GetHeader("Origin"); origin != "" {
			if origin == "null" || !sameOriginHost(origin, c.Request.Host) {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Next()
			return
		}

		if referer := c.GetHeader("Referer"); referer != "" {
			if !sameOriginHost(referer, c.Request.Host) {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
		}

		c.Next()
	}
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

// sameOriginHost 判断 origin/referer 这类绝对 URL 的 host 是否与请求 Host 同源。
// 仅比较 host[:port]，并按默认端口归一（http:80 / https:443 视为无端口）。
func sameOriginHost(rawURL, requestHost string) bool {
	u := parseHostPart(rawURL)
	if u == "" {
		return false
	}
	return normalizeHostPort(u) == normalizeHostPort(requestHost)
}

// parseHostPart 从 "https://example.com:8443/path" 中取出 "example.com:8443"。
// 不引入 net/url 以保持判定严格：无法识别的输入返回空串（调用方据此拒绝）。
func parseHostPart(rawURL string) string {
	s := strings.TrimSpace(rawURL)
	i := strings.Index(s, "://")
	if i < 0 {
		return ""
	}
	rest := s[i+3:]
	if j := strings.IndexAny(rest, "/?#"); j >= 0 {
		rest = rest[:j]
	}
	// 去掉 userinfo。
	if at := strings.LastIndex(rest, "@"); at >= 0 {
		rest = rest[at+1:]
	}
	return strings.ToLower(rest)
}

func normalizeHostPort(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	host, port, err := net.SplitHostPort(h)
	if err != nil {
		return strings.Trim(h, "[]")
	}
	if port == "80" || port == "443" {
		return strings.Trim(host, "[]")
	}
	return net.JoinHostPort(strings.Trim(host, "[]"), port)
}

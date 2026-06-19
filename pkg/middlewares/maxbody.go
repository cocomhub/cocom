// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodySize 限制请求体大小（字节），超过则返回 413
func MaxBodySize(maxBytes int64) gin.HandlerFunc {
	if maxBytes <= 0 {
		maxBytes = 10 << 20 // 默认 10MB
	}
	return func(c *gin.Context) {
		// 在读取 body 之前检查 Content-Length，超过限制直接拒绝
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatus(http.StatusRequestEntityTooLarge)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

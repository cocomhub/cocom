// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodySize 限制请求体大小（字节），超过则返回 413。
// 对于显式 Content-Length 超限的请求在读取前直接 413；
// 对于 chunked / 未知长度的请求，由 MaxBytesReader 在读取过程中触发 MaxBytesError，
// 这里统一将其映射为 413。
func MaxBodySize(maxBytes int64) gin.HandlerFunc {
	if maxBytes <= 0 {
		maxBytes = 10 << 20 // 默认 10MB
	}
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatus(http.StatusRequestEntityTooLarge)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
		for _, gErr := range c.Errors {
			if isMaxBytesErr(gErr.Err) {
				// 不覆盖已有状态码（如 500）时写 413
				if c.Writer.Status() == http.StatusOK {
					c.AbortWithStatus(http.StatusRequestEntityTooLarge)
				}
				return
			}
		}
	}
}

func isMaxBytesErr(err error) bool {
	var mbe *http.MaxBytesError
	return errors.As(err, &mbe)
}

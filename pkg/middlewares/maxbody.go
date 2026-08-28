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
				// 统一映射 413：c.Errors 中的 MaxBytesError 即请求体超限（chunked/未知长度）。
				// 覆盖 bind 写下的 400（BindJSON 走 MustBindWith，读超限会先写 400）——
				// gin 的 responseWriter 允许在 body 未落盘前改状态码，此处必须为 413 才符合语义。
				c.AbortWithStatus(http.StatusRequestEntityTooLarge)
				return
			}
		}
	}
}

func isMaxBytesErr(err error) bool {
	var mbe *http.MaxBytesError
	return errors.As(err, &mbe)
}

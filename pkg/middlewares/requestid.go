// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/pkg/logging"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

func RequestID() gin.HandlerFunc {
	// 死键接线（铁律 2）：log.disableTraceID=true 时不再把 request ID 写入 ctx，
	// 使日志不打印 trace 字段。读取发生在请求时（config.Get 缓存已就绪），
	// config 未初始化（仅测试）时默认启用。
	traceEnabled := func() bool {
		cfg, err := config.GetE()
		if err != nil {
			return true
		}
		return !cfg.Log.DisableTraceID
	}
	return requestid.New(
		requestid.WithCustomHeaderStrKey(HeaderXRequestID),
		requestid.WithHandler(func(c *gin.Context, rid string) {
			if !traceEnabled() {
				return
			}
			if rid != "" {
				c.Request = c.Request.WithContext(logging.WithTraceID(c.Request.Context(), rid))
			}
		}),
	)
}

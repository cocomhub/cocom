// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func AccessLog(ctx context.Context, pathPatterns ...string) gin.HandlerFunc {
	slog.DebugContext(ctx, "patterns", slog.Any("patterns", pathPatterns))
	res := make([]*regexp.Regexp, 0, len(pathPatterns))
	for _, pattern := range pathPatterns {
		res = append(res, regexp.MustCompile(pattern))
	}
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		latency := time.Since(start)
		ip := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path
		errmsg := ""
		if len(c.Errors) > 0 {
			errmsg = strings.TrimSpace(c.Errors.ByType(gin.ErrorTypeAny).String())
		}

		for _, re := range res {
			if re.MatchString(path) {
				slog.InfoContext(c.Request.Context(), "access_log",
					slog.String("method", method),
					slog.String("operation", path),
					slog.Int("status", status),
					slog.String("latency", latency.String()),
					slog.String("client_ip", ip),
					slog.String("error", errmsg))
				return
			}
		}
	}
}

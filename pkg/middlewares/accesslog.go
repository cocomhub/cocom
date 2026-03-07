// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/cocomhub/cocom/pkg/clog"
	"github.com/gin-gonic/gin"
)

func AccessLog(pathPatterns ...string) gin.HandlerFunc {
	clog.Debugf(context.TODO(), "patterns: %+v", pathPatterns)
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
				clog.Infof(c.Request.Context(), "access_log method=%s path=%s status=%d latency=%s client_ip=%s error=%q",
					method, path, status, latency.String(), ip, errmsg)
				return
			}
		}
	}
}

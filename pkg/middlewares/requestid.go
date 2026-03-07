// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"github.com/cocomhub/cocom/pkg/clog"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

func RequestID() gin.HandlerFunc {
	return requestid.New(
		requestid.WithCustomHeaderStrKey(HeaderXRequestID),
		requestid.WithHandler(func(c *gin.Context, rid string) {
			if rid != "" {
				c.Request = c.Request.WithContext(clog.WithTraceID(c.Request.Context(), rid))
			}
		}),
	)
}

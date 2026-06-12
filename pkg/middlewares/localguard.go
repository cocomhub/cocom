// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"net/http"

	"github.com/cocomhub/cocom/internal/config"
	"github.com/gin-gonic/gin"
)

func LocalGuard(configKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := config.Get()
		if configKey == "admin.allow_remote" && cfg.Server.Admin.AllowRemote {
			c.Next()
			return
		}
		ip := c.ClientIP()
		if ip != "127.0.0.1" && ip != "::1" {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}

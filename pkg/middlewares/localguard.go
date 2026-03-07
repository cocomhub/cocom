// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func LocalGuard(configKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if viper.GetBool(configKey) {
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

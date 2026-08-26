// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/cocomhub/cocom/internal/config"
)

// CORS 根据配置创建 CORS 中间件。
// cfg 由调用方传入（通常从 config.Get().Server.CORS 获取）。
// 注意：AllowOrigins 中的 * 中缀通配符（如 https://*.example.com）会被 internal/config.Validate
// 在启动期 fail-fast，本函数分支仅作为防御（异常装配路径下直接返回 500 提示运维修正）。
func CORS(cfg config.CORS) gin.HandlerFunc {
	originStr := cfg.AllowOrigins
	if originStr == "" {
		originStr = "*"
	}

	if strings.TrimSpace(originStr) == "*" {
		// 整体 * → AllowAllOrigins
		return cors.New(cors.Config{
			AllowAllOrigins: true,
			AllowMethods:    splitCSV(cfg.AllowMethods, "GET,POST,PUT,DELETE,OPTIONS"),
			AllowHeaders:    splitCSV(cfg.AllowHeaders, "*"),
			ExposeHeaders:   splitCSV(cfg.ExposeHeaders, ""),
		})
	}
	if strings.Contains(originStr, "*") {
		// 防御分支：internal/config.Validate 已前置拦截 * 中缀（启动期 fail-fast）。
		// 此处仅对绕过 Validate 的异常装配路径保持显式 500，而非悄悄放行任意来源。
		return func(c *gin.Context) {
			c.AbortWithStatusJSON(500, gin.H{"error": "allow_origins 含 * 中缀通配符（如 https://*.example.com）不生效，请用完整域名列表或仅整体 *"})
		}
	}
	return cors.New(cors.Config{
		AllowOrigins:  splitCSV(originStr, ""),
		AllowMethods:  splitCSV(cfg.AllowMethods, "GET,POST,PUT,DELETE,OPTIONS"),
		AllowHeaders:  splitCSV(cfg.AllowHeaders, "*"),
		ExposeHeaders: splitCSV(cfg.ExposeHeaders, ""),
	})
}

// splitCSV 按逗号拆分并去掉空白。
func splitCSV(s, def string) []string {
	if strings.TrimSpace(s) == "" {
		if def == "" {
			return nil
		}
		s = def
	}
	var out []string
	for p := range strings.SplitSeq(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

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
func CORS(cfg config.CORS) gin.HandlerFunc {
	originStr := cfg.AllowOrigins
	if originStr == "" {
		originStr = "*"
	}
	methodStr := cfg.AllowMethods
	if methodStr == "" {
		methodStr = "GET,POST,PUT,DELETE,OPTIONS"
	}
	headerStr := cfg.AllowHeaders
	if headerStr == "" {
		headerStr = "*"
	}

	cc := cors.Config{}
	if strings.TrimSpace(originStr) == "*" {
		cc.AllowAllOrigins = true
	} else {
		var list []string
		for i := range strings.SplitSeq(originStr, ",") {
			i = strings.TrimSpace(i)
			if i != "" {
				list = append(list, i)
			}
		}
		cc.AllowOrigins = list
	}

	var methods []string
	for m := range strings.SplitSeq(methodStr, ",") {
		m = strings.TrimSpace(m)
		if m != "" {
			methods = append(methods, m)
		}
	}
	cc.AllowMethods = methods

	if strings.TrimSpace(headerStr) == "*" {
		cc.AllowHeaders = []string{"*"}
	} else {
		var headers []string
		for h := range strings.SplitSeq(headerStr, ",") {
			h = strings.TrimSpace(h)
			if h != "" {
				headers = append(headers, h)
			}
		}
		cc.AllowHeaders = headers
	}

	return cors.New(cc)
}

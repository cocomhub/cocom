// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func CORS() gin.HandlerFunc {
	origins := viper.GetString("server.cors.allow_origins")
	if origins == "" {
		origins = "*"
	}
	methods := viper.GetString("server.cors.allow_methods")
	if methods == "" {
		methods = "GET,POST,PUT,DELETE,OPTIONS"
	}
	headers := viper.GetString("server.cors.allow_headers")
	if headers == "" {
		headers = "*"
	}
	cfg := cors.Config{}
	if strings.TrimSpace(origins) == "*" {
		cfg.AllowAllOrigins = true
	} else {
		var list []string
		for _, p := range strings.Split(origins, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				list = append(list, p)
			}
		}
		cfg.AllowOrigins = list
	}
	var mlist []string
	for _, p := range strings.Split(methods, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			mlist = append(mlist, p)
		}
	}
	cfg.AllowMethods = mlist
	if strings.TrimSpace(headers) == "*" {
		cfg.AllowHeaders = []string{"*"}
	} else {
		var hlist []string
		for _, p := range strings.Split(headers, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				hlist = append(hlist, p)
			}
		}
		cfg.AllowHeaders = hlist
	}
	return cors.New(cfg)
}

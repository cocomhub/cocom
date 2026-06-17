// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"github.com/cocomhub/cocom/internal/config"
	"github.com/spf13/viper"
)

// TestServerConfig 从当前 viper 状态读取 ServerConfig 用于测试。
// 等价于被重构的 testCfg / testCfgMiddleware / testCfgPprof / testCfgGrace。
func TestServerConfig() *config.Server {
	return &config.Server{
		AccessLog: config.AccessLog{
			Patterns: viper.GetStringSlice("server.access_log.patterns"),
		},
		CORS: config.CORS{
			Enabled: viper.GetBool("server.cors.enabled"),
		},
		Gzip: config.Gzip{
			Enabled: viper.GetBool("server.gzip.enabled"),
			Level:   viper.GetInt("server.gzip.level"),
		},
		RateLimit: config.RateLimit{
			Enabled: viper.GetBool("server.ratelimit.enabled"),
			RPS:     viper.GetInt("server.ratelimit.rps"),
			Burst:   viper.GetInt("server.ratelimit.burst"),
		},
	}
}

// TestServerConfigMinimal 返回最小化 ServerConfig（仅关闭限流），
// 适用于 settings_integration_test 等不需要中间件的场景。
func TestServerConfigMinimal() *config.Server {
	return &config.Server{
		RateLimit: config.RateLimit{
			Enabled: false,
		},
	}
}

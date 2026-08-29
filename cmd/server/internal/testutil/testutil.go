// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package testutil

import "github.com/cocomhub/cocom/internal/config"

// TestServerConfigMinimal 返回最小化 ServerConfig（仅关闭限流），
// 适用于 settings_integration_test 等不需要中间件的场景。
func TestServerConfigMinimal() *config.Server {
	return &config.Server{
		RateLimit: config.RateLimit{
			Enabled: false,
		},
	}
}

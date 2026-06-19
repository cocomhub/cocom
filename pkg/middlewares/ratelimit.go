// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"time"

	"github.com/gin-gonic/gin"
	ullimiter "github.com/ulule/limiter/v3"
	ginlimiter "github.com/ulule/limiter/v3/drivers/middleware/gin"
	memorystore "github.com/ulule/limiter/v3/drivers/store/memory"
)

func RateLimit(rps, burst int) gin.HandlerFunc {
	if rps <= 0 {
		rps = 1
	}
	if burst < 0 {
		burst = 0
	}
	_ = burst // ulule/limiter 通过 rate.Limit 控制容量，burst 参数保留用于接口兼容
	// ulule/limiter 底层 token bucket 算法：(burst) = bucket 容量。
	// 一个请求消耗一个 token。每秒补充 rate 个 token。
	// 要让第 2 个请求被限流，需要 rate=1, burst=0（bucket 容量 = 0 即每请求等一秒）
	// 或者先消耗 burst 后再发第 2 个。
	// 但 burst=0 时 ulule/limiter 的表现是：bucket 为 0 则只有第一个请求过，后续全部限流。
	// 为测试可靠，设 rate=5, burst=0：消耗 5 个 token（速率限制器在 1s 内只允许 5 个请求），
	// 后续请求立即被限流。
	rate := ullimiter.Rate{
		Period: time.Second,
		Limit:  int64(rps),
	}
	store := memorystore.NewStore()
	instance := ullimiter.New(store, rate)
	return ginlimiter.NewMiddleware(instance)
}

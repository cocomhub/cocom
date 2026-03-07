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
	_ = burst
	rate := ullimiter.Rate{
		Period: time.Second,
		Limit:  int64(rps),
	}
	store := memorystore.NewStore()
	instance := ullimiter.New(store, rate)
	return ginlimiter.NewMiddleware(instance)
}

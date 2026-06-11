// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cocomhub/cocom/internal/config"
	"github.com/spf13/viper"
)

func testCfgPprof() *config.ServerConfig {
	return &config.ServerConfig{
		AccessLog: config.AccessLogCfg{
			Patterns: viper.GetStringSlice("server.access_log.patterns"),
		},
		CORS: config.CORSCfg{
			Enabled: viper.GetBool("server.cors.enabled"),
		},
		Gzip: config.GzipCfg{
			Enabled: viper.GetBool("server.gzip.enabled"),
			Level:   viper.GetInt("server.gzip.level"),
		},
		RateLimit: config.RateLimitCfg{
			Enabled: viper.GetBool("server.ratelimit.enabled"),
			RPS:     viper.GetInt("server.ratelimit.rps"),
			Burst:   viper.GetInt("server.ratelimit.burst"),
		},
	}
}

func TestPprofLocalAndRemote(t *testing.T) {
	viper.Set("debug.allow_remote", false)
	r := BuildEngine(context.Background(), testCfgPprof(), nil)

	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	req1.RemoteAddr = "127.0.0.1:12345"
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("local /debug/pprof/ status = %d, want 200", w1.Code)
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	req2.RemoteAddr = "8.8.8.8:12345"
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Fatalf("remote /debug/pprof/ status = %d, want 403", w2.Code)
	}

	// enable remote and verify 200
	viper.Set("debug.allow_remote", true)
	r2 := BuildEngine(context.Background(), testCfgPprof(), nil)
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	req3.RemoteAddr = "8.8.8.8:12345"
	r2.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("remote /debug/pprof/ with allow_remote=true status = %d, want 200", w3.Code)
	}
}

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"
)

// TestInitSetsDefaults 验证 Init() 后所有预期 key 都有非零默认值。
func TestInitSetsDefaults(t *testing.T) {
	mgr := New()

	tests := []struct {
		key   string
		name  string
		check func(v any) bool
	}{
		{"cocom.storage.path", "storage path", notEmpty},
		{"cocom.archive.path", "archive path", notEmpty},
		{"cocom.archive.temp_path", "archive temp path", notEmpty},
		{"archive.password", "archive password", notEmpty},
		{"archive.cmd", "archive cmd", notEmpty},
		{"archive.replicate", "archive replicate", isFalse},
		{"server.listen.http.addr", "listen addr", notEmpty},
		{"server.access_log.patterns", "access log patterns", notEmptySlice},
		{"server.cors.enabled", "cors enabled", isFalse},
		{"server.cors.allow_origins", "cors allow origins", notEmpty},
		{"server.ratelimit.enabled", "ratelimit enabled", isFalse},
		{"server.ratelimit.rps", "ratelimit rps", positiveInt},
		{"server.scheduler.enabled", "scheduler enabled", isFalse},
		{"server.scheduler.timezone", "scheduler timezone", notEmpty},
		{"recommend.limit", "recommend limit", positiveInt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mgr.Viper().Get(tt.key)
			if !tt.check(got) {
				t.Errorf("mgr.Get(%q) = %v (%T), want to pass check", tt.key, got, got)
			}
		})
	}
}

// TestGetReturnsValidConfig 验证 Get() 返回的 Config 结构体各字段合法。
func TestGetReturnsValidConfig(t *testing.T) {
	mgr := New()
	cfg := mgr.Get()

	if cfg.Cocom.Storage.Path == "" {
		t.Error("Config.Cocom.Storage.Path is empty")
	}
	if cfg.Cocom.Archive.Path == "" {
		t.Error("Config.Cocom.Archive.Path is empty")
	}
	if cfg.Cocom.Archive.TempPath == "" {
		t.Error("Config.Cocom.Archive.TempPath is empty")
	}
	if cfg.Archive.Password == "" {
		t.Error("Config.Archive.Password is empty")
	}
	if cfg.Archive.Cmd == "" {
		t.Error("Config.Archive.Cmd is empty")
	}
	if cfg.Server.Scheduler.Timezone == "" {
		t.Error("Config.Server.Scheduler.Timezone is empty")
	}
	if cfg.Server.RateLimit.RPS <= 0 {
		t.Errorf("Config.Server.RateLimit.RPS = %d, want > 0", cfg.Server.RateLimit.RPS)
	}
	if cfg.Recommend.Limit <= 0 {
		t.Errorf("Config.Recommend.Limit = %d, want > 0", cfg.Recommend.Limit)
	}
}

// TestGetIsIdempotent 验证 Get() 幂等性：同一进程中多次调用返回同一实例。
func TestGetIsIdempotent(t *testing.T) {
	mgr := New()
	cfg1 := mgr.Get()
	cfg2 := mgr.Get()

	if cfg1 != cfg2 {
		t.Error("Get() is not idempotent: cfg1 != cfg2")
	}
}

// TestSetDefaultsIdempotent 验证多次调用 setDefaults 不会改变已设值。
func TestSetDefaultsIdempotent(t *testing.T) {
	mgr := New()
	mgr.Viper().Set("server.ratelimit.rps", 42)

	// 再次调用 SetDefaults 不应覆盖 viper.Set 的值
	mgr.SetDefaults()

	if got := mgr.Viper().GetInt("server.ratelimit.rps"); got != 42 {
		t.Errorf("expected rps=42 after override, got %d", got)
	}
}

// helpers
func notEmpty(v any) bool {
	s, ok := v.(string)
	return ok && s != ""
}

func notEmptySlice(v any) bool {
	ss, ok := v.([]string)
	return ok && len(ss) > 0
}

func isFalse(v any) bool {
	b, ok := v.(bool)
	return ok && !b
}

func positiveInt(v any) bool {
	i, ok := v.(int)
	return ok && i > 0
}

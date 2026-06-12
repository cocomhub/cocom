// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/spf13/viper"
)

// TestInitSetsDefaults 验证 Init() 后所有预期 key 都有非零默认值。
func TestInitSetsDefaults(t *testing.T) {
	// 注意：init() 已在包加载时运行，viper.SetDefault() 幂等
	Init()

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
			got := viper.Get(tt.key)
			if !tt.check(got) {
				t.Errorf("viper.Get(%q) = %v (%T), want to pass check", tt.key, got, got)
			}
		})
	}
}

// TestGetReturnsValidConfig 验证 Get() 返回的 Config 结构体各字段合法。
func TestGetReturnsValidConfig(t *testing.T) {
	Init()
	cfg := Get()

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
	Init()
	cfg1 := Get()
	cfg2 := Get()

	if cfg1 != cfg2 {
		t.Error("Get() is not idempotent: cfg1 != cfg2")
	}
}

// TestBackwardCompatKeys 验证双 key 兼容：cocom.archive.password / archive.password
func TestBackwardCompatKeys(t *testing.T) {
	Init()

	// 情景 A：默认值 — 没有 cocom.archive.password，archive.password 生效
	if got := GetArchivePassword(); got != "archive@123456" {
		t.Errorf("expected default 'archive@123456', got %q", got)
	}

	// 情景 B：cocom.archive.password 设置 → 旧 key 兼容生效（cmp.Or 中排前面）
	viper.Set("cocom.archive.password", "legacy_value")
	if got := GetArchivePassword(); got != "legacy_value" {
		t.Errorf("expected 'legacy_value', got %q", got)
	}

	// 情景 C：两 key 均设置 → cocom.archive.password 优先（cmp.Or 中第一参数）
	viper.Set("archive.password", "new_preferred")
	if got := GetArchivePassword(); got != "legacy_value" {
		t.Errorf("expected 'legacy_value' (cocom.archive.password wins in cmp.Or), got %q", got)
	}
}

// TestSetDefaultsIdempotent 验证多次调用 setDefaults 不会改变已设值。
func TestSetDefaultsIdempotent(t *testing.T) {
	Init()
	viper.Set("server.ratelimit.rps", 42)

	Init() // 再次调用，不应覆盖用户设值

	if got := viper.GetInt("server.ratelimit.rps"); got != 42 {
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

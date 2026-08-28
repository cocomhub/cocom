// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"strings"
	"testing"
)

func TestArchiveString(t *testing.T) {
	tests := []struct {
		name   string
		newVal string
		oldVal string
		want   string
	}{
		{"new wins", "new", "old", "new"},
		{"fallback to old", "", "old", "old"},
		{"both empty", "", "", ""},
		{"same values no warn", "x", "x", "x"},
		{"new empty old empty no fallback", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ArchiveString(tt.newVal, tt.oldVal, "password"); got != tt.want {
				t.Errorf("ArchiveString(%q,%q) = %q, want %q", tt.newVal, tt.oldVal, got, tt.want)
			}
		})
	}
}

func TestArchiveBool(t *testing.T) {
	tests := []struct {
		name   string
		newVal bool
		oldVal bool
		want   bool
	}{
		{"new true", true, false, true},
		{"new false old true fallback", false, true, true},
		{"both false", false, false, false},
		{"new true old true conflict warn", true, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ArchiveBool(tt.newVal, tt.oldVal, "replicate"); got != tt.want {
				t.Errorf("ArchiveBool(%v,%v) = %v, want %v", tt.newVal, tt.oldVal, got, tt.want)
			}
		})
	}
}

func TestArchiveInt(t *testing.T) {
	tests := []struct {
		name   string
		newVal int
		oldVal int
		want   int
	}{
		{"new wins", 4, 8, 4},
		{"fallback to old when new zero", 0, 8, 8},
		{"both zero", 0, 0, 0},
		{"new positive old zero no fallback", 4, 0, 4},
		{"new zero old zero zero", 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ArchiveInt(tt.newVal, tt.oldVal, "algorithm.single.concurrency"); got != tt.want {
				t.Errorf("ArchiveInt(%d,%d) = %d, want %d", tt.newVal, tt.oldVal, got, tt.want)
			}
		})
	}
}

func TestValidate_InvalidIndexType(t *testing.T) {
	mgr := New()
	mgr.Set("archive.manager.index.type", "mongo-comic")
	cfg := mgr.Get()
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() should fail for invalid index.type mongo-comic")
	}
	if !strings.Contains(err.Error(), `"archive.manager.index.type"`) {
		t.Errorf("Validate() error should mention key, got: %v", err)
	}
	if !strings.Contains(err.Error(), "mongo-comicInfo") {
		t.Errorf("Validate() error should list valid types, got: %v", err)
	}
}

func TestValidate_MongoIndexRequiresMongoHost(t *testing.T) {
	mgr := New()
	mgr.Set("archive.manager.index.type", "mongo-cocom")
	mgr.Set("mongo.host", "")
	cfg := mgr.Get()
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() should fail when mongo index type has empty mongo.host")
	}
}

func TestValidate_InvalidShutdownTimeout(t *testing.T) {
	mgr := New()
	mgr.Set("server.shutdown_timeout", "not-a-duration")
	cfg := mgr.Get()
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() should fail for invalid shutdown_timeout")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := New().Get()
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() should pass for default config, got: %v", err)
	}
}

func TestIsMongoIndexType(t *testing.T) {
	for _, typ := range []string{"mongo", "mongo-cocom", "mongo-comicInfo"} {
		if !IsMongoIndexType(typ) {
			t.Errorf("IsMongoIndexType(%q) = false, want true", typ)
		}
	}
	for _, typ := range []string{"", "memory", "file", "bogus"} {
		if IsMongoIndexType(typ) {
			t.Errorf("IsMongoIndexType(%q) = true, want false", typ)
		}
	}
}

// TestArchiveConflict 验证新键优先 + 冲突场景取值（告警是否输出由 warnOnce 消费，测试只断值）。
func TestArchiveConflict(t *testing.T) {
	if got := ArchiveString("new", "old", "password"); got != "new" {
		t.Errorf("ArchiveString conflict = %q, want new", got)
	}
	if got := ArchiveBool(true, true, "replicate"); got != true {
		t.Errorf("ArchiveBool conflict = %v, want true", got)
	}
	if got := ArchiveInt(4, 8, "algorithm.single.concurrency"); got != 4 {
		t.Errorf("ArchiveInt conflict = %d, want 4", got)
	}
}

// TestGetEParseError 验证解析失败经 GetE 返回错误而非 panic（消费点 fail-fast 用的错误路径）。
func TestGetEParseError(t *testing.T) {
	mgr := New()
	mgr.Set("server.shutdown_timeout", []int{1, 2}) // 类型不匹配 → Unmarshal 错误
	mgr.Reset()
	if _, err := mgr.GetE(); err == nil {
		t.Fatal("GetE should return error on unmarshal failure")
	}
	// 验证错误缓存：不 Reset 时重复 GetE 返回同一错误
	if _, err := mgr.GetE(); err == nil {
		t.Fatal("GetE should return cached error on repeat call")
	}
	// 修正类型后 Reset → GetE 应成功（错误不残留）
	mgr.Set("server.shutdown_timeout", "5s")
	mgr.Reset()
	if _, err := mgr.GetE(); err != nil {
		t.Fatalf("GetE after reset should succeed, got: %v", err)
	}
}

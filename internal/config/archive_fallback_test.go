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
	mgr.Viper().Set("archive.manager.index.type", "mongo-comic")
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
	mgr.Viper().Set("archive.manager.index.type", "mongo-cocom")
	mgr.Viper().Set("mongo.host", "")
	cfg := mgr.Get()
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() should fail when mongo index type has empty mongo.host")
	}
}

func TestValidate_InvalidShutdownTimeout(t *testing.T) {
	mgr := New()
	mgr.Viper().Set("server.shutdown_timeout", "not-a-duration")
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

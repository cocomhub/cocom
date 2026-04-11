// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage_test

import (
	"testing"

	"github.com/cocomhub/cocom/pkg/storage"
	_ "github.com/cocomhub/cocom/pkg/storage/localfs"
	"github.com/spf13/viper"
)

func TestSetDuplicateAndEmpty(t *testing.T) {
	v := t.TempDir()
	config := storage.Config{Name: "ext1", Type: "localfs", MetaData: map[string]any{"root": v}}
	if err := storage.SetFromConfig(config); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if err := storage.SetFromConfig(config); err == nil {
		t.Fatalf("duplicate register should error: %v", err)
	}
	if _, ok := storage.Get("ext1"); !ok {
		t.Fatalf("ext1 should remain registered even when config empty")
	}
}

func TestSetFromViper(t *testing.T) {
	v := t.TempDir()
	viper.Set(storage.DefaultBackendsKey, []any{
		storage.Config{Name: "ext1", Type: "localfs", MetaData: map[string]any{"root": v}},
		map[string]any{"name": "ext2", "type": "localfs", "metadata": map[string]any{"root": v}},
	})
	if err := storage.SetFromViper(); err != nil {
		t.Fatalf("SetFromViper: %v", err)
	}
	if _, ok := storage.Get("ext1"); !ok {
		t.Fatalf("ext1 not registered from storage.backends")
	}
	if _, ok := storage.Get("ext2"); !ok {
		t.Fatalf("ext2 not registered from storage.backends")
	}

	viper.Set(storage.DefaultBackendsKey, []any{})
	if err := storage.SetFromViper(); err != nil {
		t.Fatalf("register with empty path should not error: %v", err)
	}
	if _, ok := storage.Get("ext1"); !ok {
		t.Fatalf("ext1 not registered from storage.backends")
	}
	if _, ok := storage.Get("ext2"); !ok {
		t.Fatalf("ext2 not registered from storage.backends")
	}
}

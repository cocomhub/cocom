// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage_test

import (
	"testing"

	"github.com/cocomhub/cocom/pkg/storage"
	_ "github.com/cocomhub/cocom/pkg/storage/localfs"
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

func TestSetFromConfigs(t *testing.T) {
	v := t.TempDir()
	configs := []storage.Config{
		{Name: "ext11", Type: "localfs", MetaData: map[string]any{"root": v}},
		{Name: "ext22", Type: "localfs", MetaData: map[string]any{"root": v}},
	}
	if err := storage.SetFromConfigs(configs); err != nil {
		t.Fatalf("SetFromConfigs: %v", err)
	}
	if _, ok := storage.Get("ext11"); !ok {
		t.Fatalf("ext11 not registered from SetFromConfigs")
	}
	if _, ok := storage.Get("ext22"); !ok {
		t.Fatalf("ext22 not registered from SetFromConfigs")
	}

	if err := storage.SetFromConfigs(nil); err != nil {
		t.Fatalf("SetFromConfigs(nil) should not error: %v", err)
	}
	if _, ok := storage.Get("ext11"); !ok {
		t.Fatalf("ext11 should remain registered after empty SetFromConfigs")
	}
	if _, ok := storage.Get("ext22"); !ok {
		t.Fatalf("ext22 should remain registered after empty SetFromConfigs")
	}
}

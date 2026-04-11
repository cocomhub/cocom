// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package localfs

import (
	"testing"

	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/spf13/viper"
)

func TestSetFromViper(t *testing.T) {
	v := t.TempDir()
	viper.Set("storage.path", v)
	if err := SetFromViper("storage.path"); err != nil {
		t.Fatalf("SetFromViper: %v", err)
	}
	s, ok := storage.Get("storage.path")
	if !ok {
		t.Fatalf("storage.path not set")
	}
	uri, err := storage.URI(s, "a.txt")
	if err != nil {
		t.Fatalf("URI: %v", err)
	}
	if uri != "localfs://storage.path/a.txt" {
		t.Fatalf("unexpected uri: %s", uri)
	}

	viper.Set("storage.path", "")
	if err := SetFromViper("storage.path"); err == nil {
		t.Fatalf("register with empty path should error")
	}

	viper.Set("storage.path", 0)
	if err := SetFromViper("storage.path"); err == nil {
		t.Fatalf("register with not string path should error")
	}

	if _, ok := storage.Get("storage.path"); !ok {
		t.Fatalf("storage.path should remain registered even when config empty")
	}
}

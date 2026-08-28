// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package localfs

import (
	"testing"

	"github.com/cocomhub/cocom/pkg/storage"
)

func TestSetFromMap(t *testing.T) {
	storage.Clear()
	defer storage.Clear()

	v := t.TempDir()
	if err := SetFromMap(map[string]string{"storage.path": v}); err != nil {
		t.Fatalf("SetFromMap: %v", err)
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

	if err := SetFromMap(map[string]string{"storage.path": ""}); err == nil {
		t.Fatal("SetFromMap with empty root should error")
	}
}

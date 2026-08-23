// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import "testing"

func TestConfigDocGen_ExtractPrefix(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"server.listen.http.addr", "server.*"},
		{"cocom.storage.path", "cocom.*"},
		{"single", "single.*"},
		{"", ".*"},
	}
	for _, tt := range tests {
		if got := extractPrefix(tt.in); got != tt.want {
			t.Errorf("extractPrefix(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestConfigDocGen_GetOrCreateEntry(t *testing.T) {
	before := len(entries)

	e := getOrCreateEntry("test.doc.key")
	if e == nil {
		t.Fatal("getOrCreateEntry returned nil")
	}
	if e.Key != "test.doc.key" {
		t.Errorf("entry key = %q, want test.doc.key", e.Key)
	}
	if len(entries) != before+1 {
		t.Errorf("entries grew to %d, want %d", len(entries), before+1)
	}

	// 幂等：同一 key 返回同一指针，不重复创建
	e2 := getOrCreateEntry("test.doc.key")
	if e2 != e {
		t.Error("getOrCreateEntry should return the same pointer for the same key")
	}
}

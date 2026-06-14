// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package genwget

import (
	"testing"
)

func TestGenwget_NewManager(t *testing.T) {
	mgr := NewManager()
	if mgr == nil {
		t.Fatal("NewManager should return non-nil")
	}
	t.Logf("Manager created: %T", mgr)
}

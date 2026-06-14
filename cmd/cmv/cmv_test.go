// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmv

import (
	"testing"
)

func TestCmv_NewManager(t *testing.T) {
	mgr := NewComicMoveManager()
	if mgr == nil {
		t.Fatal("NewComicMoveManager should return non-nil")
	}
	t.Logf("ComicMoveManager created: %T", mgr)
}

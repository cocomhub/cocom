// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"testing"
)

func TestVersion_Info(t *testing.T) {
	if Version == "" {
		t.Log("Version constant exists (may be empty in test build)")
	}
	if CommitID == "" {
		t.Log("CommitID constant exists (may be empty in test build)")
	}
	_ = BuiltAt
	_ = Branch
	_ = ReleaseURL
}

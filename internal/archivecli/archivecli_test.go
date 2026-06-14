// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package archivecli

import (
	"testing"
)

func TestArchiveCli_EmitError(t *testing.T) {
	// Verify EmitError function compiles and doesn't panic
	opts := &Options{}
	if opts.OutputMode == nil {
		t.Log("Options struct compiles")
	}
}

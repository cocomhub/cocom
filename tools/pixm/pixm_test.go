// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import "testing"

func TestPixm_OutputMode(t *testing.T) {
	flagOutput = ""
	t.Cleanup(func() { flagOutput = "" })

	if got := outputMode(); got != "text" {
		t.Errorf("empty flag output mode = %q, want text", got)
	}
	flagOutput = "json"
	if got := outputMode(); got != "json" {
		t.Errorf("json flag output mode = %q, want json", got)
	}
	flagOutput = "  json  "
	if got := outputMode(); got != "  json  " {
		t.Errorf("trimmed flag output mode = %q, want %q", got, "  json  ")
	}
}

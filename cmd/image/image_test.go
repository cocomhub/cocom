// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"testing"
)

func TestImage_CommandDefined(t *testing.T) {
	if Cmd.Use == "" {
		t.Error("Cmd.Use should not be empty")
	} else {
		t.Logf("image command: %s - %s", Cmd.Use, Cmd.Short)
	}
}

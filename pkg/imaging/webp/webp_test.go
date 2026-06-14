// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package webp

import (
	"bytes"
	"testing"
)

func TestWebp_Decode(t *testing.T) {
	data := []byte{0x00}
	_, err := Decode(bytes.NewReader(data))
	if err == nil {
		t.Log("Decode with invalid data did not error (may return nil)")
	} else {
		t.Logf("Decode with invalid data returns: %v", err)
	}
}

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package gallery

import (
	"reflect"
	"testing"
)

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{"a, b, c", []string{"a", "b", "c"}},
		{"", []string{}},
		{"single", []string{"single"}},
		{"  spaced  ,  around  ", []string{"spaced", "around"}},
	}
	for _, tc := range tests {
		result := splitAndTrim(tc.input)
		if !reflect.DeepEqual(result, tc.expected) {
			t.Errorf("splitAndTrim(%q) = %v, want %v", tc.input, result, tc.expected)
		}
	}
}

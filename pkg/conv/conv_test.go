// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package conv

import "testing"

func TestConv_JSON(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"struct", struct {
			A string `json:"a"`
		}{"x"}, `{"a":"x"}`},
		{"map", map[string]any{"k": 1}, `{"k":1}`},
		{"slice", []int{1, 2, 3}, `[1,2,3]`},
		{"nil", nil, "null"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JSON(tt.in); got != tt.want {
				t.Errorf("JSON(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

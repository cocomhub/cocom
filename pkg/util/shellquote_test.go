// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package util

import "testing"

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty string", "", "''"},
		{"plain ascii", "abc", "'abc'"},
		{"with space", "a b", "'a b'"},
		{"single quote", "x'y", `'x'\''y'`},
		{"leading/trailing quote", "'abc'", `''\''abc'\'''`},
		{"dollar sign", "$HOME", "'$HOME'"},
		{"backtick", "`id`", "'`id`'"},
		{"semicolon", "a;rm -rf /", "'a;rm -rf /'"},
		{"backslash", `a\b`, `'a\b'`},
		{"newline", "a\nb", "'a\nb'"},
		{"mixed injection", `[123] x'; rm -rf ~; echo '`, `'[123] x'\''; rm -rf ~; echo '\'''`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShellQuote(tt.in); got != tt.want {
				t.Errorf("ShellQuote(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

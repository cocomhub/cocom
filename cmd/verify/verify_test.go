// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package verify

import (
	"testing"
)

func TestVerifyCommand_Registration(t *testing.T) {
	if Cmd == nil {
		t.Fatal("Cmd should not be nil")
	}
	if Cmd.Use != "verify" {
		t.Errorf("expected Use 'verify', got %s", Cmd.Use)
	}
}

func TestVerifyCommand_HasSubcommands(t *testing.T) {
	subCommands := Cmd.Commands()
	names := make(map[string]bool)
	for _, cmd := range subCommands {
		names[cmd.Use] = true
	}
	expected := []string{"status", "cancel", "schedule"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected subcommand %s not found", name)
		}
	}
}

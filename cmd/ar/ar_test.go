// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ar

import (
	"testing"
)

func TestArCommand_Registration(t *testing.T) {
	if Cmd == nil {
		t.Fatal("Cmd should not be nil")
	}
	if Cmd.Use != "ar" {
		t.Errorf("expected Use 'ar', got %s", Cmd.Use)
	}
}

func TestArCommand_HasPersistentFlags(t *testing.T) {
	f := Cmd.PersistentFlags().Lookup("cid")
	if f == nil {
		t.Fatal("expected --cid flag")
	}
	f = Cmd.PersistentFlags().Lookup("output")
	if f == nil {
		t.Fatal("expected --output flag")
	}
}

func TestArCommand_OutputModeDefault(t *testing.T) {
	if arOutput != "text" {
		t.Errorf("expected default output 'text', got %s", arOutput)
	}
}

func TestArCommand_HasSubcommands(t *testing.T) {
	cmds := Cmd.Commands()
	if len(cmds) == 0 {
		t.Fatal("expected at least one subcommand")
	}
	found := make(map[string]bool)
	for _, c := range cmds {
		found[c.Name()] = true
	}
	expected := []string{"pack", "unpack", "query", "backup", "check"}
	for _, name := range expected {
		if !found[name] {
			t.Errorf("expected subcommand %q not found", name)
		}
	}
}

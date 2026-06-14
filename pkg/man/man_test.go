// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package man

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestMan_AddManCmd(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	AddManCmd(rootCmd)

	// Verify the man subcommand was added
	manCmd, _, err := rootCmd.Find([]string{"man"})
	if err != nil {
		t.Fatalf("expected 'man' subcommand to be registered: %v", err)
	}
	if manCmd.Use != "man" {
		t.Errorf("expected cmd.Use='man', got %q", manCmd.Use)
	}
	if !manCmd.Hidden {
		t.Error("expected man command to be hidden")
	}
}

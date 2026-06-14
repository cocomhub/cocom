// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"testing"
)

func TestCmd_RootCommandDefined(t *testing.T) {
	if rootCmd.Use == "" {
		t.Error("rootCmd.Use should not be empty")
	} else {
		t.Logf("root command: %s", rootCmd.Use)
	}
}

func TestCmd_SubCommands(t *testing.T) {
	cmds := rootCmd.Commands()
	t.Logf("root command has %d sub-commands", len(cmds))
	if len(cmds) > 0 {
		names := make([]string, 0, len(cmds))
		for _, c := range cmds {
			names = append(names, c.Name())
		}
		t.Logf("sub-commands: %v", names)
	}
}

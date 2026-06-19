// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package rootcli

import (
	"github.com/cocomhub/cocom/pkg/logging"
	"github.com/spf13/cobra"
)

func TestRootcli_InitRootCmd(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	InitRootCmd(rootCmd)

	configFlag := rootCmd.PersistentFlags().Lookup("config")
	if configFlag == nil {
		t.Error("expected --config flag to be registered")
	} else {
		t.Logf("config flag registered: %s", configFlag.Usage)
	}
}

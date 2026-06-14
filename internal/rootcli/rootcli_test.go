// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package rootcli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestRootcli_InitRootCmd(t *testing.T) {
	defer viper.Reset()

	rootCmd := &cobra.Command{Use: "test"}
	InitRootCmd(rootCmd)

	configFlag := rootCmd.PersistentFlags().Lookup("config")
	if configFlag == nil {
		t.Error("expected --config flag to be registered")
	} else {
		t.Logf("config flag registered: %s", configFlag.Usage)
	}
}

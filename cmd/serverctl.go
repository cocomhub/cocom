// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
)

// serverctlCmd represents the serverctl command
var serverctlCmd = &cobra.Command{
	Use:   "serverctl",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		slog.InfoContext(cmd.Context(), "serverctl called")
	},
}

func init() {
	rootCmd.AddCommand(serverctlCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serverctlCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serverctlCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

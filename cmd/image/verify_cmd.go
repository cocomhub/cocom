// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"github.com/cocomhub/cocom/pkg/imaging"
	"github.com/cocomhub/cocom/pkg/logging"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify <src...> [flags]",
	Short: "验证图片完整性",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := logging.NewTraceCtx("verify")

		opts := &imaging.BatchOptions{
			Workers:    imageFlag.workers,
			Op:         "verify",
			ResultFile: imageFlag.resultFile,
		}

		return imaging.ProcessBatch(ctx, args, opts, func(h *imaging.ImageHandler) error {
			return h.Verify()
		})
	},
}

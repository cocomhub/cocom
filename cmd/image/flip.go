// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"github.com/cocomhub/cocom/pkg/imaging"
	"github.com/cocomhub/cocom/pkg/logging"
	"github.com/spf13/cobra"
)

var flipCmd = &cobra.Command{
	Use:   "flip <src...> <dst>",
	Short: "垂直翻转图片",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := logging.NewTraceCtx("flip")

		dst := args[len(args)-1]
		srcs := args[:len(args)-1]

		params := map[string]string{
			"format": imageFlag.format,
		}

		return processImage(ctx, srcs, dst, "flip", params,
			func(h *imaging.ImageHandler) error {
				return h.Flip()
			})
	},
}

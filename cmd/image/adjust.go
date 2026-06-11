// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"strconv"

	"github.com/cocomhub/cocom/pkg/errwrap"
	"github.com/cocomhub/cocom/pkg/imaging"
	"github.com/cocomhub/cocom/pkg/logging"
	"github.com/spf13/cobra"
)

var adjustCmd = &cobra.Command{
	Use:   "adjust <src...> <dst> <brightness> <contrast>",
	Short: "调整图片亮度和对比度",
	Args:  cobra.MinimumNArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := logging.NewTraceCtx("adjust")

		brightness, err := strconv.ParseFloat(args[len(args)-2], 64)
		if err != nil {
			return errwrap.ErrInvalidArgs.SetIErrF("无效的亮度值: %v", err)
		}
		contrast, err := strconv.ParseFloat(args[len(args)-1], 64)
		if err != nil {
			return errwrap.ErrInvalidArgs.SetIErrF("无效的对比度值: %v", err)
		}

		dst := args[len(args)-3]
		srcs := args[:len(args)-3]

		params := map[string]string{
			"format":     imageFlag.format,
			"brightness": strconv.FormatFloat(brightness, 'f', -1, 64),
			"contrast":   strconv.FormatFloat(contrast, 'f', -1, 64),
		}

		return processImage(ctx, srcs, dst, "adjust", params,
			func(h *imaging.ImageHandler) error {
				return h.Adjust(brightness, contrast)
			})
	},
}

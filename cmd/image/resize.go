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

var resizeCmd = &cobra.Command{
	Use:   "resize <src...> <dst> <width> <height>",
	Short: "调整图片大小",
	Args:  cobra.MinimumNArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := logging.NewTraceCtx("resize")

		width, err := strconv.Atoi(args[len(args)-2])
		if err != nil {
			return errwrap.ErrInvalidArgs.SetIErrF("无效的宽度值: %v", err)
		}
		height, err := strconv.Atoi(args[len(args)-1])
		if err != nil {
			return errwrap.ErrInvalidArgs.SetIErrF("无效的高度值: %v", err)
		}

		dst := args[len(args)-3]
		srcs := args[:len(args)-3]

		params := map[string]string{
			"format": imageFlag.format,
			"w":      strconv.Itoa(width),
			"h":      strconv.Itoa(height),
		}

		return processImage(ctx, srcs, dst, "resize", params,
			func(h *imaging.ImageHandler) error {
				return h.Resize(width, height)
			})
	},
}

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

var cropCmd = &cobra.Command{
	Use:   "crop <src...> <dst> <x> <y> <width> <height>",
	Short: "裁剪图片",
	Args:  cobra.MinimumNArgs(6),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := logging.NewTraceCtx("crop")

		x, err := strconv.Atoi(args[len(args)-4])
		if err != nil {
			return errwrap.ErrInvalidArgs.SetIErrF("无效的 x 坐标: %v", err)
		}
		y, err := strconv.Atoi(args[len(args)-3])
		if err != nil {
			return errwrap.ErrInvalidArgs.SetIErrF("无效的 y 坐标: %v", err)
		}
		width, err := strconv.Atoi(args[len(args)-2])
		if err != nil {
			return errwrap.ErrInvalidArgs.SetIErrF("无效的宽度值: %v", err)
		}
		height, err := strconv.Atoi(args[len(args)-1])
		if err != nil {
			return errwrap.ErrInvalidArgs.SetIErrF("无效的高度值: %v", err)
		}

		dst := args[len(args)-5]
		srcs := args[:len(args)-5]

		params := map[string]string{
			"format": imageFlag.format,
			"x":      strconv.Itoa(x),
			"y":      strconv.Itoa(y),
			"w":      strconv.Itoa(width),
			"h":      strconv.Itoa(height),
		}

		return processImage(ctx, srcs, dst, "crop", params,
			func(h *imaging.ImageHandler) error {
				return h.Crop(x, y, width, height)
			})
	},
}

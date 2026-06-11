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

var rotateCmd = &cobra.Command{
	Use:   "rotate <src...> <dst> <angle>",
	Short: "旋转图片",
	Args:  cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := logging.NewTraceCtx("rotate")

		angle, err := strconv.ParseFloat(args[len(args)-1], 64)
		if err != nil {
			return errwrap.ErrInvalidArgs.SetIErrF("无效的角度值: %v", err)
		}

		dst := args[len(args)-2]
		srcs := args[:len(args)-2]

		params := map[string]string{
			"format": imageFlag.format,
			"angle":  strconv.FormatFloat(angle, 'f', -1, 64),
		}

		return processImage(ctx, srcs, dst, "rotate", params,
			func(h *imaging.ImageHandler) error {
				return h.Rotate(angle)
			})
	},
}

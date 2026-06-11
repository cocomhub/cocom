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

var sharpenCmd = &cobra.Command{
	Use:   "sharpen <src...> <dst> <sigma>",
	Short: "锐化处理图片",
	Args:  cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := logging.NewTraceCtx("sharpen")

		sigma, err := strconv.ParseFloat(args[len(args)-1], 64)
		if err != nil {
			return errwrap.ErrInvalidArgs.SetIErrF("无效的 sigma 值: %v", err)
		}

		dst := args[len(args)-2]
		srcs := args[:len(args)-2]

		params := map[string]string{
			"format": imageFlag.format,
			"sigma":  strconv.FormatFloat(sigma, 'f', -1, 64),
		}

		return processImage(ctx, srcs, dst, "sharpen", params,
			func(h *imaging.ImageHandler) error {
				return h.Sharpen(sigma)
			})
	},
}

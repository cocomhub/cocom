// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"strings"

	"github.com/cocomhub/cocom/pkg/errwrap"
	"github.com/cocomhub/cocom/pkg/imaging"
	"github.com/cocomhub/cocom/pkg/imaging/webp"
	"github.com/cocomhub/cocom/pkg/logging"
	"github.com/spf13/cobra"
)

var convertCmd = &cobra.Command{
	Use:   "convert <src...> <dst> <format>",
	Short: "转换图片格式",
	Args:  cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := logging.NewTraceCtx("convert")

		format := args[len(args)-1]

		if strings.ToLower(format) == "webp" && !webp.HasWebPUtil() {
			return errwrap.ErrImageFormat.SetIErrF("未安装 WebP 工具，请运行 'cocom install webp' 安装")
		}

		dst := args[len(args)-2]
		srcs := args[:len(args)-2]

		params := map[string]string{
			"format": format,
		}

		return processImage(ctx, srcs, dst, "convert", params,
			func(h *imaging.ImageHandler) error {
				return h.ConvertFormat(format)
			})
	},
}

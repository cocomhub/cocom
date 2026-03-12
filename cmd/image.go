// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/cocomhub/cocom/pkg/errwrap"
	"github.com/cocomhub/cocom/pkg/imaging"
	"github.com/cocomhub/cocom/pkg/imaging/webp"
	"github.com/cocomhub/cocom/pkg/logging"
	"github.com/spf13/cobra"
)

type imageFlags struct {
	format     string
	workers    int
	batch      bool
	resultFile string
}

var imageFlag imageFlags

var imageCmd = &cobra.Command{
	Use:   "image <command> [flags]",
	Short: "图片处理工具",
	Long: `提供图片处理功能，包括调整大小、裁剪、旋转等。

使用示例：
  # 调整单个图片大小（输出到文件）
  cocom image resize input.jpg output.jpg 800 600

  # 调整单个图片大小（输出到目录）
  cocom image resize input.jpg ./output/ 800 600
  # 将生成: ./output/input_resize_w800h600.jpg

  # 裁剪图片（指定坐标和尺寸）
  cocom image crop input.jpg ./output/ 100 100 400 300
  # 将生成: ./output/input_crop_x100y100w400h300.jpg

  # 旋转图片（指定角度）
  cocom image rotate input.jpg ./output/ 90
  # 将生成: ./output/input_rotate_angle90.jpg

  # 调整亮度和对比度
  cocom image adjust input.jpg ./output/ 0.5 1.5
  # 将生成: ./output/input_adjust_brightness0.5contrast1.5.jpg

  # 模糊处理（指定 sigma 值）
  cocom image blur input.jpg ./output/ 0.5
  # 将生成: ./output/input_blur_sigma0.5.jpg

  # 锐化处理（指定 sigma 值）
  cocom image sharpen input.jpg ./output/ 0.5
  # 将生成: ./output/input_sharpen_sigma0.5.jpg

  # 垂直翻转
  cocom image flip input.jpg ./output/
  # 将生成: ./output/input_flip.jpg

  # 水平翻转
  cocom image flop input.jpg ./output/
  # 将生成: ./output/input_flop.jpg

  # 格式转换
  cocom image convert input.jpg ./output/ png
  # 将生成: ./output/input_convert.png

  # 验证单个图片完整性
  cocom image verify input.jpg

  # 批量调整图片大小（指定源目录）
  cocom image resize "./src/*.jpg" ./output/ 800 600 --batch

  # 批量处理多个源
  cocom image resize "./photos/*.jpg" "./images/*.png" output/ 800 600 --batch

  # 批量格式转换（转换为 WebP）
  cocom image convert "./src/*.*" ./output/ webp --batch

  # 批量验证图片（保存结果）
  cocom image verify "./src/*.jpg" --batch -o results.json

支持的图片格式：
  - JPEG (.jpg, .jpeg)
  - PNG (.png)
  - GIF (.gif)
  - BMP (.bmp)
  - TIFF (.tif, .tiff)
  - WebP (.webp)

输出文件命名规则：
  - resize: {name}_resize_w{width}h{height}.{ext}
  - crop:   {name}_crop_x{x}y{y}w{width}h{height}.{ext}
  - rotate: {name}_rotate_angle{angle}.{ext}
  - adjust: {name}_adjust_brightness{b}contrast{c}.{ext}
  - blur:   {name}_blur_sigma{sigma}.{ext}
  - sharpen:{name}_sharpen_sigma{sigma}.{ext}
  - flip:   {name}_flip.{ext}
  - flop:   {name}_flop.{ext}
  - convert:{name}_convert.{new_ext}

全局参数：
  -f, --format string   输出格式(jpg,png,gif,tiff,bmp,webp)
  -b, --batch          批量处理模式
  -n, --workers int    并发工作协程数 (默认: 4)
  -o, --output string  结果输出文件路径`,
}

var resizeCmd = &cobra.Command{
	Use:   "resize <src...> <dst> <width> <height>",
	Short: "调整图片大小",
	Args:  cobra.MinimumNArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := logging.NewTraceCtx("resize")

		// 最后两个参数是宽高
		width, err := strconv.Atoi(args[len(args)-2])
		if err != nil {
			return errwrap.ErrInvalidArgs.SetIErrF("无效的宽度值: %v", err)
		}
		height, err := strconv.Atoi(args[len(args)-1])
		if err != nil {
			return errwrap.ErrInvalidArgs.SetIErrF("无效的高度值: %v", err)
		}

		// 倒数第三个参数是目标路径
		dst := args[len(args)-3]
		// 其余都是源路径
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

var cropCmd = &cobra.Command{
	Use:   "crop <src...> <dst> <x> <y> <width> <height>",
	Short: "裁剪图片",
	Args:  cobra.MinimumNArgs(6),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := logging.NewTraceCtx("crop")

		// 解析位置参数
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

		// 倒数第五个参数是目标路径
		dst := args[len(args)-5]
		// 其余都是源路径
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

var rotateCmd = &cobra.Command{
	Use:   "rotate <src...> <dst> <angle>",
	Short: "旋转图片",
	Args:  cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := logging.NewTraceCtx("rotate")

		// 最后一个参数是角度
		angle, err := strconv.ParseFloat(args[len(args)-1], 64)
		if err != nil {
			return errwrap.ErrInvalidArgs.SetIErrF("无效的角度值: %v", err)
		}

		// 倒数第二个参数是目标路径
		dst := args[len(args)-2]
		// 其余都是源路径
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

var adjustCmd = &cobra.Command{
	Use:   "adjust <src...> <dst> <brightness> <contrast>",
	Short: "调整图片亮度和对比度",
	Args:  cobra.MinimumNArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := logging.NewTraceCtx("adjust")

		// 最后两个参数是亮度和对比度
		brightness, err := strconv.ParseFloat(args[len(args)-2], 64)
		if err != nil {
			return errwrap.ErrInvalidArgs.SetIErrF("无效的亮度值: %v", err)
		}
		contrast, err := strconv.ParseFloat(args[len(args)-1], 64)
		if err != nil {
			return errwrap.ErrInvalidArgs.SetIErrF("无效的对比度值: %v", err)
		}

		// 倒数第三个参数是目标路径
		dst := args[len(args)-3]
		// 其余都是源路径
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

var blurCmd = &cobra.Command{
	Use:   "blur <src...> <dst> <sigma>",
	Short: "模糊处理图片",
	Args:  cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := logging.NewTraceCtx("blur")

		// 最后一个参数是 sigma
		sigma, err := strconv.ParseFloat(args[len(args)-1], 64)
		if err != nil {
			return errwrap.ErrInvalidArgs.SetIErrF("无效的 sigma 值: %v", err)
		}

		// 倒数第二个参数是目标路径
		dst := args[len(args)-2]
		// 其余都是源路径
		srcs := args[:len(args)-2]

		params := map[string]string{
			"format": imageFlag.format,
			"sigma":  strconv.FormatFloat(sigma, 'f', -1, 64),
		}

		return processImage(ctx, srcs, dst, "blur", params,
			func(h *imaging.ImageHandler) error {
				return h.Blur(sigma)
			})
	},
}

var sharpenCmd = &cobra.Command{
	Use:   "sharpen <src...> <dst> <sigma>",
	Short: "锐化处理图片",
	Args:  cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := logging.NewTraceCtx("sharpen")

		// 最后一个参数是 sigma
		sigma, err := strconv.ParseFloat(args[len(args)-1], 64)
		if err != nil {
			return errwrap.ErrInvalidArgs.SetIErrF("无效的 sigma 值: %v", err)
		}

		// 倒数第二个参数是目标路径
		dst := args[len(args)-2]
		// 其余都是源路径
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

var flipCmd = &cobra.Command{
	Use:   "flip <src...> <dst>",
	Short: "垂直翻转图片",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := logging.NewTraceCtx("flip")

		// 最后一个参数是目标路径
		dst := args[len(args)-1]
		// 其余都是源路径
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

var flopCmd = &cobra.Command{
	Use:   "flop <src...> <dst>",
	Short: "水平翻转图片",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := logging.NewTraceCtx("flop")

		// 最后一个参数是目标路径
		dst := args[len(args)-1]
		// 其余都是源路径
		srcs := args[:len(args)-1]

		params := map[string]string{
			"format": imageFlag.format,
		}

		return processImage(ctx, srcs, dst, "flop", params,
			func(h *imaging.ImageHandler) error {
				return h.Flop()
			})
	},
}

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

var convertCmd = &cobra.Command{
	Use:   "convert <src...> <dst> <format>",
	Short: "转换图片格式",
	Args:  cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := logging.NewTraceCtx("convert")

		// 最后一个参数是目标格式
		format := args[len(args)-1]

		// 如果是 WebP 格式，检查工具是否安装
		if strings.ToLower(format) == "webp" && !webp.HasWebPUtil() {
			return errwrap.ErrImageFormat.SetIErrF("未安装 WebP 工具，请运行 'cocom install webp' 安装")
		}

		// 倒数第二个参数是目标路径
		dst := args[len(args)-2]
		// 其余都是源路径
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

func init() {
	rootCmd.AddCommand(imageCmd)
	imageCmd.AddCommand(resizeCmd)
	imageCmd.AddCommand(cropCmd)
	imageCmd.AddCommand(rotateCmd)
	imageCmd.AddCommand(adjustCmd)
	imageCmd.AddCommand(blurCmd)
	imageCmd.AddCommand(sharpenCmd)
	imageCmd.AddCommand(flipCmd)
	imageCmd.AddCommand(flopCmd)
	imageCmd.AddCommand(verifyCmd)
	imageCmd.AddCommand(convertCmd)

	// 全局标志
	imageCmd.PersistentFlags().StringVarP(&imageFlag.format, "format", "f", "", "输出格式(jpg,png,gif,tiff,bmp,webp)")
	imageCmd.PersistentFlags().IntVarP(&imageFlag.workers, "workers", "n", 4, "并发工作协程数")
	imageCmd.PersistentFlags().BoolVarP(&imageFlag.batch, "batch", "b", false, "批量处理模式")
	imageCmd.PersistentFlags().StringVarP(&imageFlag.resultFile, "output", "o", "", "结果输出文件路径")
}

func processImage(ctx context.Context, srcs []string, dst string, op string, params map[string]string, processor func(*imaging.ImageHandler) error) error {
	if imageFlag.batch {
		opts := &imaging.BatchOptions{
			DstDir:     dst,
			Workers:    imageFlag.workers,
			Params:     params,
			Op:         op,
			ResultFile: imageFlag.resultFile,
		}
		batch := imaging.NewBatchProcessor(ctx, opts)
		if err := batch.AddFiles(srcs...); err != nil {
			return err
		}

		// 使用单独的 channel 控制进度显示
		done := make(chan struct{})
		defer close(done)

		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					progress := batch.GetProgress()
					fmt.Printf("\r处理进度: %.2f%% (%d/%d), 失败: %d",
						progress.Progress, progress.Completed, progress.Total, progress.Failed)
				}
			}
		}()

		err := batch.Process(processor)
		fmt.Println() // 换行
		return err
	}

	// 单文件处理
	if len(srcs) != 1 {
		return errwrap.ErrInvalidArgs.SetIErrF("非批量模式只能处理单个文件")
	}

	imgInfo, err := imaging.ProcessImage(ctx, srcs[0], dst, op, params, processor)
	if err != nil {
		return err
	}

	if imageFlag.resultFile != "" {
		result := &imaging.BatchResult{
			Path:      imgInfo.Path,
			Image:     imgInfo,
			Timestamp: time.Now(),
		}
		if err := imaging.SaveResults(imageFlag.resultFile, result); err != nil {
			slog.ErrorContext(ctx, "保存结果失败", slog.String("errmsg", err.Error()), slog.Any("result", result))
		}
	}

	return nil
}

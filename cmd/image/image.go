// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/cocomhub/cocom/pkg/imaging"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "image <command> [flags]",
	Short: "图片处理工具",
}

type imageFlags struct {
	format     string
	workers    int
	batch      bool
	resultFile string
}

var imageFlag imageFlags

func init() {
	Cmd.AddCommand(resizeCmd, cropCmd, rotateCmd, adjustCmd, blurCmd, sharpenCmd, flipCmd, flopCmd, convertCmd, verifyCmd)
	Cmd.PersistentFlags().StringVarP(&imageFlag.format, "format", "f", "", "输出格式(jpg,png,gif,tiff,bmp,webp)")
	Cmd.PersistentFlags().IntVarP(&imageFlag.workers, "workers", "n", 4, "并发工作协程数")
	Cmd.PersistentFlags().BoolVarP(&imageFlag.batch, "batch", "b", false, "批量处理模式")
	Cmd.PersistentFlags().StringVarP(&imageFlag.resultFile, "output", "o", "", "结果输出文件路径")
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
					fmt.Printf("处理进度: %.2f%% (%d/%d), 失败: %d",
						progress.Progress, progress.Completed, progress.Total, progress.Failed)
				}
			}
		}()

		err := batch.Process(processor)
		fmt.Println()
		return err
	}

	if len(srcs) != 1 {
		return fmt.Errorf("非批量模式只能处理单个文件")
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

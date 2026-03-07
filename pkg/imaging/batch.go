// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package imaging

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cocomhub/cocom/pkg/clog"
	"github.com/cocomhub/cocom/pkg/errwrap"
	"github.com/cocomhub/cocom/pkg/imaging/webp"
)

// BatchOptions 批处理选项
type BatchOptions struct {
	DstDir     string            // 输出目录
	Workers    int               // 并发数
	Params     map[string]string // 处理参数
	Op         string            // 操作类型
	ResultFile string            // 结果输出文件路径
}

type BatchResult struct {
	Path      string     `json:"path"`
	Error     string     `json:"error,omitempty"`
	Image     *ImageInfo `json:"image"`
	Timestamp time.Time  `json:"timestamp"`
}

// BatchProcessor 批量处理器
type BatchProcessor struct {
	ctx     context.Context
	files   []string
	opts    *BatchOptions
	results []*BatchResult // 处理结果
	errs    []string       // 错误信息
	mu      sync.RWMutex   // 保护 results 和 errs
	wg      sync.WaitGroup
}

// NewBatchProcessor 创建批量处理器
func NewBatchProcessor(ctx context.Context, opts *BatchOptions) *BatchProcessor {
	return &BatchProcessor{
		ctx:  ctx,
		opts: opts,
	}
}

// AddFiles 添加待处理文件
func (b *BatchProcessor) AddFiles(patterns ...string) error {
	for _, pattern := range patterns {
		files, err := filepath.Glob(pattern)
		if err != nil {
			return errwrap.ErrInvalidArgs.SetIErrF("无效的源文件模式: %v", err)
		}
		b.files = append(b.files, files...)
	}
	if len(b.files) == 0 {
		return errwrap.ErrImageEmpty.SetIErrF("未找到匹配的源文件")
	}
	return nil
}

// Process 执行批量处理
func (b *BatchProcessor) Process(processor func(*ImageHandler) error) error {
	if len(b.files) == 0 {
		return errwrap.ErrImageEmpty.SetIErrF("源文件列表为空")
	}

	// 如果需要目标目录，确保它存在
	if b.opts.DstDir != "" {
		if err := os.MkdirAll(b.opts.DstDir, 0o755); err != nil {
			return errwrap.ErrImageDir.SetIErrF("创建目标目录失败: %v", err)
		}
	}

	// 如果是转换为 WebP 格式，先检查工具是否安装
	if format := b.opts.Params["format"]; format == "webp" && !webp.HasWebPUtil() {
		return errwrap.ErrImageFormat.SetIErrF("未安装 WebP 工具，请运行 'cocom install webp' 安装")
	}

	jobs := make(chan string, len(b.files))
	results := make(chan *BatchResult, len(b.files))
	errors := make(chan string, len(b.files))

	// 添加进度日志
	go func() {
		for {
			select {
			case <-b.ctx.Done():
				return
			case <-time.After(time.Second):
				progress := b.GetProgress()
				clog.Infof(b.ctx, "处理进度: %.2f%% (%d/%d), 失败: %d",
					progress.Progress, progress.Completed, progress.Total, progress.Failed)
			}
		}
	}()

	// 启动工作协程
	for i := 0; i < b.opts.Workers; i++ {
		b.wg.Add(1)
		go func() {
			defer b.wg.Done()
			for src := range jobs {
				dst := ""
				if b.opts.DstDir != "" {
					dst = GenerateOutputPath(src, b.opts.DstDir, b.opts.Op, b.opts.Params)
				}

				result := &BatchResult{
					Path:      src,
					Timestamp: time.Now(),
				}

				handler, err := NewImageHandler(b.ctx, src, dst)
				if err != nil {
					result.Error = err.Error()
					results <- result
					errors <- fmt.Sprintf("处理文件 %s 失败: %v", src, err)
					continue
				}

				err = processor(handler)
				if err != nil {
					result.Error = err.Error()
					results <- result
					errors <- fmt.Sprintf("处理文件 %s 失败: %v", src, err)
					continue
				}
				result.Image = handler.GetInfo()

				if handler.DstPath != "" {
					if err := handler.Save(handler.DstPath); err != nil {
						result.Error = err.Error()
						results <- result
						errors <- fmt.Sprintf("保存文件 %s 失败: %v", src, err)
						continue
					}
					clog.Debugf(b.ctx, "成功处理文件: %s -> %s", src, dst)
				}

				results <- result
			}
		}()
	}

	// 发送任务
	for _, file := range b.files {
		jobs <- file
	}
	close(jobs)

	var resultWg sync.WaitGroup
	resultWg.Add(2)
	go func() {
		defer resultWg.Done()
		for result := range results {
			b.mu.Lock()
			b.results = append(b.results, result)
			b.mu.Unlock()
		}
	}()

	go func() {
		defer resultWg.Done()
		for err := range errors {
			b.mu.Lock()
			b.errs = append(b.errs, err)
			b.mu.Unlock()
		}
	}()

	// 等待所有任务完成
	b.wg.Wait()
	close(results)
	close(errors)
	resultWg.Wait()

	// 保存处理结果
	if b.opts.ResultFile != "" {
		if err := SaveResults(b.opts.ResultFile, b.results...); err != nil {
			clog.Errorf(b.ctx, "保存批量处理结果失败: %v，results: %v", err, b.results)
		}
	}

	if len(b.errs) > 0 {
		return errwrap.ErrImageBatch.SetIErrF("批量处理出现错误:\n%s", strings.Join(b.errs, "\n"))
	}

	return nil
}

// Results 获取处理结果
func (b *BatchProcessor) Results() []*BatchResult {
	return b.results
}

// Errors 获取错误信息
func (b *BatchProcessor) Errors() []string {
	return b.errs
}

// ProcessBatch 通用批处理函数
func ProcessBatch(ctx context.Context, patterns []string, opts *BatchOptions, processor func(*ImageHandler) error) error {
	batch := NewBatchProcessor(ctx, opts)
	if err := batch.AddFiles(patterns...); err != nil {
		return err
	}
	return batch.Process(processor)
}

// SaveResults 保存结果到文件
func SaveResults(path string, results ...*BatchResult) error {
	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	// 保存为 JSON 格式
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

// BatchProgress 批处理进度
type BatchProgress struct {
	Total     int     `json:"total"`
	Completed int     `json:"completed"`
	Failed    int     `json:"failed"`
	Progress  float64 `json:"progress"`
}

// GetProgress 获取处理进度
func (b *BatchProcessor) GetProgress() *BatchProgress {
	b.mu.RLock()
	defer b.mu.RUnlock()

	total := len(b.files)
	completed := len(b.results)
	failed := len(b.errs)
	progress := float64(completed) / float64(total) * 100

	return &BatchProgress{
		Total:     total,
		Completed: completed,
		Failed:    failed,
		Progress:  progress,
	}
}

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMetricsCollector_Basic(t *testing.T) {
	collector := NewMetricsCollector()
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.metrics)
	assert.NotZero(t, collector.metrics.StartTime)

	// 测试添加处理文件
	t.Run("add processed files", func(t *testing.T) {
		// 添加成功处理的文件
		collector.AddProcessedFile(1024*1024, false)   // 1MB
		collector.AddProcessedFile(2*1024*1024, false) // 2MB

		// 添加失败的文件
		collector.AddProcessedFile(512*1024, true) // 0.5MB

		time.Sleep(time.Millisecond)
		metrics := collector.GetMetrics()
		assert.Equal(t, 3, metrics.TotalFiles)
		assert.Equal(t, 1, metrics.FailedFiles)
		assert.InDelta(t, 3.5, metrics.ProcessedMB, 0.1) // 3.5MB
		assert.True(t, metrics.Duration > 0)
		assert.True(t, metrics.AverageSpeed > 0)
	})

	// 测试重置
	t.Run("reset metrics", func(t *testing.T) {
		collector.Reset()
		metrics := collector.GetMetrics()
		assert.Equal(t, 0, metrics.TotalFiles)
		assert.Equal(t, 0, metrics.FailedFiles)
		assert.Equal(t, float64(0), metrics.ProcessedMB)
		assert.True(t, metrics.StartTime.After(time.Now().Add(-time.Second)))
	})
}

func TestMetricsCollector_Concurrent(t *testing.T) {
	collector := NewMetricsCollector()

	// 测试并发安全性
	t.Run("concurrent access", func(t *testing.T) {
		done := make(chan struct{})
		go func() {
			for range 100 {
				collector.AddProcessedFile(1024*1024, false)
			}
			done <- struct{}{}
		}()
		go func() {
			for range 100 {
				collector.AddProcessedFile(1024*1024, true)
			}
			done <- struct{}{}
		}()
		go func() {
			for range 100 {
				collector.GetMetrics()
			}
			done <- struct{}{}
		}()

		// 等待所有协程完成
		for range 3 {
			<-done
		}

		metrics := collector.GetMetrics()
		assert.Equal(t, 200, metrics.TotalFiles)
		assert.Equal(t, 100, metrics.FailedFiles)
		assert.InDelta(t, 200, metrics.ProcessedMB, 1)
	})
}

func TestMetricsCollector_Calculation(t *testing.T) {
	collector := NewMetricsCollector()

	// 测试性能指标计算
	t.Run("metrics calculation", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			collector.Reset()
			startTime := time.Now()
			collector.metrics.StartTime = startTime

			// 添加一些处理文件
			collector.AddProcessedFile(100*1024*1024, false) // 100MB
			collector.AddProcessedFile(50*1024*1024, true)   // 50MB

			// 设置结束时间为 2 秒后
			time.Sleep(2 * time.Second)

			metrics := collector.GetMetrics()
			assert.Equal(t, 2, metrics.TotalFiles)
			assert.Equal(t, 1, metrics.FailedFiles)
			assert.InDelta(t, 150, metrics.ProcessedMB, 1)
			assert.InDelta(t, 75, metrics.AverageSpeed, 1)
			assert.Equal(t, 2*time.Second, metrics.Duration)
		})
	})
}

func TestMetricsCollector_EdgeCases(t *testing.T) {
	collector := NewMetricsCollector()

	// 测试边界条件
	t.Run("edge cases", func(t *testing.T) {
		collector.Reset()

		// 测试零大小文件
		collector.AddProcessedFile(0, false)
		metrics := collector.GetMetrics()
		assert.Equal(t, 1, metrics.TotalFiles)
		assert.Equal(t, float64(0), metrics.ProcessedMB)

		// 测试极小文件
		collector.AddProcessedFile(1, false)
		metrics = collector.GetMetrics()
		assert.True(t, metrics.ProcessedMB > 0)
		assert.True(t, metrics.ProcessedMB < 0.1)

		// 测试极大文件
		collector.AddProcessedFile(10*1024*1024*1024, false) // 10GB
		metrics = collector.GetMetrics()
		assert.InDelta(t, 10*1024, metrics.ProcessedMB, 1)
	})
}

func TestMetricsCollector_Integration(t *testing.T) {
	collector := NewMetricsCollector()

	// 测试完整工作流程
	t.Run("workflow", func(t *testing.T) {
		// 添加一批文件
		for i := range 10 {
			size := int64((i + 1) * 1024 * 1024) // 1MB 到 10MB
			failed := i%3 == 0                   // 每三个文件失败一个
			collector.AddProcessedFile(size, failed)
		}

		metrics := collector.GetMetrics()
		assert.Equal(t, 10, metrics.TotalFiles)
		assert.Equal(t, 4, metrics.FailedFiles)       // 0,3,6,9 失败
		assert.InDelta(t, 55, metrics.ProcessedMB, 1) // 总大小约 55MB
		assert.True(t, metrics.Duration >= 0)
		assert.True(t, metrics.AverageSpeed >= 0)

		// 重置并再次测试
		collector.Reset()
		assert.Equal(t, 0, collector.GetMetrics().TotalFiles)

		// 添加新的文件
		collector.AddProcessedFile(1024*1024, false)
		time.Sleep(time.Millisecond)
		metrics = collector.GetMetrics()
		assert.Equal(t, 1, metrics.TotalFiles)
		assert.Equal(t, 0, metrics.FailedFiles)
		assert.InDelta(t, 1, metrics.ProcessedMB, 0.1)
	})
}

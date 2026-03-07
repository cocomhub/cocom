// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/synctest"
	"time"

	"github.com/cocomhub/cocom/pkg/clog"
	"github.com/stretchr/testify/assert"
)

func TestMonitor_Basic(t *testing.T) {
	ctx := clog.NewTraceCtx("test")
	metrics := NewMetricsCollector()
	monitor := NewMonitor(ctx, metrics, time.Millisecond*100)

	// 测试基本功能
	t.Run("basic", func(t *testing.T) {
		monitor.Start()
		defer monitor.Stop()

		// 添加一些测试数据
		metrics.AddProcessedFile(1024*1024, false)   // 1MB
		metrics.AddProcessedFile(2*1024*1024, false) // 2MB
		metrics.AddProcessedFile(512*1024, true)     // 0.5MB

		// 等待监控更新
		time.Sleep(time.Millisecond * 200)

		stats := monitor.GetStats()
		assert.NotNil(t, stats)
		assert.Equal(t, 3, stats.TotalFiles)
		assert.Equal(t, 2, stats.ProcessedFiles)
		assert.Equal(t, 1, stats.FailedFiles)
		assert.InDelta(t, 3.5, stats.ProcessedMB, 0.1)
		assert.True(t, stats.Duration > 0)
		assert.True(t, stats.CurrentSpeed >= 0)
		assert.True(t, stats.NumGoroutine > 0)
		assert.Equal(t, runtime.NumCPU(), stats.NumCPU)
	})
}

func TestMonitor_SaveStats(t *testing.T) {
	ctx := clog.NewTraceCtx("test")
	metrics := NewMetricsCollector()
	monitor := NewMonitor(ctx, metrics, time.Millisecond*100)

	// 测试保存统计数据
	t.Run("save stats", func(t *testing.T) {
		tmpDir := t.TempDir()
		statsPath := filepath.Join(tmpDir, "stats.json")

		monitor.Start()
		defer monitor.Stop()

		// 添加一些测试数据
		metrics.AddProcessedFile(1024*1024, false)
		time.Sleep(time.Millisecond * 200)

		// 保存统计数据
		err := monitor.SaveStats(statsPath)
		assert.NoError(t, err)
		assert.FileExists(t, statsPath)

		// 验证保存的数据
		data, err := os.ReadFile(statsPath)
		assert.NoError(t, err)

		var stats MonitorStats
		err = json.Unmarshal(data, &stats)
		assert.NoError(t, err)
		assert.Equal(t, 1, stats.TotalFiles)
		assert.Equal(t, 1, stats.ProcessedFiles)
		assert.InDelta(t, 1, stats.ProcessedMB, 0.1)
	})
}

func TestMonitor_Performance(t *testing.T) {
	ctx := clog.NewTraceCtx("test")

	// 测试性能统计
	t.Run("performance", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			metrics := NewMetricsCollector()
			monitor := NewMonitor(ctx, metrics, time.Millisecond*50)

			monitor.Start()
			defer monitor.Stop()

			// 模拟高负载
			for i := 0; i < 100; i++ {
				metrics.AddProcessedFile(1024*1024, i%5 == 0)
				time.Sleep(time.Millisecond)
			}
			time.Sleep(100 * time.Millisecond)

			stats := monitor.GetStats()
			assert.Equal(t, 100, stats.TotalFiles)
			assert.Equal(t, 80, stats.ProcessedFiles)
			assert.Equal(t, 20, stats.FailedFiles)
			assert.InDelta(t, 100, stats.ProcessedMB, 1)

			// 验证性能指标
			perfStats := monitor.GetPerformanceStats()
			assert.NotNil(t, perfStats)
			assert.True(t, perfStats.CPUUsage > 0, "cpu usage should be > 0, actual: %v", perfStats.CPUUsage)
			assert.True(t, perfStats.MemoryUsage > 0, "memory usage should be > 0, actual: %v", perfStats.MemoryUsage)
			assert.True(t, perfStats.ErrorCount == 20, "error count should be 20, actual: %d", perfStats.ErrorCount)
			assert.True(t, perfStats.RetryCount >= 0, "retry count should be >= 0, actual: %d", perfStats.RetryCount)

			// 验证资源使用
			resStats := monitor.GetResourceStats()
			assert.NotNil(t, resStats)
			assert.True(t, resStats.CPUTime > 0, "cpu time should be > 0, actual: %v", resStats.CPUTime)
			assert.True(t, resStats.MaxMemory > 0, "max memory should be > 0, actual: %v", resStats.MaxMemory)
			assert.True(t, resStats.DiskRead >= 0, "disk read should be >= 0, actual: %d", resStats.DiskRead)
			assert.True(t, resStats.DiskWrite >= 0, "disk write should be >= 0, actual: %d", resStats.DiskWrite)
		})
	})
}

func TestMonitor_Checkpoints(t *testing.T) {
	ctx := clog.NewTraceCtx("test")
	metrics := NewMetricsCollector()
	monitor := NewMonitor(ctx, metrics, time.Millisecond*100)

	// 测试检查点功能
	t.Run("checkpoints", func(t *testing.T) {
		monitor.Start()
		defer monitor.Stop()

		// 添加一些检查点
		monitor.checkpoints = append(monitor.checkpoints, "point1")
		monitor.checkpoints = append(monitor.checkpoints, "point2")
		monitor.checkpoints = append(monitor.checkpoints, "point3")

		// 添加一些重试队列
		monitor.retryQueue = append(monitor.retryQueue, "retry1.jpg")
		monitor.retryQueue = append(monitor.retryQueue, "retry2.jpg")

		// 验证获取方法
		checkpoints := monitor.GetCheckpoints()
		assert.Len(t, checkpoints, 3)
		assert.Equal(t, []string{"point1", "point2", "point3"}, checkpoints)

		retryQueue := monitor.GetRetryQueue()
		assert.Len(t, retryQueue, 2)
		assert.Equal(t, []string{"retry1.jpg", "retry2.jpg"}, retryQueue)
	})
}

package comic

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/suixibing/cocom/pkg/clog"
)

// MonitorStats 性能监控统计
type MonitorStats struct {
	StartTime      time.Time        `json:"startTime"`      // 开始时间
	Duration       time.Duration    `json:"duration"`       // 运行时长
	NumGoroutine   int              `json:"numGoroutine"`   // 协程数量
	NumCPU         int              `json:"numCpu"`         // CPU 核心数
	MemStats       runtime.MemStats `json:"memStats"`       // 内存统计
	ProcessedMB    float64          `json:"processedMb"`    // 处理数据量(MB)
	AverageSpeed   float64          `json:"averageSpeed"`   // 平均速度(MB/s)
	CurrentSpeed   float64          `json:"currentSpeed"`   // 当前速度(MB/s)
	TotalFiles     int              `json:"totalFiles"`     // 总文件数
	ProcessedFiles int              `json:"processedFiles"` // 已处理文件数
	FailedFiles    int              `json:"failedFiles"`    // 失败文件数
	CPUUsage       float64          `json:"cpuUsage"`       // CPU 使用率
	MemoryUsage    float64          `json:"memoryUsage"`    // 内存使用率
	DiskIO         int64            `json:"diskIo"`         // 磁盘 IO
	NetworkIO      int64            `json:"networkIo"`      // 网络 IO
	GCStats        runtime.MemStats `json:"gcStats"`        // GC 统计
	RetryCount     int              `json:"retryCount"`     // 重试计数
	QueueLength    int              `json:"queueLength"`    // 队列长度
}

// PerformanceStats 性能统计类型
type PerformanceStats struct {
	CPUUsage    float64          `json:"cpuUsage"`    // CPU 使用率
	MemoryUsage float64          `json:"memoryUsage"` // 内存使用率
	DiskIO      int64            `json:"diskIo"`      // 磁盘 IO
	NetworkIO   int64            `json:"networkIo"`   // 网络 IO
	GCStats     runtime.MemStats `json:"gcStats"`     // GC 统计
	ErrorCount  int              `json:"errorCount"`  // 错误计数
	RetryCount  int              `json:"retryCount"`  // 重试计数
	QueueLength int              `json:"queueLength"` // 队列长度
}

// ResourceStats 资源使用统计类型
type ResourceStats struct {
	CPUTime      time.Duration `json:"cpuTime"`      // CPU 时间
	MaxMemory    uint64        `json:"maxMemory"`    // 最大内存使用
	DiskRead     int64         `json:"diskRead"`     // 磁盘读取量
	DiskWrite    int64         `json:"diskWrite"`    // 磁盘写入量
	NetworkRead  int64         `json:"networkRead"`  // 网络读取量
	NetworkWrite int64         `json:"networkWrite"` // 网络写入量
}

// Monitor 性能监控器
type Monitor struct {
	ctx          context.Context
	cancel       context.CancelFunc
	interval     time.Duration
	stats        *MonitorStats
	metrics      *MetricsCollector
	performance  *PerformanceStats // 添加性能统计
	resources    *ResourceStats    // 添加资源统计
	checkpoints  []string          // 添加检查点列表
	retryQueue   []string          // 添加重试队列
	skippedFiles []string          // 添加跳过的文件
	mu           sync.RWMutex
}

// NewMonitor 创建性能监控器
func NewMonitor(ctx context.Context, metrics *MetricsCollector, interval time.Duration) *Monitor {
	ctx, cancel := context.WithCancel(ctx)
	return &Monitor{
		ctx:      ctx,
		cancel:   cancel,
		interval: interval,
		stats: &MonitorStats{
			StartTime: time.Now(),
			NumCPU:    runtime.NumCPU(),
		},
		metrics: metrics,
	}
}

// Start 启动监控
func (m *Monitor) Start() {
	go func() {
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()

		var lastProcessedMB float64
		var lastTime time.Time

		for {
			select {
			case <-m.ctx.Done():
				return
			case <-ticker.C:
				m.mu.Lock()
				// 更新基本统计信息
				m.stats.NumGoroutine = runtime.NumGoroutine()
				runtime.ReadMemStats(&m.stats.MemStats)
				m.stats.Duration = time.Since(m.stats.StartTime)

				// 更新性能指标
				metrics := m.metrics.GetMetrics()
				m.stats.ProcessedMB = metrics.ProcessedMB
				m.stats.AverageSpeed = metrics.AverageSpeed
				m.stats.TotalFiles = metrics.TotalFiles
				m.stats.ProcessedFiles = metrics.TotalFiles - metrics.FailedFiles
				m.stats.FailedFiles = metrics.FailedFiles

				// 计算当前速度
				if !lastTime.IsZero() {
					duration := time.Since(lastTime).Seconds()
					processed := m.stats.ProcessedMB - lastProcessedMB
					m.stats.CurrentSpeed = processed / duration
				}
				lastTime = time.Now()
				lastProcessedMB = m.stats.ProcessedMB

				// 更新 GC 统计
				m.stats.GCStats = m.stats.MemStats

				// 更新 IO 统计
				// TODO: 添加实际的 IO 统计逻辑

				// 更新错误计数
				m.stats.RetryCount = len(m.retryQueue)
				m.stats.QueueLength = len(m.checkpoints)

				m.mu.Unlock()

				// 输出监控日志
				m.logStats()
			}
		}
	}()
}

// Stop 停止监控
func (m *Monitor) Stop() {
	m.cancel()
}

// GetStats 获取监控统计
func (m *Monitor) GetStats() MonitorStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m.stats
}

// SaveStats 保存监控统计
func (m *Monitor) SaveStats(path string) error {
	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	m.mu.RLock()
	data, err := json.MarshalIndent(m.stats, "", "  ")
	m.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("序列化统计数据失败: %v", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("保存统计数据失败: %v", err)
	}

	return nil
}

// logStats 输出监控日志
func (m *Monitor) logStats() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clog.Infof(m.ctx, "性能监控统计:\n"+
		"运行时长: %v\n"+
		"协程数量: %d\n"+
		"内存使用: %.2f MB\n"+
		"处理数据: %.2f MB\n"+
		"平均速度: %.2f MB/s\n"+
		"当前速度: %.2f MB/s\n"+
		"总文件数: %d\n"+
		"已处理: %d\n"+
		"失败数: %d",
		m.stats.Duration,
		m.stats.NumGoroutine,
		float64(m.stats.MemStats.Alloc)/1024/1024,
		m.stats.ProcessedMB,
		m.stats.AverageSpeed,
		m.stats.CurrentSpeed,
		m.stats.TotalFiles,
		m.stats.ProcessedFiles,
		m.stats.FailedFiles,
	)
}

// 添加获取方法
func (m *Monitor) GetCheckpoints() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string{}, m.checkpoints...)
}

func (m *Monitor) GetRetryQueue() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string{}, m.retryQueue...)
}

func (m *Monitor) GetSkippedFiles() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string{}, m.skippedFiles...)
}

func (m *Monitor) GetStartTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats.StartTime
}

func (m *Monitor) GetDuration() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats.Duration
}

// GetPerformanceStats 获取性能统计
func (m *Monitor) GetPerformanceStats() *PerformanceStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return &PerformanceStats{
		CPUUsage:    float64(m.stats.NumGoroutine) / float64(m.stats.NumCPU) * 100,
		MemoryUsage: float64(m.stats.MemStats.Alloc) / float64(m.stats.MemStats.Sys) * 100,
		DiskIO:      m.stats.DiskIO,
		NetworkIO:   m.stats.NetworkIO,
		GCStats:     m.stats.GCStats,
		ErrorCount:  m.stats.FailedFiles,
		RetryCount:  m.stats.RetryCount,
		QueueLength: m.stats.QueueLength,
	}
}

// GetResourceStats 获取资源统计
func (m *Monitor) GetResourceStats() *ResourceStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return &ResourceStats{
		CPUTime:      m.stats.Duration,
		MaxMemory:    m.stats.MemStats.TotalAlloc,
		DiskRead:     m.stats.DiskIO,
		DiskWrite:    m.stats.DiskIO,
		NetworkRead:  m.stats.NetworkIO,
		NetworkWrite: m.stats.NetworkIO,
	}
}

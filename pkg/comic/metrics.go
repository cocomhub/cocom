package comic

import (
	"sync"
	"time"
)

// VerifyMetrics 验证性能指标
type VerifyMetrics struct {
	StartTime     time.Time     `json:"startTime"`     // 开始时间
	EndTime       time.Time     `json:"endTime"`       // 结束时间
	Duration      time.Duration `json:"duration"`      // 总耗时
	TaskSubmitted int           `json:"taskSubmitted"` // 已提交的任务数
	TaskFailed    int           `json:"taskFailed"`    // 失败的任务数
	TotalFiles    int           `json:"totalFiles"`    // 总文件数
	ProcessedMB   float64       `json:"processedMb"`   // 处理的总大小(MB)
	FailedFiles   int           `json:"failedFiles"`   // 失败文件数
	AverageSpeed  float64       `json:"averageSpeed"`  // 平均速度(MB/s)
}

// MetricsCollector 指标收集器
type MetricsCollector struct {
	mu      sync.RWMutex
	metrics *VerifyMetrics
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: &VerifyMetrics{
			StartTime: time.Now(),
		},
	}
}

// AddProcessedFile 添加处理文件
func (c *MetricsCollector) AddProcessedFile(size int64, failed bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics.TotalFiles++
	c.metrics.ProcessedMB += float64(size) / 1024 / 1024
	if failed {
		c.metrics.FailedFiles++
	}
}

// AddProcessedTask 添加已处理的任务数
func (c *MetricsCollector) TaskSubmitted() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics.TaskSubmitted++
}

func (c *MetricsCollector) TaskFailed() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics.TaskFailed++
}

// GetMetrics 获取性能指标
func (c *MetricsCollector) GetMetrics() VerifyMetrics {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics.EndTime = time.Now()
	c.metrics.Duration = c.metrics.EndTime.Sub(c.metrics.StartTime)
	if c.metrics.Duration > 0 {
		c.metrics.AverageSpeed = c.metrics.ProcessedMB / c.metrics.Duration.Seconds()
	}

	return *c.metrics
}

// Reset 重置指标
func (c *MetricsCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics = &VerifyMetrics{
		StartTime: time.Now(),
	}
}

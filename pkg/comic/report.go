package comic

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/suixibing/cocom/pkg/clog"
)

// VerifyReport 验证报告
type VerifyReport struct {
	StartTime     time.Time           `json:"start_time"`     // 开始时间
	EndTime       time.Time           `json:"end_time"`       // 结束时间
	Duration      time.Duration       `json:"duration"`       // 持续时间
	Pattern       string              `json:"pattern"`        // 匹配规则
	TotalComics   int                 `json:"total_comics"`   // 总漫画数
	ValidComics   int                 `json:"valid_comics"`   // 有效漫画数
	InvalidComics int                 `json:"invalid_comics"` // 无效漫画数
	FixedComics   int                 `json:"fixed_comics"`   // 修复漫画数
	TotalFiles    int                 `json:"total_files"`    // 总文件数
	ValidFiles    int                 `json:"valid_files"`    // 有效文件数
	InvalidFiles  int                 `json:"invalid_files"`  // 无效文件数
	FixedFiles    int                 `json:"fixed_files"`    // 修复文件数
	ProcessedMB   float64             `json:"processed_mb"`   // 处理数据量(MB)
	AverageSpeed  float64             `json:"average_speed"`  // 平均速度(MB/s)
	Results       []*VerifyResult     `json:"results"`        // 详细结果
	ErrorDetails  map[string][]string `json:"error_details"`  // 错误详情
	RetryHistory  []RetryRecord       `json:"retry_history"`  // 重试历史
	Performance   PerformanceStats    `json:"performance"`    // 性能统计
	ResourceUsage ResourceStats       `json:"resource_usage"` // 资源使用
	Suggestions   []string            `json:"suggestions"`    // 优化建议
}

type RetryRecord struct {
	Time    time.Time `json:"time"`
	File    string    `json:"file"`
	Error   string    `json:"error"`
	Attempt int       `json:"attempt"`
	Success bool      `json:"success"`
}

// GenerateReport 生成验证报告
func (v *ComicVerifier) GenerateReport() *VerifyReport {
	v.mu.RLock()
	defer v.mu.RUnlock()

	metrics := v.metrics.GetMetrics()
	report := &VerifyReport{
		StartTime:    v.progress.StartTime,
		EndTime:      time.Now(),
		Pattern:      v.lastPattern,
		TotalFiles:   v.progress.Total,
		ValidFiles:   v.progress.Total - v.progress.Invalid,
		InvalidFiles: v.progress.Invalid,
		FixedFiles:   v.progress.Fixed,
		ProcessedMB:  metrics.ProcessedMB,
		AverageSpeed: metrics.AverageSpeed,
		Results:      v.results,
	}

	report.Duration = report.EndTime.Sub(report.StartTime)

	// 统计漫画数量
	comicStats := make(map[string]bool)
	for _, result := range v.results {
		comicStats[result.ComicID.Hex()] = result.InvalidCount == 0
	}

	report.TotalComics = len(comicStats)
	for _, valid := range comicStats {
		if valid {
			report.ValidComics++
		} else {
			report.InvalidComics++
		}
	}

	return report
}

// SaveReport 保存验证报告
func (v *ComicVerifier) SaveReport(path string) error {
	report := v.GenerateReport()

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	// 保存为 JSON 格式
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化报告失败: %v", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("保存报告失败: %v", err)
	}

	clog.Infof(v.ctx, "验证报告已保存到: %s", path)
	return nil
}

// PrintReport 打印验证报告
func (v *ComicVerifier) PrintReport() {
	report := v.GenerateReport()

	fmt.Printf("\n验证报告:\n")
	fmt.Printf("开始时间: %v\n", report.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("结束时间: %v\n", report.EndTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("持续时间: %v\n", report.Duration)
	fmt.Printf("匹配规则: %s\n", report.Pattern)
	fmt.Printf("\n漫画统计:\n")
	fmt.Printf("- 总数: %d\n", report.TotalComics)
	fmt.Printf("- 有效: %d\n", report.ValidComics)
	fmt.Printf("- 无效: %d\n", report.InvalidComics)
	fmt.Printf("\n文件统计:\n")
	fmt.Printf("- 总数: %d\n", report.TotalFiles)
	fmt.Printf("- 有效: %d\n", report.ValidFiles)
	fmt.Printf("- 无效: %d\n", report.InvalidFiles)
	fmt.Printf("- 已修复: %d\n", report.FixedFiles)
	fmt.Printf("\n性能指标:\n")
	fmt.Printf("- 处理数据: %.2f MB\n", report.ProcessedMB)
	fmt.Printf("- 平均速度: %.2f MB/s\n", report.AverageSpeed)
}

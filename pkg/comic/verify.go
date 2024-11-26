package comic

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/imaging"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// ComicVerifier 漫画验证器
type ComicVerifier struct {
	ctx         context.Context
	cancel      context.CancelFunc
	db          *mongo.Database
	downloader  *Downloader
	progress    *VerifyProgress
	metrics     *MetricsCollector
	monitor     *Monitor        // 添加监控器
	lastChecked string          // 最后检查的漫画 ID
	lastPattern string          // 最后使用的匹配规则
	results     []*VerifyResult // 验证结果列表
	mu          sync.RWMutex
}

// VerifyProgress 验证进度
type VerifyProgress struct {
	Total     int       `json:"total"`      // 总图片数
	Checked   int       `json:"checked"`    // 已检查数
	Invalid   int       `json:"invalid"`    // 损坏数
	Fixed     int       `json:"fixed"`      // 已修复数
	Progress  float64   `json:"progress"`   // 进度百分比
	StartTime time.Time `json:"start_time"` // 开始时间
}

// VerifyOptions 验证选项
type VerifyOptions struct {
	Pattern    string // 匹配规则，如 ".*"
	AutoFix    bool   // 是否自动修复
	Concurrent int    // 并发数
}

// ScheduleConfig 定时检查配置
type ScheduleConfig struct {
	Pattern       string        // 匹配规则
	Interval      time.Duration // 检查间隔
	AutoFix       bool          // 是否自动修复
	Concurrent    int           // 并发数
	RetryInterval time.Duration `json:"retry_interval"` // 重试间隔
	MaxRetries    int           `json:"max_retries"`    // 最大重试次数
	TimeWindow    []TimeRange   `json:"time_window"`    // 时间窗口
	Priority      int           `json:"priority"`       // 任务优先级
	Timeout       time.Duration `json:"timeout"`        // 超时时间
}

type TimeRange struct {
	Start string `json:"start"` // 格式: "HH:MM"
	End   string `json:"end"`   // 格式: "HH:MM"
}

// NewComicVerifier 创建漫画验证器
func NewComicVerifier(ctx context.Context, db *mongo.Database) *ComicVerifier {
	ctx, cancel := context.WithCancel(ctx)
	metrics := NewMetricsCollector()
	return &ComicVerifier{
		ctx:        ctx,
		cancel:     cancel,
		db:         db,
		downloader: NewDownloader(ctx),
		progress: &VerifyProgress{
			StartTime: time.Now(),
		},
		metrics: metrics,
		monitor: NewMonitor(ctx, metrics, time.Second), // 创建监控器
	}
}

// Start 启动验证任务
func (v *ComicVerifier) Start(opts VerifyOptions) error {
	// 启动监控器
	v.monitor.Start()
	defer v.monitor.Stop()

	// 如果有保存的状态，尝试加载
	if err := v.LoadState(); err == nil {
		clog.Infof(v.ctx, "从上次检查位置继续: %s", v.lastChecked)
	}

	v.mu.Lock()
	v.lastPattern = opts.Pattern
	v.results = make([]*VerifyResult, 0)
	v.mu.Unlock()

	// 查询需要验证的漫画
	filter := bson.M{
		"title": bson.M{"$regex": opts.Pattern},
	}
	// 如果有上次检查位置，从该位置继续
	if v.lastChecked != "" {
		filter["_id"] = bson.M{"$gt": v.lastChecked}
	}

	cursor, err := v.db.Collection("comics").Find(v.ctx, filter)
	if err != nil {
		return err
	}
	defer cursor.Close(v.ctx)

	// 创建工作池
	jobs := make(chan *Comic)
	results := make(chan *VerifyResult)
	var wg sync.WaitGroup

	// 启动工作协程
	for i := 0; i < opts.Concurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for comic := range jobs {
				result := v.verifyComic(comic, opts.AutoFix)
				results <- result
			}
		}()
	}

	// 收集结果
	go func() {
		for result := range results {
			v.mu.Lock()
			v.results = append(v.results, result)
			v.progress.Checked += len(result.Images)
			v.progress.Invalid += result.InvalidCount
			v.progress.Fixed += result.FixedCount
			v.progress.Progress = float64(v.progress.Checked) / float64(v.progress.Total) * 100
			v.mu.Unlock()

			// 更新数据库
			v.updateComicStatus(result)
		}
	}()

	// 发送任务
	var comics []*Comic
	if err := cursor.All(v.ctx, &comics); err != nil {
		return err
	}

	v.mu.Lock()
	for _, comic := range comics {
		v.progress.Total += len(comic.Images)
	}
	v.mu.Unlock()

	for _, comic := range comics {
		select {
		case <-v.ctx.Done():
			return v.ctx.Err()
		case jobs <- comic:
		}
	}
	close(jobs)

	// 等待所有任务完成
	wg.Wait()
	close(results)

	// 定期保存状态
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-v.ctx.Done():
				return
			case <-ticker.C:
				if err := v.SaveState(); err != nil {
					clog.Errorf(v.ctx, "保存验证状态失败: %v", err)
				}
			}
		}
	}()

	return nil
}

// verifyComic 验证单个漫画
func (v *ComicVerifier) verifyComic(comic *Comic, autoFix bool) *VerifyResult {
	result := &VerifyResult{
		ComicID:   comic.ID,
		Title:     comic.Title,
		Images:    make([]*ImageResult, 0, len(comic.Images)),
		Timestamp: time.Now(),
	}

	for _, img := range comic.Images {
		imgResult := v.verifyImage(img)
		result.Images = append(result.Images, imgResult)

		if imgResult.Invalid {
			result.InvalidCount++
			if autoFix {
				if err := v.fixImage(img); err == nil {
					result.FixedCount++
				}
			}
		}
	}

	return result
}

// verifyImage 验证单个图片
func (v *ComicVerifier) verifyImage(img *Image) *ImageResult {
	ctx := clog.NewTraceCtx("verify_image")
	info, err := imaging.VerifyImage(ctx, img.Path)

	// 更新性能指标
	if info != nil {
		v.metrics.AddProcessedFile(info.Size, err != nil)
	}

	return &ImageResult{
		Path:    img.Path,
		URL:     img.URL,
		Invalid: err != nil || info.Invalid,
		Error:   err.Error(),
		Info:    info,
	}
}

// fixImage 修复损坏的图片
func (v *ComicVerifier) fixImage(img *Image) error {
	return v.downloader.Download(img.URL, img.Path)
}

// updateComicStatus 更新漫画状态
func (v *ComicVerifier) updateComicStatus(result *VerifyResult) error {
	update := bson.M{
		"$set": bson.M{
			"valid":         result.InvalidCount == 0,
			"invalid_count": result.InvalidCount,
			"fixed_count":   result.FixedCount,
			"last_verify":   time.Now(),
		},
	}
	_, err := v.db.Collection("comics").UpdateByID(v.ctx, result.ComicID, update)
	return err
}

// GetProgress 获取验证进度
func (v *ComicVerifier) GetProgress() *VerifyProgress {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.progress
}

// Cancel 取消验证任务
func (v *ComicVerifier) Cancel() {
	v.cancel()
}

// GetMetrics 获取性能指标
func (v *ComicVerifier) GetMetrics() *VerifyMetrics {
	return v.metrics.GetMetrics()
}

// VerifyState 验证状态
type VerifyState struct {
	LastChecked  string            `json:"last_checked"`  // 最后检查的漫画 ID
	Progress     *VerifyProgress   `json:"progress"`      // 验证进度
	Timestamp    time.Time         `json:"timestamp"`     // 时间戳
	Checkpoints  []string          `json:"checkpoints"`   // 检查点列表
	ErrorFiles   []string          `json:"error_files"`   // 错误文件列表
	RetryQueue   []string          `json:"retry_queue"`   // 重试队列
	SkippedFiles []string          `json:"skipped_files"` // 跳过的文件
	StartTime    time.Time         `json:"start_time"`    // 开始时间
	Duration     time.Duration     `json:"duration"`      // 持续时间
	Performance  *PerformanceStats `json:"performance"`   // 性能统计
	Resources    *ResourceStats    `json:"resources"`     // 资源统计
}

func (v *ComicVerifier) SaveState() error {
	v.mu.RLock()
	state := &VerifyState{
		LastChecked:  v.lastChecked,
		Progress:     v.progress,
		Timestamp:    time.Now(),
		Checkpoints:  v.monitor.GetCheckpoints(),
		ErrorFiles:   v.monitor.GetErrorFiles(),
		RetryQueue:   v.monitor.GetRetryQueue(),
		SkippedFiles: v.monitor.GetSkippedFiles(),
		StartTime:    v.monitor.GetStartTime(),
		Duration:     v.monitor.GetDuration(),
		Performance:  v.monitor.GetPerformanceStats(),
		Resources:    v.monitor.GetResourceStats(),
	}
	v.mu.RUnlock()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("verify_state.json", data, 0o644)
}

func (v *ComicVerifier) LoadState() error {
	data, err := os.ReadFile("verify_state.json")
	if err != nil {
		return err
	}
	var state VerifyState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	v.mu.Lock()
	v.lastChecked = state.LastChecked
	v.progress = state.Progress
	v.mu.Unlock()

	return nil
}

type PriorityQueue struct {
	items []*Comic
	mu    sync.RWMutex
	rules []PriorityRule
}

type PriorityRule struct {
	Name     string
	Weight   int
	Evaluate func(*Comic) float64
}

func (pq *PriorityQueue) AddRule(rule PriorityRule) {
	pq.rules = append(pq.rules, rule)
}

func (pq *PriorityQueue) calculatePriority(comic *Comic) float64 {
	var priority float64
	for _, rule := range pq.rules {
		priority += float64(rule.Weight) * rule.Evaluate(comic)
	}
	return priority
}

func (pq *PriorityQueue) Push(comic *Comic) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.items = append(pq.items, comic)
	sort.Slice(pq.items, func(i, j int) bool {
		return pq.calculatePriority(pq.items[i]) < pq.calculatePriority(pq.items[j])
	})
}

func (pq *PriorityQueue) Pop() *Comic {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	if len(pq.items) == 0 {
		return nil
	}
	comic := pq.items[0]
	pq.items = pq.items[1:]
	return comic
}

// StartSchedule 启动定时检查
func (v *ComicVerifier) StartSchedule(cfg ScheduleConfig) error {
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-v.ctx.Done():
			return v.ctx.Err()
		case <-ticker.C:
			err := v.Start(VerifyOptions{
				Pattern:    cfg.Pattern,
				AutoFix:    cfg.AutoFix,
				Concurrent: cfg.Concurrent,
			})
			if err != nil {
				clog.Errorf(v.ctx, "定时检查失败: %v", err)
			}
		}
	}
}

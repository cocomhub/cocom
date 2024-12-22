package comic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/robfig/cron/v3"
	"github.com/rs/xid"
	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/imaging"
	"go.uber.org/atomic"
)

// VerifyOptions 验证选项
type VerifyOptions struct {
	ComicFilter
	AutoFix    bool  `json:"autoFix"`    // 是否自动修复
	MaxWorkers int32 `json:"maxWorkers"` // 最大并发数
}

// VerifyProgress 验证进度
type VerifyProgress struct {
	TaskID    string        `json:"taskId"`    // 任务ID
	Total     *atomic.Int32 `json:"total"`     // 总数
	Current   *atomic.Int32 `json:"current"`   // 当前进度
	Invalid   *atomic.Int32 `json:"invalid"`   // 无效数
	Fixed     *atomic.Int32 `json:"fixed"`     // 修复数
	StartTime time.Time     `json:"startTime"` // 开始时间
	Status    *atomic.Value `json:"status"`    // 状态
	Error     error         `json:"error"`     // 错误信息

	mu      sync.RWMutex
	running []string
}

// VerifyStatus 验证状态
type VerifyStatus string

const (
	VerifyStatusPending   VerifyStatus = "pending"
	VerifyStatusRunning   VerifyStatus = "running"
	VerifyStatusCompleted VerifyStatus = "completed"
	VerifyStatusError     VerifyStatus = "error"
	VerifyStatusCanceled  VerifyStatus = "canceled"
)

// VerifyTask 验证任务
type VerifyTask struct {
	ID       string             `json:"id"`       // 任务ID
	Progress *VerifyProgress    `json:"progress"` // 进度
	Cancel   context.CancelFunc `json:"-"`        // 取消函数
}

// VerifyImageResult 图片验证结果
type VerifyImageResult struct {
	Path    string             `json:"path"`    // 文件路径
	Invalid bool               `json:"invalid"` // 是否无效
	Error   error              `json:"error"`   // 错误信息
	Info    *imaging.ImageInfo `json:"info"`    // 图片信息
}

// ScheduleConfig 定时任务配置
type ScheduleConfig struct {
	Pattern       string         `json:"pattern"`       // 标题匹配模式
	Interval      time.Duration  `json:"interval"`      // 检查间隔
	AutoFix       bool           `json:"autoFix"`       // 自动修复
	RetryInterval time.Duration  `json:"retryInterval"` // 重试间隔
	Cron          string         `json:"cron"`          // cron表达式
	Options       *VerifyOptions `json:"options"`       // 验证选项
	Active        bool           `json:"active"`        // 是否激活
	MaxRetry      int            `json:"maxRetry"`      // 最大重试次数
	RetryWait     time.Duration  `json:"retryWait"`     // 重试等待时间
}

// MarshalJSON 自定义JSON序列化
func (p *VerifyProgress) MarshalJSON() ([]byte, error) {
	type Alias VerifyProgress
	return json.Marshal(&struct {
		Total   int32        `json:"total"`
		Current int32        `json:"current"`
		Invalid int32        `json:"invalid"`
		Fixed   int32        `json:"fixed"`
		Running []string     `json:"running"`
		Status  VerifyStatus `json:"status"`
		*Alias
	}{
		Total:   p.Total.Load(),
		Current: p.Current.Load(),
		Invalid: p.Invalid.Load(),
		Fixed:   p.Fixed.Load(),
		Running: p.Running(),
		Status:  p.Status.Load().(VerifyStatus),
		Alias:   (*Alias)(p),
	})
}

// UnmarshalJSON 自定义JSON反序列化
func (p *VerifyProgress) UnmarshalJSON(data []byte) error {
	type Alias VerifyProgress
	aux := &struct {
		Total   int32        `json:"total"`
		Current int32        `json:"current"`
		Invalid int32        `json:"invalid"`
		Fixed   int32        `json:"fixed"`
		Running []string     `json:"running"`
		Status  VerifyStatus `json:"status"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	p.Total = atomic.NewInt32(aux.Total)
	p.Current = atomic.NewInt32(aux.Current)
	p.Invalid = atomic.NewInt32(aux.Invalid)
	p.Fixed = atomic.NewInt32(aux.Fixed)
	p.Status = &atomic.Value{}
	p.Status.Store(aux.Status)
	return nil
}

// NewVerifyProgress 创建新的进度跟踪器
func NewVerifyProgress(taskID string) *VerifyProgress {
	status := &atomic.Value{}
	status.Store(VerifyStatusPending)
	return &VerifyProgress{
		TaskID:    taskID,
		Total:     atomic.NewInt32(0),
		Current:   atomic.NewInt32(0),
		Invalid:   atomic.NewInt32(0),
		Fixed:     atomic.NewInt32(0),
		StartTime: time.Now(),
		Status:    status,
	}
}

func (p *VerifyProgress) Running() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

func (p *VerifyProgress) Start(id string) {
	p.mu.Lock()
	p.running = append(p.running, id)
	p.mu.Unlock()
}

func (p *VerifyProgress) End(id string) {
	p.mu.Lock()
	for i, v := range p.running {
		if v == id {
			p.running = append(p.running[:i], p.running[i+1:]...)
			break
		}
	}
	p.mu.Unlock()
}

// SetError 设置错误信息
func (p *VerifyProgress) SetError(err interface{}) {
	p.Error = fmt.Errorf("%v", err)
}

// UpdateProgress 更新进度
func (p *VerifyProgress) UpdateProgress(current, invalid, fixed int32) {
	p.Current.Store(current)
	p.Invalid.Store(invalid)
	p.Fixed.Store(fixed)

	// 更新状态
	if current >= p.Total.Load() {
		p.Status.Store(VerifyStatusCompleted)
	}
}

// SetMessage 设置最后消息
func (p *VerifyProgress) SetMessage(msg string) {
	// TODO: 实现最后消息设置
}

// IsCompleted 检查是否完成
func (p *VerifyProgress) IsCompleted() bool {
	status := p.GetStatus()
	return status == VerifyStatusCompleted || status == VerifyStatusError || status == VerifyStatusCanceled
}

// GetStatus 获取检查状态
func (p *VerifyProgress) GetStatus() VerifyStatus {
	return p.Status.Load().(VerifyStatus)
}

// GetProgress 获取进度百分比
func (p *VerifyProgress) GetProgress() float64 {
	if p.Total.Load() == 0 {
		return 0
	}
	return float64(p.Current.Load()) / float64(p.Total.Load()) * 100
}

// GetProgress 获取任务进度
func (t *VerifyTask) GetProgress() *VerifyProgress {
	return t.Progress
}

// Done 完成任务
func (t *VerifyTask) Done() {
	t.Cancel()
}

// ComicVerifier 漫画验证器
type ComicVerifier struct {
	ctx        context.Context
	cancel     context.CancelFunc
	storage    Storage
	downloader *Downloader
	metrics    *MetricsCollector
	pool       *ants.Pool
	tasks      sync.Map
	scheduler  *cron.Cron
	progressMu sync.RWMutex
	progress   map[string]*VerifyProgress
}

// NewComicVerifier 创建漫画验证器
func NewComicVerifier(ctx context.Context, storage Storage) (*ComicVerifier, error) {
	ctx, cancel := context.WithCancel(ctx)

	poolSize := runtime.NumCPU()
	clog.Infof(ctx, "创建漫画验证器工作池 size:%v", poolSize)

	// 创建工作池
	pool, err := ants.NewPool(poolSize,
		ants.WithPreAlloc(true),
		ants.WithPanicHandler(func(i interface{}) {
			clog.Errorf(ctx, "Panic in worker: %v", i)
		}),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create worker pool failed: %w", err)
	}

	return &ComicVerifier{
		ctx:        ctx,
		cancel:     cancel,
		storage:    storage,
		downloader: NewDownloader(),
		metrics:    NewMetricsCollector(),
		pool:       pool,
		tasks:      sync.Map{},
		scheduler:  cron.New(cron.WithSeconds()),
		progress:   make(map[string]*VerifyProgress),
	}, nil
}

// Start 开始验证任务
func (v *ComicVerifier) Start(ctx context.Context, opts *VerifyOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("验证选项为空")
	}

	taskID := xid.New().String()
	progress := NewVerifyProgress(taskID)

	// 创建任务上下文并存储
	taskCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	task := &VerifyTask{
		ID:       taskID,
		Progress: progress,
		Cancel:   cancel,
	}
	v.tasks.Store(taskID, task)

	// 查找匹配的漫画
	comics, err := v.storage.Find(ctx, &opts.ComicFilter)
	if err != nil {
		cancel()
		return "", fmt.Errorf("查找漫画失败: %w", err)
	}

	progress.Total.Store(int32(len(comics)))

	v.progressMu.Lock()
	v.progress[taskID] = progress
	v.progressMu.Unlock()

	// 启动验证任务
	go v.runTask(taskCtx, task, comics, opts)

	return taskID, nil
}

// runTask 运行验证任务
func (v *ComicVerifier) runTask(ctx context.Context, task *VerifyTask, comics []Comic, opts *VerifyOptions) {
	defer v.cleanupTask(task.ID)

	var wg sync.WaitGroup
	task.Progress.Status.Store(VerifyStatusRunning)
	for _, c := range comics {
		if task.Progress.Status.Load() == VerifyStatusCanceled {
			wg.Wait()
			return
		}

		wg.Add(1)
		err := v.pool.Submit(func() {
			defer wg.Done()

			task.Progress.Start(c.GetID())
			result := v.verifyComic(ctx, c, opts.AutoFix)
			task.Progress.End(c.GetID())

			task.Progress.Current.Inc()
			task.Progress.Invalid.Add(result.InvalidCount)
			task.Progress.Fixed.Add(result.FixedCount)

			c.SetVerifyResult(result)
			err := v.storage.Update(ctx, c)
			if err != nil {
				clog.Errorf(ctx, "update verify result failed. task:%s task:%v err:%s", result.ID, result, err)
			}
		})
		if err != nil {
			clog.Errorf(ctx, "提交验证任务失败: %v", err)
			wg.Done()
		}
	}

	wg.Wait()
}

// cleanupTask 清理任务
func (v *ComicVerifier) cleanupTask(taskID string) {
	v.progressMu.Lock()
	delete(v.progress, taskID)
	v.progressMu.Unlock()

	v.tasks.Delete(taskID)
}

// verifyComic 验证单个漫画
func (v *ComicVerifier) verifyComic(ctx context.Context, comic Comic, autoFix bool) *VerifyResult {
	result := &VerifyResult{
		ID:        xid.New().String(),
		ComicID:   comic.GetID(),
		Valid:     true,
		Timestamp: time.Now(),
	}
	var errs []error

	// 验证所有图片
	for _, img := range comic.GetImages() {
		imgResult := v.verifyImage(ctx, &img)
		if imgResult.Invalid {
			clog.Warnf(ctx, "[%s] 验证图片异常[%s]: %s. 异常: %s", result.ID, result.ComicID, img.Path, imgResult.Error)
			if autoFix {
				fixErr := v.fixImage(ctx, &img)
				if fixErr == nil {
					result.FixedCount++
					clog.Debugf(ctx, "[%s] 修复异常图片成功[%s]: %s", result.ID, result.ComicID, img.Path)
					continue
				}
				clog.Warnf(ctx, "[%s] 修复异常图片失败[%s]: %s. url[%s] 失败原因: %s", result.ID, result.ComicID, img.Path, img.URL, fixErr)
				imgResult.Error = errors.Join(imgResult.Error, fixErr)
			}
			errs = append(errs, imgResult.Error)
			result.Valid = false
			result.InvalidCount++
		}
	}

	if result.Valid {
		clog.Debugf(ctx, "[%s] 验证漫画结束[%s]", result.ID, result.ComicID)
	} else {
		result.Error = errors.Join(errs...)
		clog.Warnf(ctx, "[%s] 验证漫画存在异常[%s]. 修复成功:%d 仍然异常:%d", result.ID, result.ComicID, result.FixedCount, result.InvalidCount)
	}
	return result
}

// verifyImage 验证单张图片
func (v *ComicVerifier) verifyImage(ctx context.Context, img *Image) *VerifyImageResult {
	result := &VerifyImageResult{
		Path: img.Path,
	}

	// 检查文件是否存在
	if _, err := os.Stat(img.Path); os.IsNotExist(err) {
		result.Invalid = true
		result.Error = fmt.Errorf("文件不存在")
		return result
	}

	// 打开文件
	file, err := os.Open(img.Path)
	if err != nil {
		result.Invalid = true
		result.Error = fmt.Errorf("打开文件失败: %w", err)
		return result
	}
	defer file.Close()

	// 检查文件完整性
	if info, err := imaging.VerifyImage(ctx, img.Path); err != nil {
		result.Invalid = true
		result.Error = fmt.Errorf("解码图片失败: %w", err)
		return result
	} else {
		result.Info = info
	}

	return result
}

// fixImage 修复损坏的图片
func (v *ComicVerifier) fixImage(ctx context.Context, img *Image) error {
	fixImg := *img
	fixImg.Path += ".fix"

	err := v.downloader.Download(ctx, fixImg.URL, fixImg.Path)
	if err != nil {
		return err
	}

	imgResult := v.verifyImage(ctx, &fixImg)
	if imgResult.Invalid {
		_ = os.Remove(fixImg.Path)
		return imgResult.Error
	}

	return os.Rename(fixImg.Path, img.Path)
}

// GetTask 获取任务信息
func (v *ComicVerifier) GetTask(ctx context.Context, taskID string) (*VerifyTask, error) {
	value, ok := v.tasks.Load(taskID)
	if !ok {
		return nil, fmt.Errorf("任务不存在: %s", taskID)
	}

	task, ok := value.(*VerifyTask)
	if !ok {
		return nil, fmt.Errorf("任务异常: %s", taskID)
	}
	return task, nil
}

// CancelTask 取消验证任务
func (v *ComicVerifier) CancelTask(ctx context.Context, taskID string) error {
	v.progressMu.Lock()
	defer v.progressMu.Unlock()

	progress, ok := v.progress[taskID]
	if !ok {
		return fmt.Errorf("任务不存在: %s", taskID)
	}

	if progress.IsCompleted() {
		return fmt.Errorf("任务已完成: %s", taskID)
	}

	progress.Status.Store(VerifyStatusCanceled)
	clog.Infof(ctx, "任务已取消: %s", taskID)
	return nil
}

// GetTasks 获取所有任务
func (v *ComicVerifier) GetTasks() []*VerifyTask {
	v.progressMu.RLock()
	defer v.progressMu.RUnlock()

	tasks := make([]*VerifyTask, 0, len(v.progress))
	for _, progress := range v.progress {
		tasks = append(tasks, &VerifyTask{
			ID:       progress.TaskID,
			Progress: progress,
		})
	}
	return tasks
}

// GetTaskProgress 获取任务进度
func (v *ComicVerifier) GetTaskProgress(taskID string) *VerifyProgress {
	v.progressMu.RLock()
	defer v.progressMu.RUnlock()
	return v.progress[taskID]
}

// StartSchedule 启动定时任务
func (v *ComicVerifier) StartSchedule(ctx context.Context, cfg *ScheduleConfig) error {
	if cfg == nil {
		return fmt.Errorf("定时任务配置为空")
	}

	if !cfg.Active {
		clog.Infof(ctx, "定时任务未激活，跳过启动")
		return nil
	}

	if v.scheduler == nil {
		v.scheduler = cron.New(cron.WithSeconds())
	}

	// 如果没有指定cron表达式，使用interval生成
	if cfg.Cron == "" && cfg.Interval > 0 {
		cfg.Cron = fmt.Sprintf("@every %s", cfg.Interval.String())
	}

	if cfg.Cron == "" {
		return fmt.Errorf("未指定执行时间")
	}

	clog.Infof(ctx, "启动定时任务，执行规则：%s", cfg.Cron)

	// 创建任务上下文
	taskCtx, cancel := context.WithCancel(ctx)
	v.tasks.Store(cfg.Pattern, cancel)

	_, err := v.scheduler.AddFunc(cfg.Cron, func() {
		// 检查上下文是否已取消
		select {
		case <-taskCtx.Done():
			return
		default:
		}

		// 启动验证任务
		taskID, err := v.Start(taskCtx, cfg.Options)
		if err != nil {
			clog.Errorf(taskCtx, "定时验证任务启动失败: %v", err)
			return
		}

		// 等待任务完成
		progress := v.GetTaskProgress(taskID)
		for progress != nil && !progress.IsCompleted() {
			time.Sleep(time.Second)
			progress = v.GetTaskProgress(taskID)
		}

		if progress != nil && progress.Error != nil {
			clog.Errorf(taskCtx, "定时验证任务执行失败: %v", progress.Error)
			return
		}

		clog.Infof(taskCtx, "定时验证任务完成 [%s]，处理: %d，损坏: %d，修复: %d",
			taskID,
			progress.Total.Load(),
			progress.Invalid.Load(),
			progress.Fixed.Load())
	})
	if err != nil {
		cancel()
		return fmt.Errorf("添加定时任务失败: %w", err)
	}

	v.scheduler.Start()
	clog.Infof(ctx, "定时任务调度器已启动")
	return nil
}

// Close 关闭验证器
func (v *ComicVerifier) Close() error {
	// 停止定时任务
	if v.scheduler != nil {
		v.scheduler.Stop()
	}

	// 取消所有任务
	v.tasks.Range(func(key, value interface{}) bool {
		if cancel, ok := value.(context.CancelFunc); ok {
			cancel()
		}
		return true
	})

	// 等待工作池清空
	if v.pool != nil {
		v.pool.Release()
	}

	return nil
}

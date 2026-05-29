// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cocomhub/cocom/pkg/errwrap"
	"github.com/cocomhub/cocom/pkg/imaging"
	"github.com/panjf2000/ants/v2"
	"github.com/robfig/cron/v3"
	"github.com/rs/xid"
	"github.com/spf13/viper"
	"go.uber.org/atomic"
)

// VerifyOptions 验证选项
type VerifyOptions struct {
	ComicFilter
	AutoFix     bool  `json:"autoFix"`     // 是否自动修复
	GenDownList bool  `json:"genDownList"` // 是否生成下载列表
	MaxWorkers  int32 `json:"maxWorkers"`  // 最大并发数
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

	mu         sync.RWMutex
	running    []string
	waitFixing []string
	fixing     []string
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
	GenDownList   bool           `json:"genDownList"`   // 生成下载列表
	RetryInterval time.Duration  `json:"retryInterval"` // 重试间隔
	Cron          string         `json:"cron"`          // cron表达式
	Options       *VerifyOptions `json:"options"`       // 验证选项
	Active        bool           `json:"active"`        // 是否激活
	MaxRetry      int            `json:"maxRetry"`      // 最大重试次数
	RetryWait     time.Duration  `json:"retryWait"`     // 重试等待时间
}

// MarshalJSON 自定义JSON序列化
func (p *VerifyProgress) MarshalJSON() ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	type Alias VerifyProgress
	waitFixing := p.waitFixing
	if len(waitFixing) > 10 {
		waitFixing = make([]string, 0, 10)
		waitFixing = append(waitFixing, p.waitFixing[:8]...)
		waitFixing = append(waitFixing,
			fmt.Sprintf("...隐藏%d个元素...", len(p.waitFixing)-9),
			p.waitFixing[len(p.waitFixing)-1])
	}
	return json.Marshal(&struct {
		Total      int32        `json:"total"`
		Current    int32        `json:"current"`
		Invalid    int32        `json:"invalid"`
		Fixed      int32        `json:"fixed"`
		Running    []string     `json:"running"`
		WaitFixing []string     `json:"waitFixing"`
		Fixing     []string     `json:"fixing"`
		Status     VerifyStatus `json:"status"`
		*Alias
	}{
		Total:      p.Total.Load(),
		Current:    p.Current.Load(),
		Invalid:    p.Invalid.Load(),
		Fixed:      p.Fixed.Load(),
		Running:    p.running,
		WaitFixing: waitFixing,
		Fixing:     p.fixing,
		Status:     p.Status.Load().(VerifyStatus),
		Alias:      (*Alias)(p),
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

func (p *VerifyProgress) WaitFixing() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.waitFixing
}

func (p *VerifyProgress) Fixing() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.fixing
}

func (p *VerifyProgress) Start(id string) {
	p.mu.Lock()
	p.running = append(p.running, id)
	p.mu.Unlock()
}

func (p *VerifyProgress) End(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, v := range p.running {
		if v == id {
			p.running = append(p.running[:i], p.running[i+1:]...)
			return
		}
	}
	for i, v := range p.waitFixing {
		if v == id {
			p.waitFixing = append(p.waitFixing[:i], p.waitFixing[i+1:]...)
			return
		}
	}
	for i, v := range p.fixing {
		if v == id {
			p.fixing = append(p.fixing[:i], p.fixing[i+1:]...)
			return
		}
	}
}

func (p *VerifyProgress) WaitFix(id string) {
	p.mu.Lock()
	for i, v := range p.running {
		if v == id {
			p.running = append(p.running[:i], p.running[i+1:]...)
			break
		}
	}
	p.waitFixing = append(p.waitFixing, id)
	p.mu.Unlock()
}

func (p *VerifyProgress) Fix(id string) {
	p.mu.Lock()
	for i, v := range p.waitFixing {
		if v == id {
			p.waitFixing = append(p.waitFixing[:i], p.waitFixing[i+1:]...)
			break
		}
	}
	p.fixing = append(p.fixing, id)
	p.mu.Unlock()
}

// SetError 设置错误信息
func (p *VerifyProgress) SetError(err any) {
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

type Downloader interface {
	Download(ctx context.Context, url, path string) error
}

// ComicVerifier 漫画验证器
type ComicVerifier struct {
	ctx           context.Context
	cancel        context.CancelFunc
	storage       Storage
	downloader    Downloader
	metrics       *MetricsCollector
	verifyPool    *ants.Pool
	fixPool       *ants.Pool
	fixFnCh       chan func()
	fixWorkerWG   sync.WaitGroup
	fixWorkerOnce sync.Once
	poolMu        sync.Mutex
	tasks         sync.Map
	scheduler     *cron.Cron
	progressMu    sync.RWMutex
	progress      map[string]*VerifyProgress
}

// NewComicVerifier 创建漫画验证器
func NewComicVerifier(ctx context.Context, storage Storage) (*ComicVerifier, error) {
	ctx, cancel := context.WithCancel(ctx)

	verifyPoolSize := runtime.NumCPU()
	fixPoolSize := 2 * runtime.NumCPU()
	slog.InfoContext(ctx, "创建漫画验证器工作池", slog.Int("verifyPoolSize", verifyPoolSize), slog.Int("fixPoolSize", fixPoolSize))

	// 创建工作池
	verifyPool, err := ants.NewPool(
		verifyPoolSize,
		ants.WithPreAlloc(true),
		ants.WithPanicHandler(func(i any) {
			slog.ErrorContext(ctx, "Panic in worker", slog.Any("err", i))
		}),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create worker pool failed: %w", err)
	}

	fixPool, err := ants.NewPool(
		fixPoolSize,
		ants.WithPreAlloc(true),
		ants.WithPanicHandler(func(i any) {
			slog.ErrorContext(ctx, "Panic in worker", slog.Any("err", i))
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
		downloader: NewWgetDownloader(),
		metrics:    NewMetricsCollector(),
		verifyPool: verifyPool,
		fixPool:    fixPool,
		fixFnCh:    make(chan func(), 1000*fixPoolSize),
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
	total, err := v.storage.FindTotal(ctx, &opts.ComicFilter)
	if err != nil {
		cancel()
		return "", fmt.Errorf("查找漫画总数失败: %w", err)
	}

	comicsChannel, err := v.storage.FindChannel(context.WithoutCancel(ctx), &opts.ComicFilter)
	if err != nil {
		cancel()
		return "", fmt.Errorf("查找漫画失败: %w", err)
	}

	progress.Total.Store(int32(total))

	v.progressMu.Lock()
	v.progress[taskID] = progress
	v.progressMu.Unlock()

	// 启动验证任务
	go v.runTask(taskCtx, task, comicsChannel, opts)

	v.fixWorkerOnce.Do(func() {
		v.fixWorkerWG.Go(func() {
			for fn := range v.fixFnCh {
				for {
					err := v.fixPool.Submit(fn)
					if err == nil {
						break
					}
					time.Sleep(1 * time.Second)
				}
			}
		})
	})

	return taskID, nil
}

// runTask 运行验证任务
func (v *ComicVerifier) runTask(ctx context.Context, task *VerifyTask, comicsChannel chan Comic, opts *VerifyOptions) {
	defer v.cleanupTask(task.ID)

	var wg sync.WaitGroup
	task.Progress.Status.Store(VerifyStatusRunning)
	for c := range comicsChannel {
		if task.Progress.Status.Load() == VerifyStatusCanceled {
			wg.Wait()
			return
		}

		wg.Add(1)
		err := v.verifyPool.Submit(func() {
			defer wg.Done()

			task.Progress.Start(c.GetID())
			result := v.verifyComic(ctx, c)
			if result.InvalidCount > 0 && opts.AutoFix {
				task.Progress.WaitFix(c.GetID())
				wg.Add(1)
				fn := func() {
					defer wg.Done()

					task.Progress.Fix(c.GetID())
					result.Valid = true
					result.Error = nil
					for _, img := range result.fixImages {
						if task.Progress.IsCompleted() {
							continue
						}

						fixErr := v.fixImage(ctx, &img)
						if fixErr != nil {
							slog.WarnContext(ctx, "修复异常图片失败",
								slog.String("taskID", task.ID),
								slog.String("comicID", result.ComicID),
								fmt.Sprintf("%s", img.Path),
								slog.String("url", img.URL),
								slog.String("err", fixErr.Error()))
							result.Valid = false
							result.Error = errors.Join(result.Error, fixErr)
							continue
						}
						slog.DebugContext(ctx, "修复异常图片成功",
							slog.String("taskID", task.ID),
							slog.String("comicID", result.ComicID),
							fmt.Sprintf("%s", img.Path),
							slog.String("url", img.URL))
						result.FixedCount++
					}

					if result.Valid {
						slog.DebugContext(ctx, "验证漫画结束",
							slog.String("taskID", task.ID),
							slog.String("comicID", result.ComicID),
							slog.String("id", result.ID))
					} else {
						slog.WarnContext(ctx, "验证漫画存在异常",
							slog.String("taskID", task.ID),
							slog.String("comicID", result.ComicID),
							slog.String("id", result.ID),
							slog.Int("fixedCount", int(result.FixedCount)),
							slog.Int("invalidCount", int(result.InvalidCount)))
					}

					task.Progress.End(c.GetID())

					task.Progress.Current.Inc()
					task.Progress.Invalid.Add(result.InvalidCount)
					task.Progress.Fixed.Add(result.FixedCount)

					c.SetVerifyResult(result)
					err := v.storage.Update(ctx, c)
					if err != nil {
						slog.ErrorContext(ctx, "更新验证结果失败",
							slog.String("taskID", task.ID),
							slog.String("comicID", result.ComicID),
							slog.String("id", result.ID),
							slog.String("err", err.Error()))
					}
				}
				select {
				case v.fixFnCh <- fn:
					return
				default:
				}
			} else if result.InvalidCount > 0 && opts.GenDownList {
				slog.InfoContext(ctx, "生成下载列表",
					slog.String("taskID", task.ID),
					slog.String("comicID", result.ComicID),
					slog.String("id", result.ID))
				var downList strings.Builder
				for _, img := range result.fixImages {
					downList.WriteString(fmt.Sprintf("%s\n", img.URL))
				}

				path := path.Join(viper.GetString("download.downloadDir"), "downList", c.GetID()+".txt")
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					slog.ErrorContext(ctx, "创建保存下载列表目录失败",
						slog.String("taskID", task.ID),
						slog.String("comicID", result.ComicID),
						slog.String("id", result.ID),
						slog.String("err", err.Error()))
				}
				err := os.WriteFile(path, []byte(downList.String()), 0o644)
				if err != nil {
					slog.ErrorContext(ctx, "保存下载列表失败",
						slog.String("taskID", task.ID),
						slog.String("comicID", result.ComicID),
						slog.String("id", result.ID),
						slog.String("err", err.Error()))
				}
				task.Progress.End(c.GetID())

				task.Progress.Current.Inc()
				task.Progress.Invalid.Add(result.InvalidCount)
				task.Progress.Fixed.Add(result.FixedCount)

				c.SetVerifyResult(result)
				return
			}

			if result.Valid {
				slog.DebugContext(ctx, "验证漫画结束",
					slog.String("taskID", task.ID),
					slog.String("comicID", result.ComicID),
					slog.String("id", result.ID))
			} else {
				slog.WarnContext(ctx, "验证漫画存在异常",
					slog.String("taskID", task.ID),
					slog.String("comicID", result.ComicID),
					slog.String("id", result.ID),
					slog.Int("invalidCount", int(result.InvalidCount)))
			}

			task.Progress.End(c.GetID())

			task.Progress.Current.Inc()
			task.Progress.Invalid.Add(result.InvalidCount)
			task.Progress.Fixed.Add(result.FixedCount)

			c.SetVerifyResult(result)
			err := v.storage.Update(ctx, c)
			if err != nil {
				slog.ErrorContext(ctx, "更新验证结果失败",
					slog.String("taskID", task.ID),
					slog.String("comicID", result.ComicID),
					slog.String("id", result.ID),
					slog.String("err", err.Error()))
			}
		})
		if err != nil {
			slog.ErrorContext(ctx, "提交验证任务失败",
				slog.String("taskID", task.ID),
				slog.String("err", err.Error()))
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
func (v *ComicVerifier) verifyComic(ctx context.Context, comic Comic) *VerifyResult {
	result := &VerifyResult{
		ID:        xid.New().String(),
		ComicID:   comic.GetID(),
		Valid:     true,
		Timestamp: time.Now(),
	}

	// 验证所有图片
	for _, img := range comic.GetImages() {
		imgResult := v.verifyImage(ctx, &img)
		if imgResult.Invalid {
			slog.WarnContext(ctx, "验证图片异常",
				slog.String("id", result.ID),
				slog.String("comicID", result.ComicID),
				slog.String("path", img.Path),
				slog.String("errmsg", imgResult.Error.Error()))
			if errors.Is(imgResult.Error, errwrap.ErrImageSubsampling) {
				result.InvalidSubsamplingCount++
			} else {
				result.fixImages = append(result.fixImages, img)
				result.Error = errors.Join(result.Error, imgResult.Error)
				result.Valid = false
				result.InvalidCount++
			}
		}
	}
	return result
}

// verifyImage 验证单张图片
func (v *ComicVerifier) verifyImage(ctx context.Context, img *Image) *VerifyImageResult {
	result := &VerifyImageResult{
		Path: img.Path,
	}

	// 检查文件完整性
	if info, err := imaging.VerifyImage(ctx, img.Path); err != nil {
		result.Invalid = true
		if errors.Is(err, os.ErrNotExist) {
			result.Error = fmt.Errorf("文件不存在")
		} else if errors.Is(err, os.ErrPermission) {
			result.Error = fmt.Errorf("文件权限不足")
		} else if errors.Is(err, errwrap.ErrImageSubsampling) {
			result.Error = fmt.Errorf("图片子采样比例错误: %w", err)
		} else {
			result.Error = fmt.Errorf("解码图片失败: %w", err)
		}
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
		if errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(err, context.Canceled) {
			slog.WarnContext(ctx, "下载图片超时，重试下载",
				slog.String("path", img.Path),
				slog.String("url", img.URL))
			return v.fixImage(ctx, img)
		}
		return err
	}

	imgResult := v.verifyImage(ctx, &fixImg)
	if imgResult.Invalid && !errors.Is(imgResult.Error, errwrap.ErrImageSubsampling) {
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
	slog.InfoContext(ctx, "任务已取消", slog.String("taskID", taskID))
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
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].GetProgress().StartTime.Before(tasks[j].GetProgress().StartTime)
	})
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
		slog.InfoContext(ctx, "定时任务未激活，跳过启动")
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

	slog.InfoContext(ctx, "启动定时任务，执行规则", slog.String("cron", cfg.Cron))

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
			slog.ErrorContext(taskCtx, "定时验证任务启动失败", slog.String("taskID", taskID), slog.String("errmsg", err.Error()))
			return
		}

		// 等待任务完成
		progress := v.GetTaskProgress(taskID)
		for progress != nil && !progress.IsCompleted() {
			time.Sleep(time.Second)
			progress = v.GetTaskProgress(taskID)
		}

		if progress != nil && progress.Error != nil {
			slog.ErrorContext(taskCtx, "定时验证任务执行失败", slog.String("taskID", taskID), slog.String("errmsg", progress.Error.Error()))
			return
		}

		slog.InfoContext(taskCtx, "定时验证任务完成",
			slog.String("taskID", taskID),
			slog.Int64("total", int64(progress.Total.Load())),
			slog.Int64("invalid", int64(progress.Invalid.Load())),
			slog.Int64("fixed", int64(progress.Fixed.Load())))
	})
	if err != nil {
		cancel()
		return fmt.Errorf("添加定时任务失败: %w", err)
	}

	v.scheduler.Start()
	slog.InfoContext(ctx, "定时任务调度器已启动")
	return nil
}

// Close 关闭验证器
func (v *ComicVerifier) Close() error {
	// 停止定时任务
	if v.scheduler != nil {
		v.scheduler.Stop()
	}

	// 取消所有任务
	v.tasks.Range(func(key, value any) bool {
		if cancel, ok := value.(context.CancelFunc); ok {
			cancel()
		}
		return true
	})

	// 等待工作池清空
	if v.verifyPool != nil {
		v.verifyPool.Release()
	}
	if v.fixPool != nil {
		v.fixPool.Release()
	}

	// 关闭 fix worker 通道并等待 worker 退出
	if v.fixFnCh != nil {
		close(v.fixFnCh)
	}
	v.fixWorkerWG.Wait()

	return nil
}

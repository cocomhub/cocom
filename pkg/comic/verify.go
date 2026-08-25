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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/pkg/errwrap"
	"github.com/cocomhub/cocom/pkg/imaging"
	"github.com/panjf2000/ants/v2"
	"github.com/robfig/cron/v3"
	"github.com/rs/xid"
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
	messages  []string

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
	messages := make([]string, len(p.messages))
	copy(messages, p.messages)
	return json.Marshal(&struct {
		Total      int32        `json:"total"`
		Current    int32        `json:"current"`
		Invalid    int32        `json:"invalid"`
		Fixed      int32        `json:"fixed"`
		Running    []string     `json:"running"`
		WaitFixing []string     `json:"waitFixing"`
		Fixing     []string     `json:"fixing"`
		Status     VerifyStatus `json:"status"`
		Messages   []string     `json:"messages"`
		*Alias
	}{
		Total:      p.Total.Load(),
		Current:    p.Current.Load(),
		Invalid:    p.Invalid.Load(),
		Fixed:      p.Fixed.Load(),
		Running:    p.running,
		WaitFixing: waitFixing,
		Fixing:     p.fixing,
		Status:     p.Status.Load().(VerifyStatus), //nolint:errcheck
		Messages:   messages,
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

// SetMessage 设置进度消息
func (p *VerifyProgress) SetMessage(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.messages = append(p.messages, msg)
}

// GetMessages 获取所有进度消息
func (p *VerifyProgress) GetMessages() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make([]string, len(p.messages))
	copy(result, p.messages)
	return result
}

// IsCompleted 检查是否完成
func (p *VerifyProgress) IsCompleted() bool {
	status := p.GetStatus()
	return status == VerifyStatusCompleted || status == VerifyStatusError || status == VerifyStatusCanceled
}

// GetStatus 获取检查状态
func (p *VerifyProgress) GetStatus() VerifyStatus {
	return p.Status.Load().(VerifyStatus) //nolint:errcheck
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
	tasks         sync.Map
	scheduler     *cron.Cron
	progressMu    sync.RWMutex
	progress      map[string]*VerifyProgress
	downloadDir   string
	closeOnce     sync.Once
	fixClosed     chan struct{} // 关闭信号：close 时关闭，fixFnCh 发送侧据此退出，避免 send-on-closed panic
}

// NewComicVerifier 创建漫画验证器
func NewComicVerifier(ctx context.Context, storage Storage, downloadDir string) (*ComicVerifier, error) {
	ctx, cancel := context.WithCancel(ctx)

	// 死键接线（铁律 2）：verifyPoolSize/任务缓冲区大小读取 config.Get().Comic.Verify.*
	// （默认 10/100，与 manager.go SetDefault 一致）。pkg/comic 通过本文件顶部的
	// 懒加载 cell 读取一次，避免在测试（未初始化 config）场景抛错；生产路径
	// rootcli+config.Init 先跑，因此读取到的即为管理默认值。--workers 覆盖语义不变：
	// Start 里 opts.MaxWorkers > 0 时 Tune 到 CLI 值。
	verifyPoolSize := config.Get().Comic.Verify.Concurrent
	if verifyPoolSize <= 0 {
		verifyPoolSize = 10
	}
	taskBufferSize := config.Get().Comic.Verify.TaskBufferSize
	if taskBufferSize <= 0 {
		taskBufferSize = 100
	}
	fixPoolSize := 2 * verifyPoolSize
	slog.InfoContext(ctx, "创建漫画验证器工作池",
		slog.Int("verifyPoolSize", verifyPoolSize),
		slog.Int("taskBufferSize", taskBufferSize),
		slog.Int("fixPoolSize", fixPoolSize))

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
		fixFnCh:    make(chan func(), taskBufferSize),
		fixClosed:  make(chan struct{}),
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

	// 用 taskCtx（WithCancel(WithoutCancel(parent))）传 FindChannel：
	// HTTP 请求取消不会中断长验证任务，但任务显式取消（Close/CancelTask）会
	// 让 FindChannelHelper 的 producer 感知 ctx.Done 退出并关闭通道，避免 goroutine 泄漏。
	comicsChannel, err := v.storage.FindChannel(taskCtx, &opts.ComicFilter)
	if err != nil {
		cancel()
		return "", fmt.Errorf("查找漫画失败: %w", err)
	}

	progress.Total.Store(int32(total))

	v.progressMu.Lock()
	v.progress[taskID] = progress
	v.progressMu.Unlock()

	// 应用用户配置的最大并发数（CLI --workers 等）。
	// 已知限制：verifyPool 为共享池，多任务并发启动时 MaxWorkers 取最后一次生效。
	if opts.MaxWorkers > 0 {
		v.verifyPool.Tune(int(opts.MaxWorkers))
	}

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
					// 池已关闭（仅在异常路径触发；正常 Close 顺序保证释放 fixPool 在
					// fixWorkerWG.Wait 之后），不再自旋重试，避免无限空转。
					if errors.Is(err, ants.ErrPoolClosed) {
						slog.ErrorContext(v.ctx, "fix pool closed, drop fix task")
						return
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
	// 仅当尚未取消时置 Running：在锁内完成读-判-写，闭合与 CancelTask
	// （持写锁置 Canceled）之间的 TOCTOU——避免 Start 后立即取消被 Running 覆盖。
	v.progressMu.Lock()
	if task.Progress.GetStatus() != VerifyStatusCanceled {
		task.Progress.Status.Store(VerifyStatusRunning)
	}
	v.progressMu.Unlock()
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
								slog.String("path", img.Path),
								slog.String("url", img.URL),
								slog.String("err", fixErr.Error()))
							result.Valid = false
							result.Error = errors.Join(result.Error, fixErr)
							continue
						}
						slog.DebugContext(ctx, "修复异常图片成功",
							slog.String("taskID", task.ID),
							slog.String("comicID", result.ComicID),
							slog.String("path", img.Path),
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

					// 持锁写验证结果，避免在锁外通过活指针 SetVerifyResult（与 API 序列化竞争）。
					if err := v.storage.SaveVerifyResult(ctx, result); err != nil {
						slog.ErrorContext(ctx, "更新验证结果失败",
							slog.String("taskID", task.ID),
							slog.String("comicID", result.ComicID),
							slog.String("id", result.ID),
							slog.String("err", err.Error()))
					}
				}
				// 阻塞发送：fix worker 会持续消费 fixFnCh，不会真的死锁。
				// fixClosed 在 Close 关闭 fixFnCh 前先行关闭：若 Close 已触发，
				// 发送侧手动 wg.Done()（对齐上方 wg.Add(1)）后退出，既避免
				// send-on-closed panic，也不致 wg.Wait() 永久挂起。
				select {
				case v.fixFnCh <- fn:
				case <-v.fixClosed:
					wg.Done()
					return
				}
				return
			} else if result.InvalidCount > 0 && opts.GenDownList {
				slog.InfoContext(ctx, "生成下载列表",
					slog.String("taskID", task.ID),
					slog.String("comicID", result.ComicID),
					slog.String("id", result.ID))
				var downList strings.Builder
				for _, img := range result.fixImages {
					fmt.Fprintf(&downList, "%s\n", img.URL)
				}

				path := path.Join(v.downloadDir, "downList", c.GetID()+".txt")
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

				if err := v.storage.SaveVerifyResult(ctx, result); err != nil {
					slog.ErrorContext(ctx, "更新验证结果失败",
						slog.String("taskID", task.ID),
						slog.String("comicID", result.ComicID),
						slog.String("id", result.ID),
						slog.String("err", err.Error()))
				}
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

			if err := v.storage.SaveVerifyResult(ctx, result); err != nil {
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
	// 任务正常结束：置为完成态（R-A1 回归）。
	// 与 CancelTask 通过 progressMu 串行化，避免 lost-update：
	// CancelTask 在持 progressMu 时写 Canceled，这里同样持锁读-判-写，
	// 保证"取消"不被"完成"覆盖。
	v.progressMu.Lock()
	if task.Progress.GetStatus() != VerifyStatusCanceled {
		task.Progress.Status.Store(VerifyStatusCompleted)
	}
	v.progressMu.Unlock()
}

// cleanupTask 清理任务。
// 任务完成后延迟保留 progress 一段时间（默认 60s），使客户端能查询到最终结果；
// 取消/异常路径立即清理。
func (v *ComicVerifier) cleanupTask(taskID string) {
	v.tasks.Delete(taskID)

	go func() {
		time.Sleep(60 * time.Second)
		v.progressMu.Lock()
		delete(v.progress, taskID)
		v.progressMu.Unlock()
	}()
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
		if errors.Is(err, context.DeadlineExceeded) {
			// 超时仍可重试。
			slog.WarnContext(ctx, "下载图片超时，重试下载",
				slog.String("path", img.Path),
				slog.String("url", img.URL))
			return v.fixImage(ctx, img)
		}
		if errors.Is(err, context.Canceled) {
			// 任务取消不可重试：返回错误让 fixPool.Release() 能排空，
			// 避免无限递归把 Close 拖死。
			return err
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
		return nil, fmt.Errorf("%w: %s", ErrTaskNotFound, taskID)
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
		return fmt.Errorf("%w: %s", ErrTaskNotFound, taskID)
	}

	if progress.IsCompleted() {
		return fmt.Errorf("任务已完成: %s", taskID)
	}

	progress.Status.Store(VerifyStatusCanceled)

	// 取消底层 taskCtx：FindChannel 生产者/修复任务 rely 在 ctx.Done 上退出。
	// task 已从 tasks 映射删除时（cleanupTask 先行）无法取消，但状态已置 canceled，
	// runTask 的 wg.Wait 也会因 channel 关闭而结束，剩余 goroutine 会随 ctx 自然回收。
	if value, ok := v.tasks.Load(taskID); ok {
		if task, ok := value.(*VerifyTask); ok && task.Cancel != nil {
			task.Cancel()
		}
	}
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

		// 等待任务完成。轮询周期 1s；若任务已被取消则退出，
		// 并带截止时间保护（防止 runTask 异常导致永久轮询）。
		progress := v.GetTaskProgress(taskID)
		waitUntil := time.Now().Add(24 * time.Hour)
		for progress != nil && !progress.IsCompleted() && time.Now().Before(waitUntil) {
			select {
			case <-taskCtx.Done():
				slog.WarnContext(taskCtx, "定时验证任务已取消，停止等待", slog.String("taskID", taskID))
				return
			case <-time.After(time.Second):
			}
			progress = v.GetTaskProgress(taskID)
		}
		if progress == nil {
			slog.WarnContext(taskCtx, "定时验证任务进度已过期，停止等待", slog.String("taskID", taskID))
			return
		}
		if !progress.IsCompleted() {
			// 到达截止时间但任务仍在运行（超长任务）：不报"完成"，避免误导。
			slog.WarnContext(taskCtx, "定时验证任务仍在运行，等待超时", slog.String("taskID", taskID))
			return
		}

		if progress.Error != nil {
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

// Close 关闭验证器。幂等：可被多次调用（服务关闭 + 测试 teardown）。
func (v *ComicVerifier) Close() error {
	v.closeOnce.Do(v.close)
	return nil
}

// close 是 Close 的实际实现。
//
// 关闭顺序（修复 fixPool.Release 后 fix worker 无限自旋导致死锁）：
//  1. 停止定时任务；
//  2. 取消所有任务（*VerifyTask.Cancel / context.CancelFunc），使 taskCtx 可取消，
//     进而让 FindChannelHelper producer 退出、fixImage 取消返回；
//  3. 停 verify 池：此后不再有 verify worker 向 fixFnCh 发送；
//  4. 关闭 fixFnCh 并等待 fix worker 排空退出；
//  5. 最后释放 fix 池：等待 in-flight 修复任务完成。
func (v *ComicVerifier) close() {
	// 停止定时任务
	if v.scheduler != nil {
		v.scheduler.Stop()
	}

	// 取消所有任务：Start 注册的是 *VerifyTask（含 Cancel 字段），
	// StartSchedule 注册的是 context.CancelFunc，两种都要取消。
	v.tasks.Range(func(key, value any) bool {
		switch t := value.(type) {
		case context.CancelFunc:
			t()
		case *VerifyTask:
			if t.Cancel != nil {
				t.Cancel()
			}
		}
		return true
	})

	// 等待工作池清空
	if v.verifyPool != nil {
		v.verifyPool.Release()
	}

	// 关闭 fix worker 通道并等待 worker 退出
	// 先关 fixClosed 通知发送侧退出，再关 fixFnCh，避免 send-on-closed panic。
	if v.fixClosed != nil {
		close(v.fixClosed)
	}
	if v.fixFnCh != nil {
		close(v.fixFnCh)
	}
	v.fixWorkerWG.Wait()

	if v.fixPool != nil {
		v.fixPool.Release()
	}
}

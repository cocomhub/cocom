// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package download

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"sync"
	"time"

	"github.com/cocomhub/cocom/pkg/conv"
	"github.com/cocomhub/cocom/pkg/util"

	"github.com/cavaliergopher/grab/v3"
)

var (
	mu                sync.Mutex
	once              sync.Once
	startErr          error
	DefaultDownloader = NewDownloader(NewConfig())
)

// SetDefault 已迁移到 internal/config/manager.go setDefaultsOn()

func NewInitConfig(cfg Config) *DownloaderConfig {
	return NewConfig().
		SetDownloadDir(cfg.DownloadDir).
		SetMaxRunning(cfg.MaxRunning).
		SetEnableProxy(cfg.EnableProxy).
		SetProxyURL(cfg.ProxyURL)
}

func Init(cfg Config) {
	ReplaceDownloader(NewDownloader(NewInitConfig(cfg)))
}

func ReplaceDownloader(newDownloader *Downloader) func() {
	mu.Lock()
	defer mu.Unlock()
	oldDownloader := DefaultDownloader
	DefaultDownloader = newDownloader
	return func() {
		mu.Lock()
		defer mu.Unlock()
		DefaultDownloader = oldDownloader
	}
}

func Start() error {
	mu.Lock()
	defer mu.Unlock()
	return DefaultDownloader.Start()
}

func Close() error {
	mu.Lock()
	defer mu.Unlock()
	return DefaultDownloader.Close()
}

func Wait() {
	mu.Lock()
	defer mu.Unlock()
	DefaultDownloader.Wait()
}

func DoBatch(workers int, tasks ...*Task) (chan *TaskResult, error) {
	mu.Lock()
	defer mu.Unlock()
	once.Do(func() {
		// 记录启动错误而不是 panic：panic 会把 sync.Once 永久烧毁，
		// 后续所有 DoBatch 调用都将无法启动 worker。
		startErr = DefaultDownloader.Start()
	})
	if startErr != nil {
		return nil, startErr
	}
	return DefaultDownloader.DoBatch(workers, tasks...)
}

type DownloaderConfig struct {
	DownloadDir string `json:"downloadDir"`
	MaxRunning  int    `json:"maxRunning"`
	EnableProxy bool   `json:"enableProxy"`
	ProxyURL    string `json:"proxyURL"`
}

func NewConfig() *DownloaderConfig {
	cfg := &DownloaderConfig{}
	return cfg
}

func (cfg *DownloaderConfig) SetDownloadDir(dir string) *DownloaderConfig {
	cfg.DownloadDir = dir
	return cfg
}

func (cfg *DownloaderConfig) SetMaxRunning(maxRunning int) *DownloaderConfig {
	cfg.MaxRunning = maxRunning
	return cfg
}

func (cfg *DownloaderConfig) SetEnableProxy(enable bool) *DownloaderConfig {
	cfg.EnableProxy = enable
	return cfg
}

func (cfg *DownloaderConfig) SetProxyURL(url string) *DownloaderConfig {
	cfg.ProxyURL = url
	return cfg
}

func (cfg *DownloaderConfig) Init() *DownloaderConfig {
	if len(cfg.DownloadDir) == 0 {
		cfg.DownloadDir = "./Downloads"
	}
	if cfg.MaxRunning == 0 {
		cfg.MaxRunning = 3
	}
	return cfg
}

type Downloader struct {
	cfg    *DownloaderConfig
	client *grab.Client
	logger *slog.Logger

	m      sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
	taskCh chan *Task //nolint:unused
	reqCh  chan *grab.Request
	respCh chan *grab.Response
	// sem 是全局并发信号量（容量 = MaxRunning）。
	// DoBatch worker 直连 d.client.Do 后，由 sem 统一封顶多本漫画并发下载的总并发。
	sem chan struct{}

	wg *sync.WaitGroup
}

func NewDownloader(cfg *DownloaderConfig) *Downloader {
	if cfg == nil {
		cfg = NewConfig()
	}
	cfg.Init()

	ctx, cancel := context.WithCancel(context.Background())
	d := &Downloader{
		cfg:    cfg,
		client: grab.NewClient(),
		ctx:    ctx,
		cancel: cancel,
		logger: slog.Default().With(slog.String("module", "downloader")),
		reqCh:  make(chan *grab.Request),
		respCh: make(chan *grab.Response),
		sem:    make(chan struct{}, cfg.MaxRunning),
		wg:     &sync.WaitGroup{},
	}

	if cfg.EnableProxy {
		u, err := url.Parse(cfg.ProxyURL)
		if err == nil {
			d.client.HTTPClient = &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(u),
				},
			}
		}
	}

	return d
}

func (d *Downloader) Context() context.Context {
	if d.ctx == nil {
		d.ctx = context.Background()
	}
	return d.ctx
}

func (d *Downloader) SetContext(ctx context.Context) {
	if ctx == nil {
		panic(any("nil context"))
	}
	d.ctx = ctx
}

func (d *Downloader) Start() error {
	err := util.CreateDirIfNotExist(d.cfg.DownloadDir)
	if err != nil {
		return err
	}

	d.wg.Add(d.cfg.MaxRunning)
	for i := 0; i < d.cfg.MaxRunning; i++ {
		go func(no int) {
			defer d.wg.Done()
			for {
				select {
				case <-d.ctx.Done():
					d.logger.InfoContext(d.Context(), "Downloader stop handle new task", slog.Int("worker", no))
					return
				case req := <-d.reqCh:
					req = req.WithContext(d.Context())
					d.logger.DebugContext(d.Context(), "download start", slog.String("url", req.URL().String()), slog.String("filename", req.Filename))
					resp := d.client.Do(req)
					d.respCh <- resp
					<-resp.Done
					d.logger.DebugContext(d.Context(), "download end", slog.String("url", req.URL().String()), slog.String("filename", req.Filename), slog.Any("err", resp.Err()))
				}
			}
		}(i)
	}

	d.logger.InfoContext(d.Context(), "Downloader start")
	return nil
}

func (d *Downloader) Close() error {
	d.m.Lock()
	defer d.m.Unlock()

	select {
	case <-d.ctx.Done():
		return nil
	default:
		d.cancel()
		return nil
	}
}

func (d *Downloader) Wait() {
	timer := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-d.ctx.Done():
			d.wg.Wait()
			return
		case <-timer.C:
		}
	}
}

func (d *Downloader) DoBatch(workers int, tasks ...*Task) (chan *TaskResult, error) {
	if workers < 1 {
		workers = len(tasks)
	}
	if workers > len(tasks) {
		workers = len(tasks)
	}

	taskCh := make(chan *Task, len(tasks))
	resultCh := make(chan *TaskResult, len(tasks))
	wg := sync.WaitGroup{}
	// worker 内层循环逐个消费 taskCh，保证 len(tasks) > workers 时全部任务都被处理
	//（旧实现每个 worker 只消费 1 个 task，超出的任务被静默丢弃）。
	for i := 0; i < workers; i++ {
		wg.Go(func() {
			for task := range taskCh {
				req, err := grab.NewRequest(path.Join(d.cfg.DownloadDir, task.Dir, task.Name), task.Url)
				if err != nil {
					d.logger.ErrorContext(d.Context(), "new request failed", slog.String("task", conv.JSON(task)), slog.Any("err", err))
					resultCh <- &TaskResult{Task: task, Err: err}
					continue
				}
				// 全局信号量：多本漫画并发下载（每本各调 DoBatch）时，
				// 总并发仍被 MaxRunning 封顶，避免压垮网络/后端。
				select {
				case d.sem <- struct{}{}:
				case <-d.ctx.Done():
					// 取消时未获槽位的任务以失败结果落盘并退出 worker。
					resultCh <- &TaskResult{Task: task, Err: d.ctx.Err()}
					return
				}
				req = req.WithContext(d.Context())
				resp := d.client.Do(req)
				// 等待传输结束；ctx 取消会随 req 传播，resp.Done 必然关闭。
				select {
				case <-resp.Done:
				case <-d.ctx.Done():
				}
				<-d.sem
				resultCh <- &TaskResult{Task: task, Response: resp}
			}
		})
	}

	// 任务投递：ctx 取消时停止投递，让 worker 随 taskCh 关闭退出。
	go func() {
		for i, task := range tasks {
			select {
			case taskCh <- task:
				// 只记录值字段，避免 conv.JSON(task) 读取 *task.Status 与调用方
				// 写 *result.Task.Status 产生数据竞争。
				d.logger.DebugContext(d.Context(), "input task",
					slog.Int("index", i),
					slog.String("url", task.Url),
					slog.String("dir", task.Dir),
					slog.String("name", task.Name))
			case <-d.ctx.Done():
				close(taskCh)
				wg.Wait()
				close(resultCh)
				return
			}
		}

		close(taskCh)
		wg.Wait()
		close(resultCh)
	}()
	return resultCh, nil
}

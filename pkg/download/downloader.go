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
	"github.com/spf13/viper"
)

var (
	mu                sync.Mutex
	once              sync.Once
	DefaultDownloader = NewDownloader(NewConfig())
)

// SetDefault 已迁移到 internal/config/config.go setDefaults()

func NewInitConfig() *DownloaderConfig {
	return NewConfig().
		SetDownloadDir(viper.GetString("download.downloadDir")).
		SetMaxRunning(viper.GetInt("download.maxRunning"))
}

func Init() {
	ReplaceDownloader(NewDownloader(NewInitConfig()))
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
		err := DefaultDownloader.Start()
		if err != nil {
			panic(any("DefaultDownloader start failed. " + err.Error()))
		}
	})
	return DefaultDownloader.DoBatch(workers, tasks...)
}

type DownloaderConfig struct {
	DownloadDir string `json:"downloadDir"`
	MaxRunning  int    `json:"maxRunning"`
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
	taskCh chan *Task
	reqCh  chan *grab.Request
	respCh chan *grab.Response

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
		wg:     &sync.WaitGroup{},
	}

	if viper.GetBool("http.enable_proxy") {
		u, err := url.Parse(viper.GetString("http.proxy"))
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

	taskCh := make(chan *Task, len(tasks))
	resultCh := make(chan *TaskResult, len(tasks))
	wg := sync.WaitGroup{}
	for i := 0; i < workers; i++ {
		wg.Go(func() {
			task, ok := <-taskCh
			if !ok {
				return
			}
			req, err := grab.NewRequest(path.Join(d.cfg.DownloadDir, task.Dir, task.Name), task.Url)
			if err != nil {
				d.logger.ErrorContext(d.Context(), "new request failed", slog.String("task", conv.JSON(task)), slog.Any("err", err))
				return
			}
			d.reqCh <- req

			resp := <-d.respCh
			resultCh <- &TaskResult{
				Task:     task,
				Response: resp,
			}
		})
	}

	// queue requests
	go func() {
		for i, task := range tasks {
			d.logger.DebugContext(d.Context(), "input task", slog.Int("index", i), slog.String("task", conv.JSON(task)))
			taskCh <- task
		}

		close(taskCh)
		wg.Wait()
		close(resultCh)
	}()
	return resultCh, nil
}

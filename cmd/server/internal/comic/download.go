// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/errs"
	"github.com/cocomhub/cocom/pkg/download"
	"github.com/cocomhub/cocom/pkg/errwrap"
	"github.com/cocomhub/cocom/pkg/mutex"
	"github.com/cocomhub/cocom/pkg/util"
)

var (
	maxDownloadSize atomic.Int32
	downloadSize    atomic.Int32
)

func init() {
	// 保持默认 5，避免 Init 前 ComicDownloadConnOver 误判连接已满（0 >= 0）。
	maxDownloadSize.Store(5)
}

// SetDefault 已迁移到 internal/config/manager.go setDefaultsOn()

func Init(ctx context.Context, maxSize int32) {
	maxDownloadSize.Store(maxSize)
}

func ComicDownloadConnOver() bool {
	return downloadSize.Load() >= maxDownloadSize.Load()
}

func CreateDownloadTaskWithLock(ctx context.Context, cid, maxConn, maxRetry int, force bool) (failed int, err error) {
	unlock, err := mutex.Lock(ctx, fmt.Sprintf("comic/%d", cid))
	if err != nil {
		slog.ErrorContext(ctx, "mutex lock failed", slog.String("err", err.Error()))
		return -1, err
	}
	defer unlock()

	taskFailed, err := CreateDownloadTask(ctx, cid, maxConn, maxRetry, force)
	if err != nil {
		slog.ErrorContext(ctx, "download comic task failed",
			slog.Int("cid", cid),
			slog.Int("failed", taskFailed),
			slog.String("err", err.Error()))
	}
	return taskFailed, err
}

func CreateDownloadTasks(ctx context.Context, cids []int, maxConn, maxRetry int, force bool) (int, error) {
	wg := sync.WaitGroup{}
	wg.Add(len(cids))
	errCh := make(chan error, len(cids))
	for _, cid := range cids {
		for ComicDownloadConnOver() {
			time.Sleep(1 * time.Second)
		}

		go func(cid int) {
			defer wg.Done()

			_, err := CreateDownloadTask(ctx, cid, maxConn, maxRetry, force)
			if err != nil {
				errCh <- err
			}
		}(cid)
	}

	wg.Wait()
	close(errCh)

	errWrap := errwrap.NewErrors()
	for err := range errCh {
		errWrap.Add(err)
	}
	return errWrap.Count(), errWrap.Err()
}

func CreateDownloadTask(ctx context.Context, cid, maxConn, maxRetry int, force bool) (failed int, err error) {
	if ComicDownloadConnOver() {
		return -1, errs.ErrComicDownloadConnOver
	}
	downloadSize.Add(1)
	defer downloadSize.Add(-1)

	if maxRetry == 0 {
		maxRetry = 1
	}

	for i := 0; i < maxRetry; i++ {
		failed, err = createDownloadTask(ctx, cid, maxConn, force)
		// 致命错误（failed==0 且 err!=nil，如 Get/DoBatch 失败）：无部分成功页面，
		// 直接返回不重试。failed>0 表示部分页面失败，进入重试循环。
		if err != nil && err != errs.ErrComicAlreadyDownloaded && failed == 0 {
			failed = -1
			return
		}
		if failed == 0 {
			return
		}
		slog.DebugContext(ctx, "CreateDownloadTask failed",
			slog.Int("failed", failed),
			slog.String("err", err.Error()),
			slog.Int("retry", i))
	}
	err = errs.ErrComicDownloadRetryOver
	return
}

func createDownloadTask(ctx context.Context, cid, maxConn int, force bool) (int, error) {
	info := api.ComicInfo{}
	err := GetComicInfo(ctx, cid, &info)
	if err != nil {
		return 0, err
	}
	slog.DebugContext(ctx, "CreateDownloadTask info",
		slog.Int("cid", cid),
		slog.Int("maxConn", maxConn),
		slog.Bool("force", force),
		slog.Any("info", info))

	if info.Status && !force {
		return 0, errs.ErrComicAlreadyDownloaded
	}
	var tasks []*download.Task
	domainId := api.GetDomainId()
	downloadDir := info.SaveDir()
	for i := range info.Images.Pages {
		page := &info.Images.Pages[i]
		if page.Status && !force {
			continue
		}

		name := info.Images.PageNameByIndex(i)
		url := fmt.Sprintf("https://i%d.nhentai.net/galleries/%s/%s", domainId, info.MediaId, name)
		tasks = append(tasks, &download.Task{
			Dir:    downloadDir,
			Name:   name,
			Url:    url,
			Status: &page.Status,
		})
		slog.DebugContext(ctx, "create download task",
			slog.Int("cid", cid),
			slog.String("dir", downloadDir),
			slog.String("name", name),
			slog.String("url", url))
	}

	resultCh, err := download.DoBatch(maxConn, tasks...)
	if err != nil {
		return 0, err
	}

	errWrap := errwrap.NewErrors()
	for result := range resultCh {
		// Err 表示传输前失败（grab.NewRequest 构造失败），Response 为 nil。
		if result.Err != nil {
			slog.ErrorContext(ctx, "comic download task failed",
				slog.Int("cid", cid),
				slog.String("dir", result.Task.Dir),
				slog.String("name", result.Task.Name),
				slog.String("url", result.Task.Url),
				slog.String("err", result.Err.Error()))
			errWrap.Add(fmt.Errorf("cid[%d] dir[%s] name[%s] url[%s] download failed. errmsg: %s",
				cid, result.Task.Dir, result.Task.Name, result.Task.Url, result.Err))
			continue
		}
		if result.Response.Err() != nil {
			slog.ErrorContext(ctx, "comic download task failed",
				slog.Int("cid", cid),
				slog.String("dir", result.Task.Dir),
				slog.String("name", result.Task.Name),
				slog.String("url", result.Task.Url),
				slog.String("err", result.Response.Err().Error()))
			errWrap.Add(fmt.Errorf("cid[%d] dir[%s] name[%s] url[%s] download failed. errmsg: %s",
				cid, result.Task.Dir, result.Task.Name, result.Task.Url, result.Response.Err()))
			continue
		}
		if result.Task.Status != nil {
			*result.Task.Status = true
		}
	}

	info.CheckStatus()
	info2, err := util.ToMap(info)
	if err != nil {
		errWrap.Add(fmt.Errorf("get downloaded comic info failed. errmsg: %s", err))
	} else {
		err = UpdateComicInfo(ctx, cid, info2)
		if err != nil {
			errWrap.Add(fmt.Errorf("update downloaded comic info failed. errmsg: %s", err))
		}
	}
	return errWrap.Count(), errWrap.Err()
}

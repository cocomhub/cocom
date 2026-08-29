// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/pkg/conv"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	"github.com/cocomhub/cocom/pkg/mutex"
)

// maxDownloadConn 单次下载任务的最大并发连接数上限。
// 客户端传 max_conn 无校验时可能触发 DoBatch 起海量 goroutine 导致 OOM，此处统一约束。
const maxDownloadConn = 10

// BuildArchiveConfig 从全局配置构建 ArchiveConfig。
// 优先读规范键 cocom.archive.*，命中旧键 archive.* 时回退并告警。
func BuildArchiveConfig() comic.ArchiveConfig {
	cfg := config.Get()
	return comic.ArchiveConfig{
		Password:  config.ArchiveString(cfg.Cocom.Archive.Password, cfg.Archive.Password, "password"),
		CmdPath:   config.ArchiveString(cfg.Cocom.Archive.Cmd, cfg.Archive.Cmd, "cmd"),
		Replicate: config.ArchiveBool(cfg.Cocom.Archive.Replicate, cfg.Archive.Replicate, "replicate"),
	}
}

func SaveComicInfo(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	info := map[string]any{}
	err := json.NewDecoder(req.Body).Decode(&info)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "decode body failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("decode body failed. errmsg: %s", err))
		return
	}
	slog.DebugContext(ctx, "req info", slog.String("info", conv.JSON(info)))

	_, exist := info["cid"]
	if !exist {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "comic id not found failed")
		httpwrap.ResponseFail(ctx, w, "comic id not found failed")
		return
	}

	var cid int
	switch v := info["cid"].(type) {
	case float64:
		// float64 分支（JSON 数字默认解码）：math.Trunc 校验非整数/越界自动截断风险。
		// 非整数（如 1001.5）不能被安全映射为 int → 400。
		if v != math.Trunc(v) || v > math.MaxInt32 || v < math.MinInt32 {
			w.WriteHeader(http.StatusBadRequest)
			slog.ErrorContext(ctx, "invalid comic id (non-integer float)", slog.Float64("value", v))
			httpwrap.ResponseFail(ctx, w, "invalid comic id")
			return
		}
		cid = int(v)
		info["cid"] = cid // 回写归一化后的 int，避免 $set 写入 BSON double 造成类型漂移
	case string:
		cid, err = strconv.Atoi(v)
		info["cid"] = cid
	default:
		err = fmt.Errorf("unknown type: cid[%v]", v)
	}
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "request parse cid failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request parse cid failed. errmsg: %s", err))
		return
	}
	if cid <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "invalid comic id", slog.Int("cid", cid))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("invalid comic id: %d", cid))
		return
	}

	unlock, err := mutex.Lock(ctx, fmt.Sprintf("comic/%d", cid))
	if err != nil {
		w.WriteHeader(http.StatusTooManyRequests)
		slog.ErrorContext(ctx, "mutex lock failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("mutex lock failed. errmsg: %s", err))
		return
	}
	defer unlock()

	err = comic.UpdateComicInfo(ctx, cid, info)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.ErrorContext(ctx, "update comic info failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("update comic info failed. errmsg: %s", err))
		return
	}

	httpwrap.ResponseSucc(ctx, w, "")
}

func GetComicInfo(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	err := req.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "request parse form failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request parse form failed. errmsg: %s", err))
		return
	}

	cid, err := strconv.Atoi(cmp.Or(req.FormValue("cid"), req.FormValue("id")))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "request parse cid failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request parse cid failed. errmsg: %s", err))
		return
	}
	// GetComicInfo handler cid<=0 校验：负 cid 或 0 无意义（Mongo filter 返回空/全部），
	// 与 SaveComicInfo / GetComicPages 的校验对齐，避免 cache key 为负导致读串数据。
	if cid <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "invalid comic id", slog.Int("cid", cid))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("invalid comic id: %d", cid))
		return
	}

	unlock, err := mutex.Lock(ctx, fmt.Sprintf("comic/%d", cid))
	if err != nil {
		w.WriteHeader(http.StatusTooManyRequests)
		slog.ErrorContext(ctx, "mutex lock failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("mutex lock failed. errmsg: %s", err))
		return
	}
	defer unlock()

	info := map[string]any{}
	err = comic.GetComicInfo(ctx, cid, &info)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.ErrorContext(ctx, "get comic info failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("get comic info failed. errmsg: %s", err))
		return
	}

	httpwrap.ResponseSucc(ctx, w, info)
}

func DownloadComic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req := api.DownloadComicByIDRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "decode body failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("decode body failed. errmsg: %s", err))
		return
	}
	slog.DebugContext(ctx, "req", slog.String("req", conv.JSON(req)))

	if req.MaxConn <= 0 || req.MaxConn > maxDownloadConn {
		req.MaxConn = maxDownloadConn
	}

	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Second)
		defer cancel()
	}

	if comic.ComicDownloadConnOver() {
		slog.WarnContext(ctx, "download comic conn over", slog.Int("cid", req.Cid))
		httpwrap.Response(ctx, w, 1001, "download comic conn over", "")
		return
	}

	if !req.IsSync {
		// TODO(async 锁语义): 与 RestoreComic 相同，DownloadComic 异步分支的 mutex.Lock 不跨
		// goroutine，同一 cid 并发下载存在竞态。完整修复需把锁语义改为任务队列，改动面大，
		// 暂保留现状（见 RestoreComic 的 TODO）。
		go func() {
			bgCtx := context.WithoutCancel(ctx)
			taskFailed, dlErr := comic.CreateDownloadTaskWithLock(bgCtx, req.Cid, req.MaxConn, req.MaxRetry, req.Force)
			if dlErr != nil {
				slog.ErrorContext(bgCtx, "download comic task failed", slog.Int("taskFailed", taskFailed), slog.String("errmsg", dlErr.Error()))
				return
			}
		}()
		httpwrap.Response(ctx, w, 1000, "async download task", "")
		return
	}

	taskFailed, err := comic.CreateDownloadTaskWithLock(ctx, req.Cid, req.MaxConn, req.MaxRetry, req.Force)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "download comic task failed", slog.Int("taskFailed", taskFailed), slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("download comic task failed[%d]. errmsg: %s", taskFailed, err))
		return
	}
	httpwrap.ResponseSucc(ctx, w, "")
}

// RestoreComic 恢复漫画
func RestoreComic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := api.RestoreComicByIDRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "decode body failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("decode body failed. errmsg: %s", err))
		return
	}
	slog.DebugContext(ctx, "req", slog.String("req", conv.JSON(req)))

	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Second)
		defer cancel()
	}

	unlock, err := mutex.Lock(ctx, fmt.Sprintf("comic/%d", req.Cid))
	if err != nil {
		w.WriteHeader(http.StatusTooManyRequests)
		slog.ErrorContext(ctx, "mutex lock failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("mutex lock failed. errmsg: %s", err))
		return
	}
	defer unlock()

	if !req.IsSync {
		// TODO(async 锁语义): RestoreComic 的 mutex.Lock 在切换 goroutine 前锁住后立即返回，
		// 异步分支放锁后才真正恢复（锁不跨 goroutine）。存在“返回后立即再次请求同一 cid”的
		// 竞态窗口。完整修复需要把锁语义改为 Async+Ctx 或内部任务队列，改动面大，暂不做（保留现状）。
		ac := BuildArchiveConfig()
		go func() {
			bgCtx := context.WithoutCancel(ctx)
			if err := comic.RestoreComicByID(bgCtx, req.Cid, ac); err != nil {
				slog.ErrorContext(bgCtx, "restore comic failed", slog.Int("cid", req.Cid), slog.String("errmsg", err.Error()))
			}
		}()
		httpwrap.Response(ctx, w, 1000, "async restore task", "")
		return
	}

	if err := comic.RestoreComicByID(ctx, req.Cid, BuildArchiveConfig()); err != nil {
		slog.ErrorContext(ctx, "restore comic failed", slog.Int("cid", req.Cid), slog.String("errmsg", err.Error()))
		// 未归档漫画显式报错：映射“漫画未归档”文案（404），其余内部错误 400
		if errors.Is(err, comic.ErrComicNotArchived) {
			w.WriteHeader(http.StatusNotFound)
			httpwrap.ResponseFail(ctx, w, "漫画未归档")
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("restore comic failed. errmsg: %s", err))
		return
	}
	httpwrap.ResponseSucc(ctx, w, "")
}

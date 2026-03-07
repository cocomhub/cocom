// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/pkg/clog"
	"github.com/cocomhub/cocom/pkg/conv"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	"github.com/cocomhub/cocom/pkg/mutex"
)

func SaveComicInfo(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	info := map[string]any{}
	err := json.NewDecoder(req.Body).Decode(&info)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		clog.Errorf(ctx, "decode body failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("decode body failed. errmsg: %s", err))
		return
	}
	clog.Debugf(ctx, "req info[%s]", conv.JSON(info))

	_, exist := info["id"]
	if !exist {
		w.WriteHeader(http.StatusBadRequest)
		clog.Errorf(ctx, "comic id not found failed")
		httpwrap.ResponseFail(ctx, w, "comic id not found failed")
		return
	}

	var cid int
	switch v := info["id"].(type) {
	case float64:
		cid = int(v)
	case string:
		cid, err = strconv.Atoi(v)
		info["id"] = cid
	default:
		err = fmt.Errorf("unknown type: cid[%v]", v)
	}
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		clog.Errorf(ctx, "request parse cid failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request parse cid failed. errmsg: %s", err))
		return
	}

	unlock, err := mutex.MutexLock(fmt.Sprintf("comic/%d", cid))
	if err != nil {
		w.WriteHeader(http.StatusTooManyRequests)
		clog.Errorf(ctx, "mutex lock failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("mutex lock failed. errmsg: %s", err))
		return
	}
	defer unlock()

	err = comic.UpdateComicInfo(ctx, cid, info)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		clog.Errorf(ctx, "update comic info failed. errmsg: %s", err)
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
		clog.Errorf(ctx, "request parse form failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request parse form failed. errmsg: %s", err))
		return
	}

	cid, err := strconv.Atoi(req.FormValue("id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		clog.Errorf(ctx, "request parse cid failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request parse cid failed. errmsg: %s", err))
		return
	}

	unlock, err := mutex.MutexLock(fmt.Sprintf("comic/%d", cid))
	if err != nil {
		w.WriteHeader(http.StatusTooManyRequests)
		clog.Errorf(ctx, "mutex lock failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("mutex lock failed. errmsg: %s", err))
		return
	}
	defer unlock()

	info := map[string]any{}
	err = comic.GetComicInfo(ctx, cid, &info)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		clog.Errorf(ctx, "get comic info failed. errmsg: %s", err)
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
		clog.Errorf(ctx, "decode body failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("decode body failed. errmsg: %s", err))
		return
	}
	clog.Debugf(ctx, "req[%s]", conv.JSON(req))

	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Second)
		defer cancel()
	}

	if comic.ComicDownloadConnOver() {
		clog.Warnf(ctx, "cid[%v] download comic conn over", req.Cid)
		httpwrap.Response(ctx, w, 1001, "download comic conn over", "")
		return
	}

	if !req.IsSync {
		go func() {
			ctx := clog.NewTraceCtx(clog.GetTraceID(ctx))
			taskFailed, err := comic.CreateDownloadTaskWithLock(ctx, req.Cid, req.MaxConn, req.MaxRetry, req.Force)
			if err != nil {
				clog.Errorf(ctx, "download comic task failed[%d]. errmsg: %s", taskFailed, err)
				return
			}
		}()
		httpwrap.Response(ctx, w, 1000, "async download task", "")
		return
	}

	taskFailed, err := comic.CreateDownloadTaskWithLock(ctx, req.Cid, req.MaxConn, req.MaxRetry, req.Force)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		clog.Errorf(ctx, "download comic task failed[%d]. errmsg: %s", taskFailed, err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("download comic task failed[%d]. errmsg: %s", taskFailed, err))
		return
	}
	httpwrap.ResponseSucc(ctx, w, "")
}

func RestoreComic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := api.RestoreComicByIDRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		clog.Errorf(ctx, "decode body failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("decode body failed. errmsg: %s", err))
		return
	}
	clog.Debugf(ctx, "req[%s]", conv.JSON(req))

	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Second)
		defer cancel()
	}

	unlock, err := mutex.MutexLock(fmt.Sprintf("comic/%d", req.Cid))
	if err != nil {
		w.WriteHeader(http.StatusTooManyRequests)
		clog.Errorf(ctx, "mutex lock failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("mutex lock failed. errmsg: %s", err))
		return
	}
	defer unlock()

	if !req.IsSync {
		go func() {
			ctx := clog.NewTraceCtx(clog.GetTraceID(ctx))
			if err := comic.RestoreComicByID(ctx, req.Cid); err != nil {
				clog.Errorf(ctx, "restore comic failed. cid[%d] errmsg: %s", req.Cid, err)
			}
		}()
		httpwrap.Response(ctx, w, 1000, "async restore task", "")
		return
	}

	if err := comic.RestoreComicByID(ctx, req.Cid); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		clog.Errorf(ctx, "restore comic failed. cid[%d] errmsg: %s", req.Cid, err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("restore comic failed. errmsg: %s", err))
		return
	}
	httpwrap.ResponseSucc(ctx, w, "")
}

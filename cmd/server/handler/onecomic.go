// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cocomhub/cocom/cmd/server/internal/onecomic"
	"github.com/cocomhub/cocom/pkg/clog"
	"github.com/cocomhub/cocom/pkg/conv"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	"github.com/cocomhub/cocom/pkg/mutex"
)

func SaveOneComicInfo(w http.ResponseWriter, req *http.Request) {
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

	_, exist := info["cid"]
	if !exist {
		comicid, exist := info["comicid"]
		if !exist {
			w.WriteHeader(http.StatusBadRequest)
			clog.Errorf(ctx, "cid or comicid and site not found failed")
			httpwrap.ResponseFail(ctx, w, "cid or comicid and site not found failed")
			return
		}

		site, exist := info["site"]
		if !exist {
			w.WriteHeader(http.StatusBadRequest)
			clog.Errorf(ctx, "cid or comicid and site not found failed")
			httpwrap.ResponseFail(ctx, w, "cid or comicid and site not found failed")
			return
		}

		info["cid"] = fmt.Sprintf("[%s]%s", site, comicid)
	}

	cid := fmt.Sprint(info["cid"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		clog.Errorf(ctx, "request parse cid failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request parse cid failed. errmsg: %s", err))
		return
	}

	unlock, err := mutex.MutexLock(fmt.Sprintf("onecomic/%s", cid))
	if err != nil {
		w.WriteHeader(http.StatusTooManyRequests)
		clog.Errorf(ctx, "mutex lock failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("mutex lock failed. errmsg: %s", err))
		return
	}
	defer unlock()

	err = onecomic.UpdateOneComicInfo(ctx, cid, info)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		clog.Errorf(ctx, "update onecomic info failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("update onecomic info failed. errmsg: %s", err))
		return
	}

	httpwrap.ResponseSucc(ctx, w, "")
}

func GetOneComicInfo(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	err := req.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		clog.Errorf(ctx, "request parse form failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request parse form failed. errmsg: %s", err))
		return
	}

	cid := req.FormValue("cid")
	if len(cid) == 0 {
		comicid := req.FormValue("comicid")
		if len(comicid) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			clog.Errorf(ctx, "request parse cid failed. errmsg: cid or comicid and site not found")
			httpwrap.ResponseFail(ctx, w, fmt.Sprintf("cid or comicid and site not found"))
			return
		}

		site := req.FormValue("site")
		if len(site) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			clog.Errorf(ctx, "request parse cid failed. errmsg: cid or comicid and site not found")
			httpwrap.ResponseFail(ctx, w, fmt.Sprintf("cid or comicid and site not found"))
			return
		}

		cid = fmt.Sprintf("[%s]%s", site, comicid)
	}

	unlock, err := mutex.MutexLock(fmt.Sprintf("onecomic/%s", cid))
	if err != nil {
		w.WriteHeader(http.StatusTooManyRequests)
		clog.Errorf(ctx, "mutex lock failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("mutex lock failed. errmsg: %s", err))
		return
	}
	defer unlock()

	info := map[string]any{}
	err = onecomic.GetOneComicInfo(ctx, cid, &info)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		clog.Errorf(ctx, "get onecomic info failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("get onecomic info failed. errmsg: %s", err))
		return
	}

	httpwrap.ResponseSucc(ctx, w, info)
}

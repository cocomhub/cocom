// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/pkg/clog"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	"github.com/cocomhub/cocom/pkg/mutex"
)

type tagDiff struct {
	Added   []api.Tag `json:"added"`
	Removed []api.Tag `json:"removed"`
	Current []api.Tag `json:"current"`
}

func AddLikeTag(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	err := req.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		clog.Errorf(ctx, "request parse form failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request parse form failed. errmsg: %s", err))
		return
	}

	cid, err := strconv.Atoi(req.FormValue("cid"))
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

	info := api.ComicInfo{}
	if err = comic.GetComicInfo(ctx, cid, &info); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		clog.Errorf(ctx, "get comic info failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("get comic info failed. errmsg: %s", err))
		return
	}

	like := api.Tag{Type: "custom", Name: "like", URL: "/custom/like/", ID: 99999, Count: 1}
	diff := tagDiff{Current: info.Tags}
	updated := false
	for i, t := range info.Tags {
		if t.Type == "custom" && t.Name == "like" {
			if t.ID != like.ID || t.URL != like.URL || t.Count != like.Count {
				info.Tags[i] = like
				updated = true
			}
		}
	}
	if !updated {
		exists := false
		for _, t := range info.Tags {
			if t.Type == "custom" && t.Name == "like" {
				exists = true
				break
			}
		}
		if !exists {
			info.Tags = append(info.Tags, like)
			diff.Added = append(diff.Added, like)
			updated = true
		}
	}
	if updated {
		m, err := info.ToMapInfo()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			clog.Errorf(ctx, "encode comic info failed. errmsg: %s", err)
			httpwrap.ResponseFail(ctx, w, fmt.Sprintf("encode comic info failed. errmsg: %s", err))
			return
		}
		if err = comic.UpdateComicInfo(ctx, cid, m); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			clog.Errorf(ctx, "update comic info failed. errmsg: %s", err)
			httpwrap.ResponseFail(ctx, w, fmt.Sprintf("update comic info failed. errmsg: %s", err))
			return
		}
		diff.Current = info.Tags
	}

	httpwrap.ResponseSucc(ctx, w, diff)
}

func RemoveLikeTag(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	err := req.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		clog.Errorf(ctx, "request parse form failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request parse form failed. errmsg: %s", err))
		return
	}

	cid, err := strconv.Atoi(req.FormValue("cid"))
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

	info := api.ComicInfo{}
	if err = comic.GetComicInfo(ctx, cid, &info); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		clog.Errorf(ctx, "get comic info failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("get comic info failed. errmsg: %s", err))
		return
	}

	diff := tagDiff{}
	for _, t := range info.Tags {
		if t.Type == "custom" && t.Name == "like" {
			diff.Removed = append(diff.Removed, t)
		} else {
			diff.Current = append(diff.Current, t)
		}
	}

	if len(diff.Removed) > 0 {
		info.Tags = diff.Current
		m, err := info.ToMapInfo()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			clog.Errorf(ctx, "encode comic info failed. errmsg: %s", err)
			httpwrap.ResponseFail(ctx, w, fmt.Sprintf("encode comic info failed. errmsg: %s", err))
			return
		}
		if err = comic.UpdateComicInfo(ctx, cid, m); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			clog.Errorf(ctx, "update comic info failed. errmsg: %s", err)
			httpwrap.ResponseFail(ctx, w, fmt.Sprintf("update comic info failed. errmsg: %s", err))
			return
		}
	}

	httpwrap.ResponseSucc(ctx, w, diff)
}

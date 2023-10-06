/*
Copyright © 2023 suixibing <suixibing@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/suixibing/cocom/cmd/server/internal/comic"
	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/conv"
	"github.com/suixibing/cocom/pkg/httpwrap"
	"github.com/suixibing/cocom/pkg/mutex"
)

func init() {
	mux.HandleFunc("/comic/saveComicInfo", SaveComicInfo)
	mux.HandleFunc("/comic/getComicInfo", GetComicInfo)
}

func SaveComicInfo(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	info := map[string]interface{}{}
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

	httpwrap.ResponseSucc(ctx, w, nil)
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

	info, err := comic.GetComicInfo(ctx, cid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		clog.Errorf(ctx, "get comic info failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("get comic info failed. errmsg: %s", err))
		return
	}

	httpwrap.ResponseSucc(ctx, w, info)
}

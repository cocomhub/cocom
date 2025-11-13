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
	"strings"

	"github.com/suixibing/cocom/cmd/server/api"
	"github.com/suixibing/cocom/cmd/server/internal/setting"
	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/conv"
	"github.com/suixibing/cocom/pkg/httpwrap"
)

func init() {
	mux.HandleFunc("/api/settings", Setting)
}

func Setting(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	switch req.Method {
	case http.MethodGet:
		GetSetting(w, req)
	case http.MethodPost:
		SetSetting(w, req)
	case http.MethodDelete:
		DelSetting(w, req)
	default:
		clog.Errorf(ctx, "invalid request method[%s]", req.Method)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("invalid request method[%s]", req.Method))
	}
}

func GetSetting(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	err := req.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		clog.Errorf(ctx, "request parse form failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request parse form failed. errmsg: %s", err))
		return
	}

	settings, err := setting.GetSettings(ctx, req.FormValue("type"), strings.Split(req.FormValue("keys"), ",")...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		clog.Errorf(ctx, "get settings failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("get settings failed. errmsg: %s", err))
		return
	}

	httpwrap.ResponseSucc(ctx, w, settings)
}

func SetSetting(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	info := api.SetSettingsRequest{}
	err := json.NewDecoder(req.Body).Decode(&info)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		clog.Errorf(ctx, "decode body failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("decode body failed. errmsg: %s", err))
		return
	}
	clog.Debugf(ctx, "req info[%s]", conv.JSON(info))

	err = setting.SetSettings(ctx, info.Type, info.Settings)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		clog.Errorf(ctx, "set settings failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("set settings failed. errmsg: %s", err))
		return
	}

	httpwrap.ResponseSucc(ctx, w, "")
}

func DelSetting(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	err := req.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		clog.Errorf(ctx, "request parse form failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request parse form failed. errmsg: %s", err))
		return
	}

	_, err = setting.DelSettings(ctx, req.FormValue("type"), strings.Split(req.FormValue("keys"), ",")...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		clog.Errorf(ctx, "del settings failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("del settings failed. errmsg: %s", err))
		return
	}

	httpwrap.ResponseSucc(ctx, w, "")
}

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
	"fmt"
	"net/http"
	"strconv"

	"github.com/suixibing/cocom/cmd/server/internal/custom"
	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/httpwrap"
)

func init() {
	mux.HandleFunc("/api/comic/addLikeGroup", AddLikeGroup)
}

func AddLikeGroup(w http.ResponseWriter, req *http.Request) {
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

	err = custom.AddLikeGroup(ctx, cid)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		clog.Errorf(ctx, "request add like group failed. errmsg: %#v", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request add like group failed. errmsg: %s", err))
		return
	}
	httpwrap.ResponseSucc(ctx, w, nil)
}

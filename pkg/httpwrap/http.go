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
package httpwrap

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/suixibing/cocom/pkg/clog"
)

type ResponseHeadInfo struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	RequestID string `json:"request_id"`
	Time      string `json:"time"`
}

type ResponseInfo struct {
	Head ResponseHeadInfo `json:"head"`
	Body interface{}      `json:"body,omitempty"`
}

func Response(ctx context.Context, w http.ResponseWriter, code int, msg string, body interface{}) {
	data, _ := json.Marshal(ResponseInfo{
		Head: ResponseHeadInfo{
			Code:      code,
			Msg:       msg,
			RequestID: clog.GetTraceID(ctx),
			Time:      time.Now().Format(time.RFC3339Nano),
		},
		Body: body,
	})
	clog.Debugf(ctx, "response data(%s)", data)
	_, _ = w.Write(data)
}

func ResponseSucc(ctx context.Context, w http.ResponseWriter, body interface{}) {
	Response(ctx, w, 0, "succ", body)
}

func ResponseFail(ctx context.Context, w http.ResponseWriter, msg string) {
	Response(ctx, w, -1, msg, nil)
}

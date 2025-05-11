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
	"net/http/pprof"
	"time"

	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/util"
)

var mux = &ServeMux{ServeMux: http.NewServeMux()}

type ServeMux struct {
	*http.ServeMux
}

func (mux *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r = r.WithContext(clog.WithTraceID(r.Context(), fmt.Sprintf("%d_%d", time.Now().UnixMicro(), util.Uint64())))
	clog.Debugf(r.Context(), "recv request. uri(%s)", r.RequestURI)
	mux.ServeMux.ServeHTTP(w, r)
}

func init() {
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
}

func Mux() *ServeMux {
	return mux
}

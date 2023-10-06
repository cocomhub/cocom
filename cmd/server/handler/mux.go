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
	"math/rand"
	"net/http"
	"time"

	"github.com/suixibing/cocom/pkg/clog"
)

var (
	mux = &ServeMux{ServeMux: http.NewServeMux()}
)

type ServeMux struct {
	*http.ServeMux
}

func init() {
	rand.Seed(time.Now().UnixMicro())
}

func (mux *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r = r.WithContext(clog.WithTraceID(r.Context(), fmt.Sprintf("%d_%d", time.Now().UnixMicro(), rand.Uint64())))
	clog.Debugf(r.Context(), "recv request. uri(%s)", r.RequestURI)
	mux.ServeMux.ServeHTTP(w, r)
}

func Mux() *ServeMux {
	return mux
}

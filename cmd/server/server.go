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
package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/suixibing/cocom/cmd/server/handler"
	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/httpwrap"

	"github.com/spf13/viper"
)

var (
	server http.Server
)

func Run() {
	ctx := clog.NewTraceCtx("server")

	addr := fmt.Sprintf(":%d", viper.GetInt32("port"))
	_, _ = fmt.Fprintf(os.Stderr, "cocom server listen on [%s]", addr)
	clog.Infof(ctx, "cocom server listen on [%s]", addr)

	shutdownCh := make(chan context.Context)
	wg := sync.WaitGroup{}

	go func() {
		wg.Add(1)
		defer wg.Done()

		select {
		case <-shutdownCh:
			clog.Infof(ctx, "server shutdown start...")
			err := server.Shutdown(ctx)
			if err != nil {
				clog.Errorf(ctx, "server shutdown failed: %s", err)
				return
			}
			clog.Infof(ctx, "server shutdown")
		}
	}()

	mux := *handler.Mux()
	mux.HandleFunc("/server/shutdown", func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		select {
		case shutdownCh <- ctx:
			close(shutdownCh)
			httpwrap.ResponseSucc(ctx, w, "server shutdown start")
		default:
			httpwrap.ResponseFail(ctx, w, "shutdown failed")
		}
	})

	server = http.Server{Addr: addr, Handler: &mux}

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		clog.Fatalf(ctx, "server listen and serve failed: %s", err)
	}
	clog.Infof(ctx, "server stop listen")
	wg.Wait()
}

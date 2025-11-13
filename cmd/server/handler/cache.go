package handler

import (
	"fmt"
	"net/http"

	"github.com/suixibing/cocom/cmd/server/internal/cache"
	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/httpwrap"
)

func init() {
	mux.HandleFunc("/api/cache/reset", ResetCache)
}

func ResetCache(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	err := cache.Reset()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		clog.Errorf(ctx, "reset cache failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("reset cache failed. errmsg: %s", err))
		return
	}

	err = cache.ResetStats()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		clog.Errorf(ctx, "reset cache stats failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("reset cache stats failed. errmsg: %s", err))
		return
	}
	httpwrap.ResponseSucc(ctx, w, "")
}

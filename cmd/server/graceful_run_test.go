package server

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gin-contrib/graceful"
	"github.com/spf13/viper"
)

func TestHTTPStartAndGracefulShutdown(t *testing.T) {
	viper.Set("server.listen.http.addr", "127.0.0.1:0")
	viper.Set("server.shutdown_timeout", 500*time.Millisecond)

	shutdownCh := make(chan context.Context)
	r := BuildEngine(context.Background(), shutdownCh)

	gr, err := graceful.New(
		r,
		graceful.WithAddr(viper.GetString("server.listen.http.addr")),
		graceful.WithShutdownTimeout(viper.GetDuration("server.shutdown_timeout")),
	)
	if err != nil {
		t.Fatalf("graceful.New error: %v", err)
	}
	defer gr.Close()

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-shutdownCh
		cancel()
	}()

	errCh := make(chan error, 1)
	go func() {
		errCh <- gr.RunWithContext(runCtx)
	}()

	time.Sleep(100 * time.Millisecond)
	select {
	case shutdownCh <- context.Background():
	default:
		t.Fatalf("failed to send shutdown signal")
	}

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("server exit error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("server did not shutdown in time")
	}
}

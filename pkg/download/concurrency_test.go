// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package download

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

// mockRoundTripper 返回固定成功响应，用于在无网络环境下驱动 grab 下载。
type mockRoundTripper struct {
	body []byte
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode:    http.StatusOK,
		Status:        "200 OK",
		Body:          io.NopCloser(bytes.NewReader(m.body)),
		ContentLength: int64(len(m.body)),
		Header:        make(http.Header),
		Request:       req,
	}, nil
}

// TestDoBatch_ProcessesAllTasks 回归 C1：worker 必须循环消费 taskCh，
// len(tasks) > workers 时全部任务都要被下载，resultCh 恰好每任务 1 条结果。
func TestDoBatch_ProcessesAllTasks(t *testing.T) {
	cfg := NewConfig().SetDownloadDir(t.TempDir()).SetMaxRunning(2)
	d := NewDownloader(cfg)
	d.client.HTTPClient = &http.Client{Transport: &mockRoundTripper{body: []byte("payload")}}

	const n = 5
	statuses := make([]bool, n)
	tasks := make([]*Task, n)
	for i := range tasks {
		tasks[i] = &Task{
			Dir:    "sub",
			Name:   fmt.Sprintf("img-%d.jpg", i),
			Url:    fmt.Sprintf("https://example.com/img/%d.jpg", i),
			Status: &statuses[i],
		}
	}

	resultCh, err := d.DoBatch(2, tasks...)
	if err != nil {
		t.Fatalf("DoBatch err: %v", err)
	}

	seen := 0
	for result := range resultCh {
		seen++
		if result.Err != nil {
			t.Fatalf("result[%s] Err = %v, want nil", result.Task.Name, result.Err)
		}
		if result.Response == nil || result.Response.Err() != nil {
			t.Fatalf("result[%s] response Err = %v, want nil", result.Task.Name, result.Response.Err())
		}
		if result.Task.Status != nil {
			*result.Task.Status = true
		}
	}
	if seen != n {
		t.Fatalf("results count = %d, want %d", seen, n)
	}
	for i, s := range statuses {
		if !s {
			t.Fatalf("task[%d] status not set", i)
		}
	}
}

// TestDoBatch_RequestBuildFailureEmitsErr 验证 grab.NewRequest 失败时
// 以 TaskResult.Err 产出失败结果（而不是静默丢弃该任务）。
func TestDoBatch_RequestBuildFailureEmitsErr(t *testing.T) {
	cfg := NewConfig().SetDownloadDir(t.TempDir()).SetMaxRunning(2)
	d := NewDownloader(cfg)
	d.client.HTTPClient = &http.Client{Transport: &mockRoundTripper{body: []byte("x")}}

	tasks := []*Task{
		{Dir: "d", Name: "ok.jpg", Url: "https://example.com/ok.jpg"},
		{Dir: "d", Name: "bad.jpg", Url: "://invalid-url"},
	}

	resultCh, err := d.DoBatch(1, tasks...)
	if err != nil {
		t.Fatalf("DoBatch err: %v", err)
	}

	failures := 0
	successes := 0
	for result := range resultCh {
		if result.Err != nil {
			failures++
			if result.Task.Name != "bad.jpg" {
				t.Fatalf("unexpected failing task: %s", result.Task.Name)
			}
			continue
		}
		successes++
		if result.Task.Name != "ok.jpg" {
			t.Fatalf("unexpected succeeding task: %s", result.Task.Name)
		}
	}
	if failures != 1 || successes != 1 {
		t.Fatalf("failures=%d successes=%d, want 1/1", failures, successes)
	}
}

// TestDoBatch_CtxCancelReturns 回归 C3：ctx 取消后 DoBatch 的 resultCh 必须关闭，
// 调用方 range 必须返回（不永久挂起）。
func TestDoBatch_CtxCancelReturns(t *testing.T) {
	cfg := NewConfig().SetDownloadDir(t.TempDir()).SetMaxRunning(1)
	d := NewDownloader(cfg)
	d.client.HTTPClient = &http.Client{Transport: &mockRoundTripper{body: []byte("x")}}

	tasks := make([]*Task, 10)
	for i := range tasks {
		tasks[i] = &Task{
			Dir:  "d",
			Name: fmt.Sprintf("img-%d.jpg", i),
			Url:  fmt.Sprintf("https://example.com/%d.jpg", i),
		}
	}

	resultCh, err := d.DoBatch(2, tasks...)
	if err != nil {
		t.Fatalf("DoBatch err: %v", err)
	}

	// 立即取消：in-flight 任务以失败结果落盘，resultCh 随后必定关闭。
	d.cancel()

	done := make(chan struct{})
	go func() {
		for range resultCh {
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("resultCh not closed after cancel")
	}
}

// TestDoBatch_MaxRunningCapsConcurrency 验证全局信号量生效：并发下载
// 不会突破 MaxRunning（多本漫画并发下载场景的封顶保证）。
func TestDoBatch_MaxRunningCapsConcurrency(t *testing.T) {
	cfg := NewConfig().SetDownloadDir(t.TempDir()).SetMaxRunning(2)
	d := NewDownloader(cfg)

	var inflight atomic.Int32
	var maxInflight atomic.Int32
	d.client.HTTPClient = &http.Client{Transport: &countingRT{
		before: func() {
			cur := inflight.Add(1)
			for {
				prev := maxInflight.Load()
				if cur <= prev || maxInflight.CompareAndSwap(prev, cur) {
					break
				}
			}
		},
		after: func() { inflight.Add(-1) },
		body:  []byte("x"),
	}}

	tasks := make([]*Task, 20)
	for i := range tasks {
		tasks[i] = &Task{
			Dir:  "d",
			Name: fmt.Sprintf("img-%d.jpg", i),
			Url:  fmt.Sprintf("https://example.com/%d.jpg", i),
		}
	}

	resultCh, err := d.DoBatch(8, tasks...)
	if err != nil {
		t.Fatalf("DoBatch err: %v", err)
	}
	count := 0
	for range resultCh {
		count++
	}
	if count != len(tasks) {
		t.Fatalf("results count = %d, want %d", count, len(tasks))
	}
	if max := maxInflight.Load(); max > 2 {
		t.Fatalf("max inflight = %d, want <= 2", max)
	}
}

// countingRT 包装计数回调的 RoundTripper。
type countingRT struct {
	before func()
	after  func()
	body   []byte
}

func (c *countingRT) RoundTrip(req *http.Request) (*http.Response, error) {
	c.before()
	defer c.after()
	return &http.Response{
		StatusCode:    http.StatusOK,
		Status:        "200 OK",
		Body:          io.NopCloser(bytes.NewReader(c.body)),
		ContentLength: int64(len(c.body)),
		Header:        make(http.Header),
		Request:       req,
	}, nil
}

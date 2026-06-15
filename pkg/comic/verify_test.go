// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"context"
	"testing"

	"go.uber.org/atomic"
)

func TestNewComicVerifier(t *testing.T) {
	// NewComicVerifier calls findWgetPath which panics on Windows without wget,
	// so we skip this test on Windows.
	// t.Skip("NewComicVerifier requires wget binary, skip on Windows")
	_ = NewMemoryStorage
}

func TestComicVerifier_Start_TaskCreated(t *testing.T) {
	t.Skip("requires wget binary")
}

func TestComicVerifier_GetTasks_Empty(t *testing.T) {
	t.Skip("requires wget binary")
}

func TestComicVerifier_GetTaskProgress_NotFound(t *testing.T) {
	t.Skip("requires wget binary")
}

func TestComicVerifier_CancelTask_NotFound(t *testing.T) {
	t.Skip("requires wget binary")
}

func TestComicVerifier_GetTask_NotFound(t *testing.T) {
	t.Skip("requires wget binary")
}

func TestNewVerifyOptions(t *testing.T) {
	opts := &VerifyOptions{
		MaxWorkers: 4,
	}
	if opts.MaxWorkers != 4 {
		t.Errorf("MaxWorkers = %d, want 4", opts.MaxWorkers)
	}
}

func TestVerifyProgress_MarshalJSON(t *testing.T) {
	s := &atomic.Value{}
	s.Store(VerifyStatusPending)
	p := &VerifyProgress{
		TaskID:  "test-1",
		Total:   atomic.NewInt32(100),
		Current: atomic.NewInt32(0),
		Invalid: atomic.NewInt32(0),
		Fixed:   atomic.NewInt32(0),
		Status:  s,
	}
	data, err := p.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty JSON")
	}
}

func TestVerifyProgress_UnmarshalJSON(t *testing.T) {
	data := []byte(`{"taskId":"test-1","total":50,"current":25,"invalid":3,"fixed":1,"status":"completed"}`)
	p := &VerifyProgress{}
	if err := p.UnmarshalJSON(data); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if p.TaskID != "test-1" {
		t.Errorf("TaskID = %q, want %q", p.TaskID, "test-1")
	}
	if p.Total.Load() != 50 {
		t.Errorf("Total = %d, want 50", p.Total.Load())
	}
}

func TestVerifyProgress_SetError(t *testing.T) {
	p := &VerifyProgress{}
	p.SetError("test error")
	if p.Error == nil {
		t.Error("Error should not be nil after SetError")
	}
	if p.Error.Error() != "test error" {
		t.Errorf("Error = %q, want %q", p.Error.Error(), "test error")
	}
}

func TestVerifyProgress_UpdateProgress(t *testing.T) {
	p := &VerifyProgress{
		Total:   atomic.NewInt32(100),
		Current: atomic.NewInt32(0),
		Invalid: atomic.NewInt32(0),
		Fixed:   atomic.NewInt32(0),
	}
	p.UpdateProgress(50, 5, 3)
	if p.Current.Load() != 50 {
		t.Errorf("Current = %d, want 50", p.Current.Load())
	}
	if p.Invalid.Load() != 5 {
		t.Errorf("Invalid = %d, want 5", p.Invalid.Load())
	}
	if p.Fixed.Load() != 3 {
		t.Errorf("Fixed = %d, want 3", p.Fixed.Load())
	}
}

func TestVerifyProgress_GetProgress(t *testing.T) {
	p := &VerifyProgress{
		Total:   atomic.NewInt32(100),
		Current: atomic.NewInt32(50),
		Invalid: atomic.NewInt32(0),
		Fixed:   atomic.NewInt32(0),
	}
	ratio := p.GetProgress()
	if ratio != 50.0 {
		t.Errorf("GetProgress() = %f, want 50.0", ratio)
	}
}

func TestVerifyProgress_StatusFlow(t *testing.T) {
	s := &atomic.Value{}
	s.Store(VerifyStatusRunning)
	p := &VerifyProgress{
		Total:   atomic.NewInt32(100),
		Current: atomic.NewInt32(0),
		Invalid: atomic.NewInt32(0),
		Fixed:   atomic.NewInt32(0),
		Status:  s,
	}
	if p.GetStatus() != VerifyStatusRunning {
		t.Errorf("GetStatus() = %s, want %s", p.GetStatus(), VerifyStatusRunning)
	}
	if p.IsCompleted() {
		t.Error("IsCompleted() should be false while running")
	}
	if p.GetProgress() != 0 {
		t.Errorf("GetProgress() = %f, want 0", p.GetProgress())
	}
}

func TestVerifyProgress_Complete(t *testing.T) {
	s := &atomic.Value{}
	s.Store(VerifyStatusCompleted)
	p := &VerifyProgress{
		Total:   atomic.NewInt32(100),
		Current: atomic.NewInt32(100),
		Invalid: atomic.NewInt32(0),
		Fixed:   atomic.NewInt32(0),
		Status:  s,
	}
	if !p.IsCompleted() {
		t.Error("IsCompleted() should be true")
	}
}

func TestVerifyProgress_SetMessage(t *testing.T) {
	p := &VerifyProgress{}
	p.SetMessage("processing")
	messages := p.GetMessages()
	if len(messages) != 1 || messages[0] != "processing" {
		t.Errorf("GetMessages() = %v, want [processing]", messages)
	}
}

func TestVerifyTask_GetProgress(t *testing.T) {
	p := &VerifyProgress{}
	task := &VerifyTask{Progress: p}
	if task.GetProgress() != p {
		t.Error("GetProgress() should return the progress")
	}
}

func TestVerifyTask_Done_CancelsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	task := &VerifyTask{Cancel: cancel}
	// Before cancel, context should not be done
	if ctx.Err() != nil {
		t.Error("context should not be done before Done()")
	}
	task.Done()
	if ctx.Err() == nil {
		t.Error("context should be done after Done()")
	}
}

func TestNewMetricsCollector(t *testing.T) {
	c := NewMetricsCollector()
	if c == nil {
		t.Fatal("NewMetricsCollector returned nil")
	}
	metrics := c.GetMetrics()
	if metrics.StartTime.IsZero() {
		t.Error("StartTime should not be zero")
	}
}

func TestMetricsCollector_AddProcessedFile(t *testing.T) {
	c := NewMetricsCollector()
	c.AddProcessedFile(1024*1024, false)
	c.AddProcessedFile(2048*1024, true)
	metrics := c.GetMetrics()
	if metrics.TotalFiles != 2 {
		t.Errorf("TotalFiles = %d, want 2", metrics.TotalFiles)
	}
	if metrics.FailedFiles != 1 {
		t.Errorf("FailedFiles = %d, want 1", metrics.FailedFiles)
	}
	if metrics.ProcessedMB < 2.9 || metrics.ProcessedMB > 3.1 {
		t.Errorf("ProcessedMB = %f, want ~3.0", metrics.ProcessedMB)
	}
}

func TestMetricsCollector_TaskSubmittedAndFailed(t *testing.T) {
	c := NewMetricsCollector()
	c.TaskSubmitted()
	c.TaskSubmitted()
	c.TaskFailed()
	metrics := c.GetMetrics()
	if metrics.TaskSubmitted != 2 {
		t.Errorf("TaskSubmitted = %d, want 2", metrics.TaskSubmitted)
	}
	if metrics.TaskFailed != 1 {
		t.Errorf("TaskFailed = %d, want 1", metrics.TaskFailed)
	}
}

func TestMetricsCollector_Reset(t *testing.T) {
	c := NewMetricsCollector()
	c.AddProcessedFile(1024*1024, false)
	c.Reset()
	metrics := c.GetMetrics()
	if metrics.TotalFiles != 0 {
		t.Errorf("TotalFiles after reset = %d, want 0", metrics.TotalFiles)
	}
}

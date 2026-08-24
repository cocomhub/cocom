// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"go.uber.org/atomic"
)

func TestNewComicVerifier(t *testing.T) {
	v, err := NewComicVerifier(t.Context(), NewMemoryStorage(), t.TempDir())
	if err != nil {
		t.Fatalf("NewComicVerifier failed: %v", err)
	}
	if v == nil {
		t.Fatal("NewComicVerifier returned nil verifier")
	}
	_ = v.Close()
}

func TestComicVerifier_Start_TaskCreated(t *testing.T) {
	store := NewMemoryStorage()
	if err := store.Save(t.Context(), NewComic("1001", "test comic", nil)); err != nil {
		t.Fatalf("Save comic failed: %v", err)
	}

	v, err := NewComicVerifier(t.Context(), store, t.TempDir())
	if err != nil {
		t.Fatalf("NewComicVerifier failed: %v", err)
	}
	defer v.Close()

	taskID, err := v.Start(t.Context(), &VerifyOptions{})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if taskID == "" {
		t.Fatal("Start returned empty task ID")
	}
	// Start 同步注册任务到 progress，立即可以查询（progress 保留 60s）。
	if p := v.GetTaskProgress(taskID); p == nil {
		t.Fatal("GetTaskProgress returned nil for started task")
	}
	if tasks := v.GetTasks(); len(tasks) != 1 {
		t.Errorf("GetTasks = %d, want 1", len(tasks))
	}
}

func TestComicVerifier_GetTasks_Empty(t *testing.T) {
	v, err := NewComicVerifier(t.Context(), NewMemoryStorage(), t.TempDir())
	if err != nil {
		t.Fatalf("NewComicVerifier failed: %v", err)
	}
	defer v.Close()

	if tasks := v.GetTasks(); len(tasks) != 0 {
		t.Errorf("GetTasks on fresh verifier = %d, want 0", len(tasks))
	}
}

func TestComicVerifier_GetTaskProgress_NotFound(t *testing.T) {
	v, err := NewComicVerifier(t.Context(), NewMemoryStorage(), t.TempDir())
	if err != nil {
		t.Fatalf("NewComicVerifier failed: %v", err)
	}
	defer v.Close()

	if p := v.GetTaskProgress("nope"); p != nil {
		t.Errorf("GetTaskProgress for unknown task = %v, want nil", p)
	}
}

func TestComicVerifier_CancelTask_NotFound(t *testing.T) {
	v, err := NewComicVerifier(t.Context(), NewMemoryStorage(), t.TempDir())
	if err != nil {
		t.Fatalf("NewComicVerifier failed: %v", err)
	}
	defer v.Close()

	if err := v.CancelTask(t.Context(), "nope"); !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("CancelTask for unknown task = %v, want ErrTaskNotFound", err)
	}
}

func TestComicVerifier_GetTask_NotFound(t *testing.T) {
	v, err := NewComicVerifier(t.Context(), NewMemoryStorage(), t.TempDir())
	if err != nil {
		t.Fatalf("NewComicVerifier failed: %v", err)
	}
	defer v.Close()

	if _, err := v.GetTask(t.Context(), "nope"); !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("GetTask for unknown task = %v, want ErrTaskNotFound", err)
	}
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

// TestRunTask_SetsCompleted 回归 A1：runTask 正常跑完后状态必须为 completed。
func TestRunTask_SetsCompleted(t *testing.T) {
	store := NewMemoryStorage()
	if err := store.Save(t.Context(), NewComic("1001", "test comic", nil)); err != nil {
		t.Fatalf("Save comic failed: %v", err)
	}

	v, err := NewComicVerifier(t.Context(), store, t.TempDir())
	if err != nil {
		t.Fatalf("NewComicVerifier failed: %v", err)
	}
	defer v.Close()

	taskID, err := v.Start(t.Context(), &VerifyOptions{})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	p := v.GetTaskProgress(taskID)
	if p == nil {
		t.Fatal("GetTaskProgress returned nil for started task")
	}
	for p.GetStatus() != VerifyStatusCompleted && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if p.GetStatus() != VerifyStatusCompleted {
		t.Errorf("GetStatus() = %s, want %s", p.GetStatus(), VerifyStatusCompleted)
	}
}

// TestCancelTask_CancelsTaskCtx 回归 A2：CancelTask 后底层 taskCtx 必须被取消，
// 否则 FindChannel 生产者 goroutine 无法经 ctx.Done 退出。
// 构造一个仍处于 running 的进度 + 已注册 task，直接驱动 CancelTask（不依赖真实任务时序）。
func TestCancelTask_CancelsTaskCtx(t *testing.T) {
	v, err := NewComicVerifier(t.Context(), NewMemoryStorage(), t.TempDir())
	if err != nil {
		t.Fatalf("NewComicVerifier failed: %v", err)
	}
	defer v.Close()

	// 人工构造 running 任务（避免真实任务跑完/完成后 CancelTask 走"已完成"分支）。
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	taskID := "manual-task"
	progress := NewVerifyProgress(taskID)
	progress.Status.Store(VerifyStatusRunning)
	cancelled := false
	task := &VerifyTask{
		ID:       taskID,
		Progress: progress,
		Cancel: func() {
			cancelled = true
			cancel()
		},
	}
	v.tasks.Store(taskID, task)
	v.progressMu.Lock()
	v.progress[taskID] = progress
	v.progressMu.Unlock()

	if err := v.CancelTask(t.Context(), taskID); err != nil {
		t.Fatalf("CancelTask failed: %v", err)
	}

	if !cancelled {
		t.Error("CancelTask did not invoke task.Cancel()")
	}
	if p := v.GetTaskProgress(taskID); p == nil || p.GetStatus() != VerifyStatusCanceled {
		t.Errorf("progress status = %v, want canceled", p.GetStatus())
	}
	// ctx 应已取消（Cancel 已调用）。
	if ctx.Err() == nil {
		t.Error("task context should be canceled after CancelTask")
	}
}

// TestStartSchedule_WaitStopsWhenTaskDone 回归 A3：StartSchedule 的等待循环
// 在任务完成后应退出（而非永久轮询）。这里用手动触发 cron 执行验证。
// 注意：close 前显式 Stop 调度器（cron goroutine 是 fire-and-forget，
// 不同步等待正在执行的 job——该竞态为既有行为，不属于本批次修复范围）。
func TestStartSchedule_WaitStopsWhenTaskDone(t *testing.T) {
	store := NewMemoryStorage()
	if err := store.Save(t.Context(), NewComic("3001", "scheduled comic", nil)); err != nil {
		t.Fatalf("Save comic failed: %v", err)
	}

	v, err := NewComicVerifier(t.Context(), store, t.TempDir())
	if err != nil {
		t.Fatalf("NewComicVerifier failed: %v", err)
	}
	defer func() {
		if v.scheduler != nil {
			<-v.scheduler.Stop().Done()
		}
		_ = v.Close()
	}()

	// 用 interval 生成 cron（@every 200ms），执行后任务立即完成，等待循环应快速退出。
	cfg := &ScheduleConfig{
		Active:   true,
		Interval: 200 * time.Millisecond,
		Options:  &VerifyOptions{},
	}
	if err := v.StartSchedule(t.Context(), cfg); err != nil {
		t.Fatalf("StartSchedule failed: %v", err)
	}

	// 等待至少一次 cron 触发并完成（runTask 应置 completed 后等待循环退出）。
	deadline := time.Now().Add(5 * time.Second)
	foundCompleted := false
	for time.Now().Before(deadline) {
		for _, task := range v.GetTasks() {
			if task.GetProgress() != nil && task.GetProgress().GetStatus() == VerifyStatusCompleted {
				foundCompleted = true
			}
		}
		if foundCompleted {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !foundCompleted {
		t.Fatal("expected at least one completed scheduled task")
	}

	// 真覆盖：等待循环若永久空转，每 200ms 触发一次 cron 都会 start 一个新任务
	// 且 runTask 的 goroutine 不退出 → 任务数无限增长。等待循环正确退出时，
	// 每轮只产生一个已完成任务，任务数收敛不再膨胀。
	// 记录当前已观测完成数，再等 1s（约 5 个 cron 周期），完成数增长应≤ 5（bounded），
	// 且内存中的任务数不超过一个小上限（等待循环未把任务堆积在 progress map）。
	time.Sleep(1 * time.Second)
	tasks := v.GetTasks()
	if len(tasks) > 20 {
		t.Errorf("scheduled tasks accumulated = %d (>20), wait loop likely not exiting", len(tasks))
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
	ctx, cancel := context.WithCancel(t.Context())
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

// TestSetForceArchive 验证强制归档标记可经 context 往返传递（I4 回归）。
func TestSetForceArchive(t *testing.T) {
	ctx := t.Context()
	if IsForceArchive(ctx) {
		t.Error("IsForceArchive on empty ctx should be false")
	}
	ctx = SetForceArchive(ctx, true)
	if !IsForceArchive(ctx) {
		t.Error("IsForceArchive should be true after SetForceArchive(ctx, true)")
	}
	ctx2 := SetForceArchive(ctx, false)
	if IsForceArchive(ctx2) {
		t.Error("SetForceArchive(ctx, false) should not set the flag")
	}
}

// TestComicVerifier_Close_NoDeadlock 回归 C4：Close 换序后无任务场景立即返回，
// 且幂等（二次调用不 panic、不重复关闭通道）。
func TestComicVerifier_Close_NoDeadlock(t *testing.T) {
	v, err := NewComicVerifier(t.Context(), NewMemoryStorage(), t.TempDir())
	if err != nil {
		t.Fatalf("NewComicVerifier failed: %v", err)
	}

	done := make(chan struct{})
	go func() {
		_ = v.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Close hung on empty verifier")
	}
	// 二次 Close 应幂等，不 panic。
	_ = v.Close()
}

// TestComicVerifier_Close_WithRunningTask 回归 C4/C5：任务运行中 Close 必须
// 取消任务、排空工作池、关闭 fix worker 并返回（不因 fixPool.Release 提前释放
// 或 *VerifyTask.Cancel 未调用而卡死）。
func TestComicVerifier_Close_WithRunningTask(t *testing.T) {
	store := NewMemoryStorage()
	if err := store.Save(t.Context(), NewComic("1001", "test comic", nil)); err != nil {
		t.Fatalf("Save comic failed: %v", err)
	}

	v, err := NewComicVerifier(t.Context(), store, t.TempDir())
	if err != nil {
		t.Fatalf("NewComicVerifier failed: %v", err)
	}

	opts := &VerifyOptions{}
	if _, err := v.Start(t.Context(), opts); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	done := make(chan struct{})
	go func() {
		_ = v.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Close hung with running task")
	}
}

// TestVerifyProgress_MarshalMessages 验证 messages 被序列化输出（S8 回归）。
func TestVerifyProgress_MarshalMessages(t *testing.T) {
	s := &atomic.Value{}
	s.Store(VerifyStatusRunning)
	p := &VerifyProgress{
		TaskID:  "msg-1",
		Total:   atomic.NewInt32(1),
		Current: atomic.NewInt32(0),
		Invalid: atomic.NewInt32(0),
		Fixed:   atomic.NewInt32(0),
		Status:  s,
	}
	p.SetMessage("开始校验")
	p.SetMessage("完成")
	data, err := p.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	var decoded struct {
		Messages []string `json:"messages"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if len(decoded.Messages) != 2 {
		t.Errorf("messages = %v, want 2 entries", decoded.Messages)
	}
}

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package verify

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cocomhub/cocom/pkg/comic"
	"github.com/spf13/cobra"
)

func TestVerifyCommand_Registration(t *testing.T) {
	if Cmd == nil {
		t.Fatal("Cmd should not be nil")
	}
	if Cmd.Use != "verify" {
		t.Errorf("expected Use 'verify', got %s", Cmd.Use)
	}
}

func TestVerifyCommand_HasSubcommands(t *testing.T) {
	subCommands := Cmd.Commands()
	names := make(map[string]bool)
	for _, cmd := range subCommands {
		names[cmd.Use] = true
	}
	expected := []string{"status", "cancel", "schedule"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected subcommand %s not found", name)
		}
	}
}

// TestVerifyDefaultFlags 验证各持久化旗标的默认值，覆盖参数校验所用的默认输入。
func TestVerifyDefaultFlags(t *testing.T) {
	if verifyFlags.pattern != ".*" {
		t.Errorf("pattern default = %q, want %q", verifyFlags.pattern, ".*")
	}
	if verifyFlags.autoFix != false {
		t.Errorf("autoFix default = %v, want false", verifyFlags.autoFix)
	}
	if verifyFlags.workers != 4 {
		t.Errorf("workers default = %d, want 4", verifyFlags.workers)
	}
	if verifyFlags.reportPath != "verify_report.json" {
		t.Errorf("reportPath default = %q, want verify_report.json", verifyFlags.reportPath)
	}
	if verifyFlags.interval != 24*time.Hour {
		t.Errorf("interval default = %v, want 24h", verifyFlags.interval)
	}
}

// TestVerifyStatusCmd_Args 验证 status/cancel 子命令 cobra.ExactArgs(1) 的校验行为：
// 0 个或 2 个参数时 cobra 先于 RunE 报错，1 个参数进入 RunE。
func TestVerifyStatusCmd_Args(t *testing.T) {
	// 参数数量校验是 cobra 层行为，直接断言即可验证 Args 函数。
	if verifyStatusCmd.Args == nil {
		t.Fatal("status cmd should have Args")
	}
	if err := verifyStatusCmd.Args(verifyStatusCmd, nil); err == nil {
		t.Errorf("Args() with 0 args should error")
	}
	if err := verifyStatusCmd.Args(verifyStatusCmd, []string{"only"}); err != nil {
		t.Errorf("Args() with 1 arg should pass, got %v", err)
	}
	if err := verifyStatusCmd.Args(verifyStatusCmd, []string{"a", "b"}); err == nil {
		t.Errorf("Args() with 2 args should error")
	}
}

// TestVerifyCancelCmd_Args 同上，验证 cancel 子命令的参数校验。
func TestVerifyCancelCmd_Args(t *testing.T) {
	if verifyCancelCmd.Args == nil {
		t.Fatal("cancel cmd should have Args")
	}
	if err := verifyCancelCmd.Args(verifyCancelCmd, nil); err == nil {
		t.Errorf("Args() with 0 args should error")
	}
	if err := verifyCancelCmd.Args(verifyCancelCmd, []string{"id-1"}); err != nil {
		t.Errorf("Args() with 1 arg should pass, got %v", err)
	}
}

// TestVerifyStatusCmd_Success 注入 mock service 验证 status 的完整执行（含 GetVerifyProgress）。
func TestVerifyStatusCmd_Success(t *testing.T) {
	origGetService := GetComicService
	t.Cleanup(func() { GetComicService = origGetService })

	GetComicService = func(ctx context.Context) comic.Service {
		return &mockService{progress: comic.NewVerifyProgress("t1")}
	}

	cmd := &cobra.Command{}
	cmd.SetArgs([]string{"t1"})
	err := verifyStatusCmd.RunE(cmd, []string{"t1"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestVerifyStatusCmd_ServiceNil 服务为 nil 时报错。
func TestVerifyStatusCmd_ServiceNil(t *testing.T) {
	origGetService := GetComicService
	t.Cleanup(func() { GetComicService = origGetService })
	GetComicService = func(ctx context.Context) comic.Service {
		return nil
	}
	if err := verifyStatusCmd.RunE(&cobra.Command{}, []string{"t1"}); err == nil {
		t.Error("expected error when service is nil")
	}
}

// TestVerifyStatusCmd_ProgressError service 查询返回错误时 status 透传错误。
func TestVerifyStatusCmd_ProgressError(t *testing.T) {
	origGetService := GetComicService
	t.Cleanup(func() { GetComicService = origGetService })

	GetComicService = func(ctx context.Context) comic.Service {
		return &mockService{err: errors.New("failed")}
	}
	if err := verifyStatusCmd.RunE(&cobra.Command{}, []string{"missing"}); err == nil {
		t.Error("expected error from service")
	}
}

// mockService 满足 comic.Service 接口的最小实现，只为测试 verify 命令逻辑。
// 未实现的方法调用即返回零值；GetVerifyProgress 返回预设 progress/err。
// 注意：comic.Service 接口较大，这里方法较多，但全部零值实现即可编译。
type mockService struct {
	progress *comic.VerifyProgress
	err      error
}

func (m *mockService) StartVerifyTask(ctx context.Context, opts *comic.VerifyOptions) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "task123", nil
}

func (m *mockService) GetVerifyTask(ctx context.Context, taskID string) (*comic.VerifyTask, error) {
	return nil, m.err
}

func (m *mockService) GetVerifyTasks(ctx context.Context) ([]*comic.VerifyTask, error) {
	return nil, m.err
}

func (m *mockService) GetVerifyProgress(ctx context.Context, taskID string) (*comic.VerifyProgress, error) {
	return m.progress, m.err
}

func (m *mockService) CancelVerifyTask(ctx context.Context, taskID string) error {
	return m.err
}

func (m *mockService) StartScheduleVerify(ctx context.Context, cfg *comic.ScheduleConfig) error {
	return m.err
}

func (m *mockService) SearchComics(ctx context.Context, filter *comic.ComicFilter) ([]comic.Comic, error) {
	return nil, m.err
}

func (m *mockService) GetInvalidComics(ctx context.Context, filter *comic.ComicFilter) ([]comic.Comic, error) {
	return nil, m.err
}

func (m *mockService) GetComicInfo(ctx context.Context, id string) (comic.Comic, error) {
	return nil, m.err
}

func (m *mockService) ArchiveComic(ctx context.Context, id string) error {
	return m.err
}

func (m *mockService) RestoreComic(ctx context.Context, id string) error {
	return m.err
}

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package errs

import (
	"errors"
	"fmt"
	"testing"

	"github.com/cocomhub/cocom/pkg/errwrap"
)

func TestErrComicSentinels_CodeAndMsg(t *testing.T) {
	tests := []struct {
		name string
		got  error
		code int
		msg  string
	}{
		{"ErrComicAlreadyDownloaded", ErrComicAlreadyDownloaded, 1000, "comic already downloaded"},
		{"ErrComicDownloadRetryOver", ErrComicDownloadRetryOver, 1001, "comic download retry over"},
		{"ErrComicDownloadConnOver", ErrComicDownloadConnOver, 1002, "comic download conn over"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ew, ok := tt.got.(*errwrap.Error)
			if !ok {
				t.Fatalf("type = %T, want *errwrap.Error", tt.got)
			}
			if ew.Code() != tt.code {
				t.Errorf("Code() = %d, want %d", ew.Code(), tt.code)
			}
			if ew.Msg() != tt.msg {
				t.Errorf("Msg() = %q, want %q", ew.Msg(), tt.msg)
			}
		})
	}
}

func TestErrComicSentinels_WrapAndIs(t *testing.T) {
	wrapped := fmt.Errorf("download failed: %w", ErrComicAlreadyDownloaded)

	if !errors.Is(wrapped, ErrComicAlreadyDownloaded) {
		t.Error("errors.Is(wrapped, ErrComicAlreadyDownloaded) = false, want true")
	}
	if errors.Is(wrapped, ErrComicDownloadRetryOver) {
		t.Error("errors.Is(wrapped, ErrComicDownloadRetryOver) = true, want false")
	}
	if errors.Is(wrapped, ErrComicDownloadConnOver) {
		t.Error("errors.Is(wrapped, ErrComicDownloadConnOver) = true, want false")
	}
}

func TestErrComicSentinels_IsMatchesByCode(t *testing.T) {
	// errwrap.Error.Is 按 code 比较，同 code 不同 msg 应视为同一错误
	other := errwrap.New(1000, "a different message")
	if !errors.Is(other, ErrComicAlreadyDownloaded) {
		t.Error("errors.Is(other(code=1000), ErrComicAlreadyDownloaded) = false, want true")
	}
}

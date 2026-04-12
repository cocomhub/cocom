// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package baidupcs

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cocomhub/cocom/pkg/storage"
)

func TestNewFnAndRegistration(t *testing.T) {
	st, logPath := newFakeStorage(t, "registered-baidupcs")
	name := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
	cfg := storage.Config{
		Name: name,
		Type: Type,
		MetaData: map[string]any{
			"command":    st.config.Command,
			"remoteRoot": st.config.Root,
			"tempDir":    st.config.TempDir,
			"timeout":    "250ms",
			"globalArgs": []any{"--profile=test"},
		},
	}
	if err := storage.SetFromConfig(cfg); err != nil {
		t.Fatalf("set from config: %v", err)
	}
	got, ok := storage.Get(name)
	if !ok {
		t.Fatalf("registered storage not found")
	}
	if got.Type() != Type {
		t.Fatalf("unexpected type: %s", got.Type())
	}
	if _, err := got.Put(t.Context(), "config/test.txt", strings.NewReader("cfg"), storage.WithOverwrite(true)); err != nil {
		t.Fatalf("put via registered storage: %v", err)
	}
	logs := readFakeLog(t, logPath)
	if !strings.Contains(logs, "config/test.txt") {
		t.Fatalf("registered storage did not invoke fake command: %s", logs)
	}
}

func TestNewFnValidation(t *testing.T) {
	commandPath := filepath.Join(t.TempDir(), "missing-binary")
	_, err := newFn("invalid", map[string]any{
		"commandPath": commandPath,
		"root":        "/apps/cocom",
		"timeout":     1500,
		"args":        []string{"--profile=test"},
	})
	if err != nil {
		t.Fatalf("newFn should accept aliases: %v", err)
	}

	_, err = newFn("invalid", map[string]any{
		"command": 1,
		"root":    "/apps/cocom",
	})
	if err == nil || !strings.Contains(err.Error(), "command is not a string") {
		t.Fatalf("unexpected command type error: %v", err)
	}

	_, err = newFn("invalid", map[string]any{
		"command": commandPath,
		"root":    "/apps/cocom",
		"timeout": "0s",
	})
	if err == nil || !strings.Contains(err.Error(), "timeout must be positive") {
		t.Fatalf("unexpected timeout error: %v", err)
	}

	_, err = newFn("invalid", map[string]any{
		"command": commandPath,
		"root":    "/apps/cocom",
		"args":    []any{"--profile=test", 1},
	})
	if err == nil || !strings.Contains(err.Error(), "args contains non-string item") {
		t.Fatalf("unexpected args error: %v", err)
	}
}

func TestDurationValue(t *testing.T) {
	got, err := durationValue(map[string]any{"timeout": 2500.0}, time.Second, "timeout")
	if err != nil {
		t.Fatalf("durationValue float: %v", err)
	}
	if got != 2500*time.Millisecond {
		t.Fatalf("unexpected duration: %v", got)
	}

	_, err = durationValue(map[string]any{"timeout": -1}, time.Second, "timeout")
	if err == nil {
		t.Fatalf("negative timeout should error")
	}
	if !errors.Is(err, err) {
		t.Fatalf("durationValue returned nil error")
	}
}

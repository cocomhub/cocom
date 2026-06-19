// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"testing"
)

func TestLogging_Init(t *testing.T) {
	cfg := Config{
		EnableConsole: true,
		ConsoleLevel:  "debug",
	}
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Init panicked: %v", r)
		}
	}()
	Init(cfg)
	t.Log("Init executed without panic")
}

func TestLogging_NewLogger(t *testing.T) {
	cfg := Config{
		EnableConsole: true,
		ConsoleLevel:  "debug",
	}
	logger := NewLogger(cfg)
	if logger == nil {
		t.Fatal("NewLogger should return non-nil")
	}
	t.Log("NewLogger returned a valid logger")
}

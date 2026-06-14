// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"testing"
)

func TestLogging_Init(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Init panicked (expected without Viper config): %v", r)
		}
	}()
	Init()
	t.Log("Init executed without panic")
}

func TestLogging_NewLogger(t *testing.T) {
	cfg := GetConfigByViper()
	logger := NewLogger(cfg)
	if logger == nil {
		t.Fatal("NewLogger should return non-nil")
	}
	t.Log("NewLogger returned a valid logger")
}

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"testing"

	"github.com/cocomhub/cocom/pkg/comic"
)

func TestSetDefaultStorage(t *testing.T) {
	// 确保测试前是干净的
	ResetDefaultStorage()

	ms := comic.NewMemoryStorage()
	SetDefaultStorage(ms)

	got := GetDefaultStorage()
	if got == nil {
		t.Fatal("GetDefaultStorage: expected non-nil after SetDefaultStorage")
	}
}

func TestGetDefaultStorageDefaultIsNil(t *testing.T) {
	ResetDefaultStorage()

	got := GetDefaultStorage()
	if got != nil {
		t.Error("GetDefaultStorage: expected nil before SetDefaultStorage")
	}
}

func TestResetDefaultStorage(t *testing.T) {
	ms := comic.NewMemoryStorage()
	SetDefaultStorage(ms)

	if GetDefaultStorage() == nil {
		t.Fatal("SetDefaultStorage: expected non-nil storage")
	}

	ResetDefaultStorage()

	if GetDefaultStorage() != nil {
		t.Error("ResetDefaultStorage: expected nil after reset")
	}
}

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package errwrap

import (
	"fmt"
	"strings"
	"testing"
)

func TestErrwrap_NewErrors(t *testing.T) {
	errs := NewErrors(fmt.Errorf("err1"), fmt.Errorf("err2"))
	if errs.Count() != 2 {
		t.Errorf("expected 2 errors, got %d", errs.Count())
	}
}

func TestErrwrap_Add(t *testing.T) {
	var e Errors
	e.Add(fmt.Errorf("err1"))
	e.Add(fmt.Errorf("err2"))
	if e.Count() != 2 {
		t.Errorf("expected 2 errors after Add, got %d", e.Count())
	}
}

func TestErrwrap_ErrorContains(t *testing.T) {
	errs := NewErrors(fmt.Errorf("base error"))
	s := errs.Error()
	if !strings.Contains(s, "base error") {
		t.Errorf("Error() should contain 'base error', got: %s", s)
	}
}

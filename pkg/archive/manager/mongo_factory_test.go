// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import "testing"

func TestMongoFactoryRegistered(t *testing.T) {
	factory, ok := indexFactories["mongo"]
	if !ok || factory == nil {
		t.Fatalf("mongo factory not registered")
	}
}

func TestFirstConfiguredValue(t *testing.T) {
	if got := firstConfiguredValue("", " value ", "fallback"); got != " value " {
		t.Fatalf("unexpected configured value: %q", got)
	}
}

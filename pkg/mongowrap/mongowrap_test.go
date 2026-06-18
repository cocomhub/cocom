// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package mongowrap

import (
	"testing"

	"github.com/cocomhub/cocom/pkg/errwrap"
)

func TestMongowrap_ErrorSentinels(t *testing.T) {
	err := errwrap.New(10000, "mongo not found")
	if err == nil {
		t.Fatal("errwrap.New should not return nil")
	}
	err2 := errwrap.New(10001, "mongo duplicate")
	if err2 == nil {
		t.Fatal("errwrap.New should not return nil")
	}
	t.Log("Error sentinel types compile")
}

func TestMongowrap_BuildURI(t *testing.T) {
	uri := buildMongoDBURI(Config{
		User:     "test",
		Password: "test",
		Host:     "localhost:27017",
		Database: "test",
	})
	t.Logf("buildMongoDBURI returned: %s", uri)
}

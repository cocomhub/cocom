// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package mongowrap

import (
	"sync"
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
	want := "mongodb://test:test@localhost:27017/test?authSource="
	if uri != want {
		t.Errorf("buildMongoDBURI() = %q, want %q", uri, want)
	}
}

func TestMongowrap_BuildURI_NoUser(t *testing.T) {
	uri := buildMongoDBURI(Config{
		Host:       "10.0.0.1:27017",
		Database:   "cocom",
		AuthSource: "admin",
	})
	want := "mongodb://10.0.0.1:27017/cocom?authSource=admin"
	if uri != want {
		t.Errorf("buildMongoDBURI() = %q, want %q", uri, want)
	}
}

func TestMongowrap_ClientNotInitialized(t *testing.T) {
	// Reset package-level state for this test
	t.Cleanup(func() {
		client = nil
		initErr = nil
		onceInit = sync.Once{}
		initialized.Store(false)
	})

	_, err := Client()
	if err == nil {
		t.Error("Client() should return error when Init() was not called")
	}
	if err.Error() != "mongowrap: Init() must be called before Client()" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestMongowrap_DBNotInitialized(t *testing.T) {
	t.Cleanup(func() {
		client = nil
		initErr = nil
		onceInit = sync.Once{}
		initialized.Store(false)
	})

	_, err := DB("test")
	if err == nil {
		t.Error("DB() should return error when Init() was not called")
	}
}

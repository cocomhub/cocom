// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package mongowrap

import (
	"errors"
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

// TestMongowrap_ClientNeverNilNil 验证 C4 回归：Client() 永远不会返回 (nil, nil)，
// 即不会出现"已初始化但 client 与 initErr 皆空"导致调用方 nil 解引用 panic 的窗口。
// 通过注入包级状态（client=nil + initErr=哨兵 + initialized=true）直接断言，
// 避免真实 Mongo 连接（~5s 延迟 + goroutine 残留），测试快速且确定性。
func TestMongowrap_ClientNeverNilNil(t *testing.T) {
	t.Cleanup(func() {
		client = nil
		initErr = nil
		onceInit = sync.Once{}
		initialized.Store(false)
	})

	sentinel := errors.New("init failed")
	client = nil
	initErr = sentinel
	initialized.Store(true)

	c, cErr := Client()
	if c == nil && cErr == nil {
		t.Fatal("Client() returned (nil, nil) — nil 解引用 panic 风险 (C4)")
	}
	if cErr == nil {
		t.Fatal("Client() returned nil error despite init failure — 错误未传播 (C4)")
	}
	if cErr != sentinel {
		t.Errorf("Client() error = %v, want sentinel %v", cErr, sentinel)
	}
}

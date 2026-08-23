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

// TestMongowrap_ClientNeverNilNil 验证 C4 回归：Client() 永远不会返回 (nil, nil)，
// 即不会出现"已初始化但 client 与 initErr 皆空"导致调用方 nil 解引用 panic 的窗口。
// 该测试通过模拟 initEngine 失败（无真实 Mongo）断言 Init 后 Client 返回 (nil, err)。
func TestMongowrap_ClientNeverNilNil(t *testing.T) {
	t.Cleanup(func() {
		client = nil
		initErr = nil
		onceInit = sync.Once{}
		initialized.Store(false)
	})

	// 用不可达地址触发 initEngine 失败路径（快速 connection refused）
	err := Init(Config{
		Host:       "127.0.0.1:1", // 大概率无服务，快速 connection refused
		User:       "",
		Password:   "",
		Database:   "test",
		AuthSource: "admin",
	})
	// 无论 init 是否成功，Client() 都必须返回 (nil, err) 或 (client, nil)，绝不能 (nil, nil)
	c, cErr := Client()
	if c == nil && cErr == nil {
		t.Fatal("Client() returned (nil, nil) — nil 解引用 panic 风险 (C4)")
	}
	// init 失败时（err != nil），Client() 必须回传非 nil 错误
	if err != nil && cErr == nil {
		t.Errorf("Client() returned nil error despite init failure (%v) — 错误未传播 (C4)", err)
	}
}

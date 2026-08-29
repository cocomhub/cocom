// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package mongowrap

import (
	"context"
	"errors"
	"net/url"
	"testing"
)

// resetState 恢复包级状态，保证用例间不相互污染。
func resetState() {
	mu.Lock()
	defer mu.Unlock()
	client = nil
	initErr = nil
	initialized.Store(false)
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

// TestMongowrap_BuildURI_SpecialChars 表驱动验证 password 中 URI 保留字符
// （@ : / % 等）被正确 percent 转义，不含明文分隔符、可被 mongo-driver
// url.Parse 解析回原值，避免特殊字符破坏 userinfo 结构（旧 url.PathEscape
// 无法转义 '@'，导致 host 解析错位）。
func TestMongowrap_BuildURI_SpecialChars(t *testing.T) {
	cases := []struct {
		name     string
		user     string
		password string
	}{
		{"at-sign", "user@example", "pa@ss"},
		{"colon", "user", "pa:ss"},
		{"slash", "user", "pa/ss"},
		{"percent", "user", "pa%ss"},
		{"all-reserved", "u:ser@x/%", "pa:ss/wo@rd%"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uri := buildMongoDBURI(Config{
				User:       tc.user,
				Password:   tc.password,
				Host:       "h:27017",
				Database:   "db",
				AuthSource: "admin",
			})
			if got := countRune(uri, '@'); got > 2 {
				t.Fatalf("URI has %d @, want <= 2: %s", got, uri)
			}
			parsed, err := url.Parse(uri)
			if err != nil {
				t.Fatalf("url.Parse(%q) failed: %v", uri, err)
			}
			u := parsed.User.Username()
			p, _ := parsed.User.Password()
			if u != tc.user || p != tc.password {
				t.Errorf("round-trip user=%q password=%q, uri=%q", u, p, uri)
			}
		})
	}
}

func countRune(s string, r rune) int {
	n := 0
	for _, c := range s {
		if c == r {
			n++
		}
	}
	return n
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
	t.Cleanup(resetState)

	_, err := Client()
	if err == nil {
		t.Error("Client() should return error when Init() was not called")
	}
	if !errors.Is(err, ErrNotInitialized) {
		t.Errorf("Client() error = %v, want ErrNotInitialized", err)
	}
}

func TestMongowrap_DBNotInitialized(t *testing.T) {
	t.Cleanup(resetState)

	_, err := DB("test")
	if err == nil {
		t.Error("DB() should return error when Init() was not called")
	}
	if !errors.Is(err, ErrNotInitialized) {
		t.Errorf("DB() error = %v, want ErrNotInitialized", err)
	}
}

// TestMongowrap_ClientNeverNilNil 验证 C4 回归：Client() 永远不会返回 (nil, nil)，
// 即不会出现“已初始化但 client 与 initErr 皆空”导致调用方 nil 解引用 panic 的窗口。
// 通过注入包级状态直接断言，避免真实 Mongo 连接（~5s 延迟 + goroutine 残留），
// 测试快速且确定性。所有失败路径统一返回 ErrNotInitialized 哨兵。
func TestMongowrap_ClientNeverNilNil(t *testing.T) {
	t.Cleanup(resetState)

	t.Run("initErr non-nil", func(t *testing.T) {
		resetState()
		mu.Lock()
		client = nil
		initErr = errors.New("init failed")
		initialized.Store(true)
		mu.Unlock()

		c, cErr := Client()
		if c != nil {
			t.Errorf("Client() returned non-nil client %v, want nil", c)
		}
		if cErr == nil {
			t.Fatal("Client() returned nil error despite init failure — 错误未传播 (C4)")
		}
		if !errors.Is(cErr, ErrNotInitialized) {
			t.Errorf("Client() error = %v, want ErrNotInitialized", cErr)
		}
	})

	t.Run("nil guard hit", func(t *testing.T) {
		resetState()
		mu.Lock()
		client = nil
		initErr = nil
		initialized.Store(true)
		mu.Unlock()

		c, cErr := Client()
		if c != nil {
			t.Errorf("Client() returned non-nil client %v, want nil", c)
		}
		if cErr == nil {
			t.Fatal("Client() returned (nil, nil) — nil 解引用 panic 风险 (C4)")
		}
		if !errors.Is(cErr, ErrNotInitialized) {
			t.Errorf("Client() error = %v, want ErrNotInitialized", cErr)
		}
	})
}

// TestMongowrap_InitContextSignature 在编译期锁定 Init 的 context 签名，
// 防止误回退到旧的无 context 版本（所有调用方已显式传入 ctx）。
func TestMongowrap_InitContextSignature(t *testing.T) {
	_ = func(ctx context.Context) error { return Init(ctx, Config{Host: "127.0.0.1:1"}) }
}

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package mongo

import (
	"strings"
	"sync"
	"testing"

	"github.com/cocomhub/cocom/internal/config"
)

// resetMongoState 复位包级惰性访问器状态，保证每个子测试从干净状态出发。
// 注意 sync.Once 在 f panic 时也会通过 defer 置 done=1（Go 实现），因此首次
// DB() 失败后 initDB 已被消费、db 为 nil；子测试间必须手动复位才能独立断言。
func resetMongoState() {
	db = nil
	initDB = sync.Once{}
	comicInfo = nil
	initComicInfo = sync.Once{}
	oneComicInfo = nil
	initOneComicInfo = sync.Once{}
	videoInfo = nil
	initVideoInfo = sync.Once{}
	settings = nil
	initSettings = sync.Once{}
	custom = nil
	initCustom = sync.Once{}
	comicTag = nil
	initComicTag = sync.Once{}
	tagRelation = nil
	initTagRelation = sync.Once{}
}

// TestMongo_AccessorsFailFast 验证 sync.Once 惰性集合访问器在 mongowrap 未初始化时
// 快速失败（panic 带可读错误），而不是静默返回 nil 导致下游解引用 panic。
// 各访问器均经 DB() → mongowrap.DB() 获取 *mongo.Database；无真实 MongoDB 连接时
// 必须抛错并携带可读信息。
func TestMongo_AccessorsFailFast(t *testing.T) {
	config.Reset()
	t.Cleanup(resetMongoState)

	tests := []struct {
		name string
		fn   func() any
	}{
		{"ComicInfo", func() any { return ComicInfo() }},
		{"OneComicInfo", func() any { return OneComicInfo() }},
		{"VideoInfo", func() any { return VideoInfo() }},
		{"Settings", func() any { return Settings() }},
		{"Custom", func() any { return Custom() }},
		{"ComicTag", func() any { return ComicTag() }},
		{"TagRelation", func() any { return TagRelation() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetMongoState()
			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("访问器应在 mongowrap 未初始化时 panic（快速失败），而非返回 nil")
				}
				msg, ok := r.(error)
				if !ok {
					t.Fatalf("panic 值应为 error，got %T: %v", r, r)
				}
				if !strings.Contains(msg.Error(), "failed to get mongo db") {
					t.Errorf("panic 信息应包含 mongowrap 连接错误，got %q", msg.Error())
				}
			}()
			tt.fn()
		})
	}
}

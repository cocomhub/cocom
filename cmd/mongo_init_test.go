// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"testing"

	"github.com/cocomhub/cocom/cmd/verify"
)

// TestVerifyCmd_HasMongoInitHook 锁定 verify 命令的 MongoDB 初始化接线。
//
// 回归背景：mongowrap.Init 全仓只有 3 个调用点，而 verify 命令既不走
// initArchiveManager 也没有自己的初始化钩子；同时 mongowrap.Client() 不再
// 惰性自初始化。结果 verify 的四个子命令全部拿到 ErrNotInitialized，
// 并在 getComicServiceDefault 里报出误导性的「连接数据库失败」。
//
// 这里断言的是「钩子存在」这一必要条件：verify 不需要归档管理器，
// 因此不应改挂 initArchiveManager，而应挂只做 mongowrap.Init 的 initMongo。
func TestVerifyCmd_HasMongoInitHook(t *testing.T) {
	if verify.Cmd.PersistentPreRunE == nil {
		t.Fatal("verify 命令缺少 PersistentPreRunE：mongowrap 不会被初始化，" +
			"verify/status/cancel/schedule 将全部失败")
	}
	if rootCmd.PersistentPreRunE != nil {
		t.Fatal("根命令不应持有 PersistentPreRunE：各命令的初始化钩子必须挂在各自命令上")
	}
}

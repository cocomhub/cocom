// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package download

import (
	"github.com/cavaliergopher/grab/v3"
)

type Task struct {
	Url    string
	Dir    string
	Name   string
	Status *bool
}

type TaskResult struct {
	Task     *Task
	Response *grab.Response
	// Err 记录无法构造 grab.Request 等传输前错误（此时 Response 为 nil）。
	// 调用方应优先判断 Err，再读取 Response.Err()。
	Err error
}

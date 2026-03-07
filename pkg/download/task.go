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
}

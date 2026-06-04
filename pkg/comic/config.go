// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import "github.com/spf13/viper"

func init() {
	// config-doc: comic.verify.concurrent 验证并发协程数
	viper.SetDefault("comic.verify.concurrent", 10)
	// config-doc: comic.verify.task_buffer_size 任务缓冲区大小
	viper.SetDefault("comic.verify.task_buffer_size", 100)
}

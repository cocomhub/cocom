// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"strings"
	"testing"
)

// TestMain 隔离调用者环境中的 COCOM_* 变量。
//
// config.New()/Init() 现在会接线 AutomaticEnv（生产与测试二进制都需要——测试进程
// 不调用 Init()，若 env 接线只放在 Init() 里，CI 注入的 COCOM_MONGO_* 就会全部失效）。
// 代价是本包的「断言默认值」类用例会受调用者环境影响：CI 注入 COCOM_MONGO_PASSWORD
// 会让 TestDefaults_AllKeys/mongo.password 等失败。
//
// 因此这里统一清空 COCOM_*；需要验证环境变量覆盖的用例自行用 t.Setenv 设置。
func TestMain(m *testing.M) {
	for _, kv := range os.Environ() {
		if k, _, ok := strings.Cut(kv, "="); ok && strings.HasPrefix(k, "COCOM_") {
			_ = os.Unsetenv(k)
		}
	}
	os.Exit(m.Run())
}

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package v1 是孤儿包：其下 GetSettings/SetSettings/DelSettings 从未被任何路由
// 注册或 import（经全仓库搜索确认无消费方），原空壳 TestV1_Compiles 已删除。
// 保留本测试文件以满足 make notest 的「所有包都有测试文件」门禁。
package v1

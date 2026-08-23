// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package util

import "strings"

// ShellQuote 返回可在 bash 单引号内安全使用的字面量。
// 规则：整体用单引号包裹；内部单引号按 bash 惯例转义为「单引号-反斜杠-单引号-单引号」序列。
// 对空串返回两个单引号；对含空格、$、反引号、分号、换行等字符的输入均安全，不会发生变量展开或命令注入。
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

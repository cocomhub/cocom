// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package util

import "strings"

func FillIncrNum(s []int, begin int, sep int) []int {
	val := begin
	for i := range s {
		s[i] = val
		val = val + sep
	}
	return s
}

func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

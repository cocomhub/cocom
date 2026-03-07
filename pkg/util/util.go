// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package util

func FillIncrNum(s []int, begin int, sep int) []int {
	val := begin
	for i := range s {
		s[i] = val
		val = val + sep
	}
	return s
}

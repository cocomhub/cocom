// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package util

func SafeSubStr(raw string, left int, right int) string {
	if right <= 0 || right > len(raw) {
		right = len(raw)
	}
	if left < 0 || left > right {
		left = 0
	}
	return raw[left:right]
}

func SplitStrRightBySize(raw string, size int) []string {
	if size <= 0 {
		return []string{raw}
	}

	var result []string
	if size == 1 || len(raw)%size == 0 {
		result = make([]string, len(raw)/size)
	} else {
		result = make([]string, (len(raw)+size)/size)
	}

	i := len(result)
	for i > 0 {
		i--

		if len(raw) < size {
			result[i] = raw
			break
		}

		result[i] = raw[len(raw)-size:]
		raw = raw[:len(raw)-size]
	}
	return result
}

/*
Copyright © 2023 suixibing <suixibing@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
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

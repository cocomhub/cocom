// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package conv

import "encoding/json"

func JSON(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}

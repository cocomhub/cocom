// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package mutex

import (
	"github.com/cocomhub/cocom/pkg/mutex/local"
)

func MutexLock(key string) (func(), error) {
	return local.MutexLock(key)
}

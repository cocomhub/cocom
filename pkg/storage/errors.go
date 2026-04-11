// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage

import "errors"

var (
	ErrInvalidParam     = errors.New("storage: invalid parameter")
	ErrNotFound         = errors.New("storage: not found")
	ErrAlreadyExists    = errors.New("storage: already exists")
	ErrPermissionDenied = errors.New("storage: permission denied")
	ErrTransient        = errors.New("storage: transient error")
)

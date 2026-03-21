// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package mutex

import (
	"context"
)

type LockType string

const (
	LockTypeLocal LockType = "local"
)

type UnlockFunc func()

type Provider interface {
	Lock(ctx context.Context, key string) (UnlockFunc, error)
}

var current Provider = NewLocalProvider()

func Init(p Provider) {
	if p == nil {
		return
	}
	current = p
}

func With(ctx context.Context, key string, fn func()) error {
	unlock, err := current.Lock(ctx, key)
	if err != nil {
		return err
	}
	defer unlock()
	fn()
	return nil
}

func Lock(ctx context.Context, key string) (UnlockFunc, error) {
	return current.Lock(ctx, key)
}

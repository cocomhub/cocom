// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package mutex

import (
	"context"
	"sync"
	"sync/atomic"
)

type LocalProvider struct {
	m sync.Map
}

func NewLocalProvider() *LocalProvider {
	return &LocalProvider{}
}

type localLocker struct {
	mu    sync.Mutex
	count atomic.Int32
}

func (p *LocalProvider) Lock(ctx context.Context, key string) (UnlockFunc, error) {
	val, _ := p.m.LoadOrStore(key, &localLocker{})
	n := val.(*localLocker)
	n.count.Add(1)
	n.mu.Lock()
	return func() {
		if n.count.Add(-1) == 0 {
			p.m.Delete(key)
		}
		n.mu.Unlock()
	}, nil
}

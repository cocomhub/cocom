// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package mutex

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

type LocalProvider struct {
	mu sync.Mutex // registry 互斥锁：串行化「count 增删 + Delete 决策」，修 map 清理竞态
	m  sync.Map
}

func NewLocalProvider() *LocalProvider {
	return &LocalProvider{}
}

// localLocker 是单 key 的锁存根。
// ch 为容量 1 的信号量（代替 sync.Mutex），天然支持 ctx 取消；
// count 记录等待/持有该 key 的 goroutine 数，用于决定何时从 registry 清理。
type localLocker struct {
	ch    chan struct{}
	count atomic.Int32
}

func (p *LocalProvider) Lock(ctx context.Context, key string) (UnlockFunc, error) {
	// 在 registry 锁保护下完成 LoadOrStore + count 递增，释放 p.mu 后再等待资源锁，
	// 避免与 unlock 的 Delete 决策形成 ABBA 死锁，也闭合「count 判 0 与 Delete 之间」的竞态窗口。
	p.mu.Lock()
	val, _ := p.m.LoadOrStore(key, &localLocker{ch: make(chan struct{}, 1)})
	n, ok := val.(*localLocker)
	if !ok {
		p.mu.Unlock()
		return nil, fmt.Errorf("mutex: unexpected type %T", val)
	}
	n.count.Add(1)
	p.mu.Unlock()

	select {
	case n.ch <- struct{}{}:
	case <-ctx.Done():
		// 等待资源时被取消：回滚 count，可能触发清理。
		p.mu.Lock()
		if n.count.Add(-1) == 0 {
			p.m.Delete(key)
		}
		p.mu.Unlock()
		return nil, ctx.Err()
	}

	// sync.Once 保证 unlock 幂等：二次调用静默忽略（替代原 double-unlock panic）。
	var once sync.Once
	return func() {
		once.Do(func() {
			p.mu.Lock()
			if n.count.Add(-1) == 0 {
				p.m.Delete(key)
			}
			p.mu.Unlock()
			<-n.ch // 释放资源锁，须在 p.mu.Unlock 之后，锁序单一
		})
	}, nil
}

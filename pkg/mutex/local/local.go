// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"errors"
	"sync"
)

var localMutex = &LocalMutex{}

func MutexLock(key string) (func(), error) {
	return localMutex.MutexLock(key)
}

type LocalMutex struct {
	m  sync.Map
	mu sync.Mutex
}

func (mutex *LocalMutex) MutexLock(key string) (func(), error) {
	mutex.mu.Lock()
	defer mutex.mu.Unlock()

	mu, loaded := mutex.m.LoadOrStore(key, &sync.Mutex{})
	if loaded {
		return nil, errors.New("mutex in load")
	}
	mu.(*sync.Mutex).Lock()

	return func() {
		mutex.mu.Lock()
		defer mutex.mu.Unlock()

		mu, _ := mutex.m.LoadAndDelete(key)
		mu.(*sync.Mutex).Unlock()
	}, nil
}

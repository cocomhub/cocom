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
package local

import (
	"errors"
	"sync"
)

var (
	localMutex = &LocalMutex{}
)

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

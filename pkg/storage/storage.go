// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

var (
	newFns   sync.Map
	storages sync.Map
)

type NewFunc func(storageName string, config map[string]any) (Storage, error)

func Register(storageType string, newFn NewFunc) {
	if _, loaded := newFns.LoadOrStore(storageType, newFn); loaded {
		panic(fmt.Errorf("%w: register type %q", ErrAlreadyExists, storageType))
	}
}

func New(storageType, storageName string, config map[string]any) (Storage, error) {
	if storageName == "" {
		return nil, fmt.Errorf("%w: name is empty", ErrInvalidParam)
	}

	v, ok := newFns.Load(storageType)
	if !ok {
		return nil, fmt.Errorf("%w: new type %q", ErrNotFound, storageType)
	}

	newFn := v.(NewFunc)
	s, err := newFn(storageName, config)
	if err != nil {
		return nil, fmt.Errorf("storage: new type %q name %q: %w", storageType, storageName, err)
	}
	return s, nil
}

func Set(storageName string, s Storage) error {
	_, loaded := storages.LoadOrStore(storageName, s)
	if loaded {
		return fmt.Errorf("%w: set name %q", ErrAlreadyExists, storageName)
	}
	return nil
}

func Clear() {
	storages.Range(func(key, value any) bool {
		storages.Delete(key)
		return true
	})
}

func Get(name string) (Storage, bool) {
	s, ok := storages.Load(name)
	if !ok {
		return nil, false
	}
	return s.(Storage), ok
}

func MustGet(name string) Storage {
	s, ok := Get(name)
	if !ok {
		panic(fmt.Errorf("%w: name %q", ErrNotFound, name))
	}
	return s
}

func Path(elem ...string) (string, error) {
	if len(elem) == 0 {
		return "/", nil
	}
	p := filepath.ToSlash(filepath.Join(elem...))
	if strings.HasPrefix(p, "..") {
		return p, fmt.Errorf("storage: key %s is traversal blocked", p)
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p, nil
}

func MustPath(elem ...string) string {
	p, err := Path(elem...)
	if err != nil {
		panic(err)
	}
	return p
}

func URI(s Storage, key ...string) (string, error) {
	path, err := Path(key...)
	if err != nil {
		return "", err
	}
	return s.Type() + "://" + s.Name() + path, nil
}

func MustURI(s Storage, key ...string) string {
	uri, err := URI(s, key...)
	if err != nil {
		panic(err)
	}
	return uri
}

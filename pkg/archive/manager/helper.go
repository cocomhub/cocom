// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"sync/atomic"

	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/cocomhub/cocom/pkg/storage"
)

var global atomic.Value

func init() {
	Set(New())
}

func Set(m Manager) {
	global.Store(newHelper(m))
}

func Get() Manager {
	return GetHelper().Manager()
}

func GetHelper() Helper {
	return global.Load().(Helper)
}

func newHelper(m Manager) Helper {
	return &helper{m: m}
}

type Helper interface {
	ApplyRetention(ctx context.Context, f IndexFilter) (int, error)
	ArchiveAndRegister(ctx context.Context, srcDir, destPath string, acfg archive.Config) error
	CheckAndUpdate(ctx context.Context, id int) (ArchiveMeta, error)
	Manager() Manager
	ReplicateToStorage(ctx context.Context, dst storage.Storage, prefix string, f IndexFilter) (int, error)
}

type helper struct {
	m Manager
}

func (h *helper) Manager() Manager {
	if h.m == nil {
		return Get()
	}
	return h.m
}

func ApplyRetention(ctx context.Context, f IndexFilter) (int, error) {
	return GetHelper().ApplyRetention(ctx, f)
}

func ArchiveAndRegister(ctx context.Context, srcDir, destPath string, acfg archive.Config) error {
	return GetHelper().ArchiveAndRegister(ctx, srcDir, destPath, acfg)
}

func CheckAndUpdate(ctx context.Context, id int) (ArchiveMeta, error) {
	return GetHelper().CheckAndUpdate(ctx, id)
}

func ReplicateToStorage(ctx context.Context, dst storage.Storage, prefix string, f IndexFilter) (int, error) {
	return GetHelper().ReplicateToStorage(ctx, dst, prefix, f)
}

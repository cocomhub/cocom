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
	Archive(ctx context.Context, srcDir, destPath string, replicate bool, replicatePrefix string, acfg archive.Config) (*ArchiveMeta, error)
	CheckAndUpdate(ctx context.Context, id int) (*ArchiveMeta, error)
	Manager() Manager
	Replicate(ctx context.Context, dst storage.Storage, prefix string, f IndexFilter) (int, error)
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

func Archive(ctx context.Context, srcDir, destPath string, replicate bool, replicatePrefix string, acfg archive.Config) (*ArchiveMeta, error) {
	return GetHelper().Archive(ctx, srcDir, destPath, replicate, replicatePrefix, acfg)
}

func CheckAndUpdate(ctx context.Context, id int) (*ArchiveMeta, error) {
	return GetHelper().CheckAndUpdate(ctx, id)
}

func Replicate(ctx context.Context, dst storage.Storage, prefix string, f IndexFilter) (int, error) {
	return GetHelper().Replicate(ctx, dst, prefix, f)
}

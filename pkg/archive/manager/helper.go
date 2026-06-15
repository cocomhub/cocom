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
	h, _ := global.Load().(Helper)
	return h
}

func newHelper(m Manager) Helper {
	return &helper{m: m}
}

type Helper interface {
	ApplyRetention(ctx context.Context, f IndexFilter) (int, error)
	Archive(ctx context.Context, srcDir, destPath string, replicate bool, replicatePrefix string, acfg archive.Config) (*ArchiveMeta, error)
	Check(ctx context.Context, id int, force bool) (*ArchiveMeta, error)
	Manager() Manager
	ReplicateMore(ctx context.Context, dst storage.Storage, prefix string, f IndexFilter) ([]ArchiveMeta, error)
	Replicate(ctx context.Context, dst storage.Storage, prefix string, meta *ArchiveMeta) error
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

func Check(ctx context.Context, id int, force bool) (*ArchiveMeta, error) {
	return GetHelper().Check(ctx, id, force)
}

func ReplicateMore(ctx context.Context, dst storage.Storage, prefix string, f IndexFilter) ([]ArchiveMeta, error) {
	return GetHelper().ReplicateMore(ctx, dst, prefix, f)
}

func Replicate(ctx context.Context, dst storage.Storage, prefix string, meta *ArchiveMeta) error {
	return GetHelper().Replicate(ctx, dst, prefix, meta)
}

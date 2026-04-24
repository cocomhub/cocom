// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"time"
)

type ObjectMeta struct {
	Key     string
	Size    int64
	ETag    string
	ModTime time.Time
}

func (o ObjectMeta) String() string {
	m := ObjectMeta{}
	if o == m {
		return "(not found)"
	}
	return fmt.Sprintf("key:%s size:%d time:%s etag:%s", o.Key, o.Size, o.ModTime, o.ETag)
}

type Checksum struct {
	Algorithm string `json:"algorithm" bson:"algorithm"`
	Value     string `json:"value" bson:"value"`
}

type StorageLocator struct {
	Backend string `json:"backend,omitempty" bson:"backend"`
	Key     string `json:"key,omitempty" bson:"key"`
	ReplicaHealth
}

func NewHealthy(healthy bool) ReplicaHealth {
	return ReplicaHealth{
		Healthy:   healthy,
		CheckedAt: time.Now(),
	}
}

type ReplicaHealth struct {
	Healthy   bool      `json:"healthy" bson:"healthy"`
	CheckedAt time.Time `json:"checked_at" bson:"checked_at"`
}

type PutOptions struct {
	Overwrite bool
	Hash      hash.Hash
}

type Option func(*PutOptions)

func WithOverwrite(v bool) Option {
	return func(o *PutOptions) { o.Overwrite = v }
}

func WithSHA256() Option {
	return func(o *PutOptions) { o.Hash = sha256.New() }
}

func WithMD5() Option {
	return func(o *PutOptions) { o.Hash = md5.New() }
}

func calcHash(h hash.Hash, r io.Reader) (string, int64, error) {
	if h == nil {
		// pass-through size counting without hash
		var n int64
		buf := make([]byte, 32*1024)
		for {
			m, err := r.Read(buf)
			if m > 0 {
				n += int64(m)
			}
			if err == io.EOF {
				return "", n, nil
			}
			if err != nil {
				return "", n, err
			}
		}
	}
	w := &countingWriter{h: h}
	if _, err := io.Copy(w, r); err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), w.n, nil
}

type countingWriter struct {
	h hash.Hash
	n int64
}

func (w *countingWriter) Write(p []byte) (int, error) {
	n, err := w.h.Write(p)
	w.n += int64(n)
	return n, err
}

type Storage interface {
	Type() string
	Name() string
	Put(ctx context.Context, key string, r io.Reader, opts ...Option) (*ObjectMeta, error)
	Get(ctx context.Context, key string) (io.ReadCloser, *ObjectMeta, error)
	Stat(ctx context.Context, key string) (*ObjectMeta, error)
	Exists(ctx context.Context, key string) (bool, error)
	List(ctx context.Context, prefix string) ([]ObjectMeta, error)
	Delete(ctx context.Context, key string) error
	Copy(ctx context.Context, srcKey, dstKey string, opts ...Option) (*ObjectMeta, error)
	Move(ctx context.Context, srcKey, dstKey string, opts ...Option) (*ObjectMeta, error)
}

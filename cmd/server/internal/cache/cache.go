// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/allegro/bigcache/v3"
)

// ErrCacheNotInitialized 表示缓存未初始化（Init 未调用或上次 Init 失败后 Close）。
// 作为稳定哨兵供 errors.Is 判断，替代对 nil 的内部句柄直接解引用 panic。
var ErrCacheNotInitialized = errors.New("cache: not initialized")

var cache *bigcache.BigCache

// SetDefault 已迁移到 internal/config/manager.go setDefaultsOn()

func Init(ctx context.Context, evictionInterval, cleanInterval time.Duration) {
	// 重建语义：若已有旧实例先关闭，避免未关闭的旧 cache 句柄占用资源、
	// 以及并发 Get/Set 命中被换掉的实例（重建是幂等的，可重复调用）。
	// 注意：Close 期间的并发调用在调用方侧负责串行化，本包不额外加锁（沿用单一全局句柄约定）。
	if cache != nil {
		_ = cache.Close()
	}

	slog.InfoContext(ctx, "[cache] config", slog.Duration("evictionInterval", evictionInterval), slog.Duration("cleanInterval", cleanInterval))

	cfg := bigcache.DefaultConfig(evictionInterval)
	cfg.CleanWindow = cleanInterval

	var err error
	cache, err = bigcache.New(ctx, cfg)
	if err != nil {
		// 保留原语义：初始化失败 panic，避免缓存层静默不可用被业务忽略。
		panic(any(err))
	}
}

// Cache 返回当前缓存实例；未初始化时返回 nil（调用方可据此跳过缓存）。
func Cache() *bigcache.BigCache {
	return cache
}

// Close 关闭缓存实例；未初始化时返回 ErrCacheNotInitialized，不 panic。
// 注意：Close 后再次 Init 会重建新实例（见 Init），不会复用旧句柄。
func Close() error {
	if cache == nil {
		return ErrCacheNotInitialized
	}
	c := cache
	cache = nil // 先置 nil 再关闭，防止并发调用方在关闭期间拿到并解引用已关闭（或正在替换）的句柄
	return c.Close()
}

func Get(key string, entry any) error {
	if cache == nil {
		return ErrCacheNotInitialized
	}
	data, err := cache.Get(key)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, entry)
}

func GetWithInfo(key string, entry any) (*bigcache.Response, error) {
	if cache == nil {
		return nil, ErrCacheNotInitialized
	}
	data, response, err := cache.GetWithInfo(key)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, entry)
	return &response, err
}

func Set(key string, entry any) error {
	if cache == nil {
		return ErrCacheNotInitialized
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return cache.Set(key, data)
}

func Delete(key string) error {
	if cache == nil {
		return ErrCacheNotInitialized
	}
	err := cache.Delete(key)
	if errors.Is(err, bigcache.ErrEntryNotFound) {
		return nil
	}
	return err
}

func Reset() error {
	if cache == nil {
		return ErrCacheNotInitialized
	}
	return cache.Reset()
}

func ResetStats() error {
	if cache == nil {
		return ErrCacheNotInitialized
	}
	return cache.ResetStats()
}

func Len() int {
	if cache == nil {
		return 0
	}
	return cache.Len()
}

func Capacity() int {
	if cache == nil {
		return 0
	}
	return cache.Capacity()
}

func Stats() bigcache.Stats {
	if cache == nil {
		return bigcache.Stats{}
	}
	return cache.Stats()
}

func KeyMetadata(key string) bigcache.Metadata {
	if cache == nil {
		return bigcache.Metadata{}
	}
	return cache.KeyMetadata(key)
}

func Iterator() *bigcache.EntryInfoIterator {
	if cache == nil {
		return nil
	}
	return cache.Iterator()
}

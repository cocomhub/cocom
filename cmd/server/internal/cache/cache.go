// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/cocomhub/cocom/pkg/clog"

	"github.com/spf13/viper"
)

var cache *bigcache.BigCache

func init() {
	viper.SetDefault("cocom.cache.cleanInterval", 1*time.Minute)
	viper.SetDefault("cocom.cache.evictionInterval", 10*time.Minute)
}

func Init(ctx context.Context) {
	evictionInterval := viper.GetDuration("cocom.cache.evictionInterval")
	cleanInterval := viper.GetDuration("cocom.cache.cleanInterval")
	clog.Infof(ctx, "[cache] config: evictionInterval[%v] cleanInterval[%v]", evictionInterval, cleanInterval)

	cfg := bigcache.DefaultConfig(evictionInterval)
	cfg.CleanWindow = cleanInterval

	var err error
	cache, err = bigcache.New(ctx, cfg)
	if err != nil {
		panic(any(err))
	}
}

func Cache() *bigcache.BigCache {
	return cache
}

func Close() error {
	return cache.Close()
}

func Get(key string, entry interface{}) error {
	data, err := cache.Get(key)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, entry)
}

func GetWithInfo(key string, entry interface{}) (*bigcache.Response, error) {
	data, response, err := cache.GetWithInfo(key)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, entry)
	return &response, err
}

func Set(key string, entry interface{}) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return cache.Set(key, data)
}

func Delete(key string) error {
	err := cache.Delete(key)
	if errors.Is(err, bigcache.ErrEntryNotFound) {
		return nil
	}
	return err
}

func Reset() error {
	return cache.Reset()
}

func ResetStats() error {
	return cache.ResetStats()
}

func Len() int {
	return cache.Len()
}

func Capacity() int {
	return cache.Capacity()
}

func Stats() bigcache.Stats {
	return cache.Stats()
}

func KeyMetadata(key string) bigcache.Metadata {
	return cache.KeyMetadata(key)
}

func Iterator() *bigcache.EntryInfoIterator {
	return cache.Iterator()
}

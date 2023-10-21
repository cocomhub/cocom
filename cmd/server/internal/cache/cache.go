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
package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/suixibing/cocom/pkg/clog"

	"github.com/spf13/viper"
)

var (
	cache *bigcache.BigCache
)

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
	return cache.Delete(key)
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

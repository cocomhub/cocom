//go:build memory_storage_integration

package setting

import (
	"context"
	"sync"
)

const (
	SettingKeyType string = "type"
	SettingKeyKey  string = "key"
	SettingKeyVal  string = "val"
)

var (
	memMu     sync.Mutex
	memStores = map[string]map[string]interface{}{}
)

func GetSettings(ctx context.Context, settingType string, keys ...string) (map[string]interface{}, error) {
	memMu.Lock()
	defer memMu.Unlock()
	store, ok := memStores[settingType]
	if !ok {
		return map[string]interface{}{}, nil
	}
	// if keys is empty or first is empty string, return all
	if len(keys) == 0 || keys[0] == "" {
		out := map[string]interface{}{}
		for k, v := range store {
			out[k] = v
		}
		return out, nil
	}
	out := map[string]interface{}{}
	for _, k := range keys {
		if v, ok := store[k]; ok {
			out[k] = v
		}
	}
	return out, nil
}

func SetSettings(ctx context.Context, settingType string, kvs map[string]interface{}) error {
	memMu.Lock()
	defer memMu.Unlock()
	store, ok := memStores[settingType]
	if !ok {
		store = map[string]interface{}{}
		memStores[settingType] = store
	}
	for k, v := range kvs {
		store[k] = v
	}
	return nil
}

func DelSettings(ctx context.Context, settingType string, keys ...string) (int64, error) {
	memMu.Lock()
	defer memMu.Unlock()
	store, ok := memStores[settingType]
	if !ok {
		return 0, nil
	}
	var deleted int64
	// if keys empty or first empty string, delete all under type
	if len(keys) == 0 || keys[0] == "" {
		deleted = int64(len(store))
		delete(memStores, settingType)
		return deleted, nil
	}
	for _, k := range keys {
		if _, ok := store[k]; ok {
			delete(store, k)
			deleted++
		}
	}
	if len(store) == 0 {
		delete(memStores, settingType)
	}
	return deleted, nil
}

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package localfs

import (
	"fmt"

	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/spf13/viper"
)

func init() {
	storage.Register("localfs", newFn)
}

func newFn(storageName string, config map[string]any) (storage.Storage, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}
	root, ok := config["root"].(string)
	if !ok {
		return nil, fmt.Errorf("root is not a string")
	}
	if root == "" {
		return nil, fmt.Errorf("root is empty")
	}
	return New(storageName, root), nil
}

func SetFromViper(keys ...string) error {
	for _, key := range keys {
		if key == "" {
			continue
		}
		err := storage.SetFromConfig(storage.Config{
			Name:     key,
			Type:     "localfs",
			MetaData: map[string]any{"root": viper.GetString(key)},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"github.com/spf13/viper"
)

type Config struct {
	Name     string         `mapstructure:"name"`
	Type     string         `mapstructure:"type"`
	MetaData map[string]any `mapstructure:"metadata"`
}

func SetFromConfig(configs ...Config) error {
	for _, config := range configs {
		fs, err := New(config.Type, config.Name, config.MetaData)
		if err != nil {
			return err
		}
		if err := Set(config.Name, fs); err != nil {
			return err
		}
	}
	return nil
}

const DefaultBackendsKey = "storage.backends"

func SetFromViper(keys ...string) error {
	key := DefaultBackendsKey
	if len(keys) > 0 {
		key = keys[0]
	}
	var configs []Config
	if err := viper.UnmarshalKey(key, &configs); err != nil {
		return err
	}
	if err := SetFromConfig(configs...); err != nil {
		return err
	}
	return nil
}

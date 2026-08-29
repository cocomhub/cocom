// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage

// Config 存储后端配置。
type Config struct {
	Name     string         `mapstructure:"name" json:"name" yaml:"name"`
	Type     string         `mapstructure:"type" json:"type" yaml:"type"`
	MetaData map[string]any `mapstructure:"metadata" json:"metadata" yaml:"metadata"`
}

// SetFromConfig 注册单个存储后端配置。
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

// SetFromConfigs 注册多个存储后端配置。
func SetFromConfigs(configs []Config) error {
	for _, cfg := range configs {
		if err := SetFromConfig(cfg); err != nil {
			return err
		}
	}
	return nil
}

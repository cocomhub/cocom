// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"fmt"

	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("archive.manager.rootDir", "data/archive")
	viper.SetDefault("archive.manager.algorithm", string(archive.TypeDouble))
	viper.SetDefault("archive.manager.index.type", "memory")
	viper.SetDefault("archive.manager.index.fileStoreName", "archive-manager-index")
	viper.SetDefault("archive.manager.index.fileStorePrefix", "archive/index")
}

type Config struct {
	RootDir   string       `mapstructure:"rootDir"`
	Algorithm archive.Type `mapstructure:"algorithm"`
	Index     IndexConfig  `mapstructure:"index"`
}

type IndexConfig struct {
	Type            string `mapstructure:"type"`
	FileStoreName   string `mapstructure:"fileStoreName"`
	FileStorePrefix string `mapstructure:"fileStorePrefix"`
}

func DefaultConfig(keys ...string) Config {
	key := DefaultConfigKey
	if len(keys) > 0 {
		key = keys[0]
	}
	return Config{
		RootDir:   viper.GetString(key + ".rootDir"),
		Algorithm: archive.Type(viper.GetString(key + ".algorithm")),
		Index: IndexConfig{
			Type:            viper.GetString(key + ".index.type"),
			FileStoreName:   viper.GetString(key + ".index.fileStoreName"),
			FileStorePrefix: viper.GetString(key + ".index.fileStorePrefix"),
		},
	}
}

const DefaultConfigKey = "archive.manager"

func SetFromViper(keys ...string) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			panicErr, ok := recovered.(error)
			if ok {
				err = panicErr
				return
			}
			err = fmt.Errorf("%v", recovered)
		}
	}()
	config := DefaultConfig(keys...)
	Set(New(config))
	return nil
}

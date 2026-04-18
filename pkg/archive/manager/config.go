// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"fmt"

	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("archive.manager.algorithm", string(archive.TypeDouble))
	viper.SetDefault("archive.manager.replicates", []string{})
	viper.SetDefault("archive.manager.index.type", "memory")
	viper.SetDefault("archive.manager.index.fileStoreName", "archive-manager-index")
	viper.SetDefault("archive.manager.index.fileStorePrefix", "archive/index")
	viper.SetDefault("archive.manager.index.mongoDatabase", "cocom")
	viper.SetDefault("archive.manager.index.mongoCollection", "archiveInfo")
}

type Config struct {
	Algorithm  archive.Type `mapstructure:"algorithm"`
	Replicates []string     `mapstructure:"replicates"`
	Index      IndexConfig  `mapstructure:"index"`
}

type IndexConfig struct {
	Type            string `mapstructure:"type"`
	FileStoreName   string `mapstructure:"fileStoreName"`
	FileStorePrefix string `mapstructure:"fileStorePrefix"`
	MongoDatabase   string `mapstructure:"mongoDatabase"`
	MongoCollection string `mapstructure:"mongoCollection"`
}

func DefaultConfig(keys ...string) Config {
	key := DefaultConfigKey
	if len(keys) > 0 {
		key = keys[0]
	}
	return Config{
		Algorithm:  archive.Type(viper.GetString(key + ".algorithm")),
		Replicates: viper.GetStringSlice(key + ".replicates"),
		Index: IndexConfig{
			Type:            viper.GetString(key + ".index.type"),
			FileStoreName:   viper.GetString(key + ".index.fileStoreName"),
			FileStorePrefix: viper.GetString(key + ".index.fileStorePrefix"),
			MongoDatabase:   viper.GetString(key + ".index.mongoDatabase"),
			MongoCollection: viper.GetString(key + ".index.mongoCollection"),
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

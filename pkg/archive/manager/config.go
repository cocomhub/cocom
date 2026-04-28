// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"fmt"
	"strings"

	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("archive.manager.algorithm", string(archive.TypeDouble))
	viper.SetDefault("archive.manager.replicates", []string{})
	viper.SetDefault("archive.manager.index.type", "memory")
	viper.SetDefault("archive.manager.index.file_store_name", "archive-manager-index")
	viper.SetDefault("archive.manager.index.file_store_prefix", "archive/index")
	viper.SetDefault("archive.manager.index.mongo_database", "archiveManager")
	viper.SetDefault("archive.manager.index.mongo_collection", "archiveInfo")
}

type Config struct {
	Algorithm  archive.Type `mapstructure:"algorithm"`
	Replicates []string     `mapstructure:"replicates"`
	Index      IndexConfig  `mapstructure:"index"`
}

type IndexConfig struct {
	Type            string `mapstructure:"type"`
	FileStoreName   string `mapstructure:"file_store_name"`
	FileStorePrefix string `mapstructure:"file_store_prefix"`
	MongoDatabase   string `mapstructure:"mongo_database"`
	MongoCollection string `mapstructure:"mongo_collection"`
}

func (c *IndexConfig) GetMongoDatabase(def string) string {
	return firstConfiguredValue(
		c.MongoDatabase,
		def,
	)
}

func (c *IndexConfig) GetMongoCollection(def string) string {
	return firstConfiguredValue(
		c.MongoCollection,
		def,
	)
}

func firstConfiguredValue(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
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
			FileStoreName:   viper.GetString(key + ".index.file_store_name"),
			FileStorePrefix: viper.GetString(key + ".index.file_store_prefix"),
			MongoDatabase:   viper.GetString(key + ".index.mongo_database"),
			MongoCollection: viper.GetString(key + ".index.mongo_collection"),
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

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/cocomhub/cocom/pkg/util"
	"github.com/spf13/viper"
)

// SetDefault 已迁移到 internal/config/config.go setDefaults()

type Config struct {
	Algorithm          archive.Type `mapstructure:"algorithm"`
	MetaRecordFileList bool         `mapstructure:"meta_record_file_list"`
	Replicates         []string     `mapstructure:"replicates"`
	Index              IndexConfig  `mapstructure:"index"`
}

type IndexConfig struct {
	Type            string `mapstructure:"type"`
	FileStoreName   string `mapstructure:"file_store_name"`
	FileStorePrefix string `mapstructure:"file_store_prefix"`
	MongoDatabase   string `mapstructure:"mongo_database"`
	MongoCollection string `mapstructure:"mongo_collection"`
	MongoPrefix     string `mapstructure:"mongo_prefix"`
	MongoIDField    string `mapstructure:"mongo_id_field"`
	MongoNameField  string `mapstructure:"mongo_name_field"`
}

func (c *IndexConfig) GetMongoDatabase(def string) string {
	return util.FirstNonEmpty(
		c.MongoDatabase,
		def,
	)
}

func (c *IndexConfig) GetMongoCollection(def string) string {
	return util.FirstNonEmpty(
		c.MongoCollection,
		def,
	)
}

func DefaultConfig(keys ...string) Config {
	key := DefaultConfigKey
	if len(keys) > 0 {
		key = keys[0]
	}
	return Config{
		Algorithm:          archive.Type(viper.GetString(key + ".algorithm")),
		MetaRecordFileList: viper.GetBool(key + ".meta_record_file_list"),
		Replicates:         viper.GetStringSlice(key + ".replicates"),
		Index: IndexConfig{
			Type:            viper.GetString(key + ".index.type"),
			FileStoreName:   viper.GetString(key + ".index.file_store_name"),
			FileStorePrefix: viper.GetString(key + ".index.file_store_prefix"),
			MongoDatabase:   viper.GetString(key + ".index.mongo_database"),
			MongoCollection: viper.GetString(key + ".index.mongo_collection"),
			MongoPrefix:     viper.GetString(key + ".index.mongo_prefix"),
			MongoIDField:    viper.GetString(key + ".index.mongo_id_field"),
			MongoNameField:  viper.GetString(key + ".index.mongo_name_field"),
		},
	}
}

const DefaultConfigKey = "archive.manager"

func SetFromViper(keys ...string) error {
	config := DefaultConfig(keys...)
	m, err := tryNew(config)
	if err != nil {
		return err
	}
	Set(m)
	return nil
}

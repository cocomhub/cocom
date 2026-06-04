// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/cocomhub/cocom/pkg/util"
	"github.com/spf13/viper"
)

func init() {
	// config-doc: archive.manager.algorithm 存档算法类型
	viper.SetDefault("archive.manager.algorithm", string(archive.TypeDouble))
	// config-doc: archive.manager.meta_record_file_list 是否记录文件列表
	viper.SetDefault("archive.manager.meta_record_file_list", false)
	// config-doc: archive.manager.replicates 副本存储后端名称列表
	viper.SetDefault("archive.manager.replicates", []string{})
	// config-doc: archive.manager.index.type 索引类型
	viper.SetDefault("archive.manager.index.type", "memory")
	// config-doc: archive.manager.index.file_store_name 文件存储后端名称
	viper.SetDefault("archive.manager.index.file_store_name", "archive-manager-index")
	// config-doc: archive.manager.index.file_store_prefix 文件存储 key 前缀
	viper.SetDefault("archive.manager.index.file_store_prefix", "archive/index")
	// config-doc: archive.manager.index.mongo_database MongoDB 索引数据库名
	viper.SetDefault("archive.manager.index.mongo_database", "archiveManager")
	// config-doc: archive.manager.index.mongo_collection MongoDB 索引集合名
	viper.SetDefault("archive.manager.index.mongo_collection", "archiveInfo")
	// config-doc: archive.manager.index.mongo_prefix MongoDB key 前缀
	viper.SetDefault("archive.manager.index.mongo_prefix", "")
	// config-doc: archive.manager.index.mongo_id_field MongoDB ID 字段名
	viper.SetDefault("archive.manager.index.mongo_id_field", "id")
	// config-doc: archive.manager.index.mongo_name_field MongoDB 名称字段名
	viper.SetDefault("archive.manager.index.mongo_name_field", "name")
}

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

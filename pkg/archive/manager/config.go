// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import "github.com/cocomhub/cocom/pkg/archive"

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
	if c.MongoDatabase != "" {
		return c.MongoDatabase
	}
	return def
}

func (c *IndexConfig) GetMongoCollection(def string) string {
	if c.MongoCollection != "" {
		return c.MongoCollection
	}
	return def
}

// SetFromViper 通过传入 Config 结构体设置全局归档管理器。
// 调用方负责从配置系统读取值后构造 Config。
func SetFromViper(cfg Config) error {
	m, err := tryNew(cfg)
	if err != nil {
		return err
	}
	Set(m)
	return nil
}

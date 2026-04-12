// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"strings"

	"github.com/cocomhub/cocom/pkg/mongowrap"
	"github.com/spf13/viper"
)

func init() {
	RegisterIndexStoreFactory("mongo", func(cfg IndexConfig) IndexStore {
		_ = cfg
		return NewComicInfoArchiveIndexStore(mongowrap.DB(mongoDatabaseName()).Collection(mongoComicInfoCollectionName()))
	})
}

func mongoDatabaseName() string {
	return firstConfiguredValue(
		viper.GetString("comic.mongo.database"),
		viper.GetString("mongo.database"),
		"cocom",
	)
}

func mongoComicInfoCollectionName() string {
	return firstConfiguredValue(
		viper.GetString("comic.mongo.collections.comicInfo"),
		"comicInfo",
	)
}

func firstConfiguredValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

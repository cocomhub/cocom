// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package mongowrap

// Config MongoDB 连接配置。
// mapstructure 标签对齐 viper 键路径（如 mongo.host → Config.Host）。
type Config struct {
	User       string `mapstructure:"user" json:"user" yaml:"user"`
	Password   string `mapstructure:"password" json:"password" yaml:"password"`
	Host       string `mapstructure:"host" json:"host" yaml:"host"`
	Database   string `mapstructure:"database" json:"database" yaml:"database"`
	AuthSource string `mapstructure:"authSource" json:"authSource" yaml:"authSource"`
}

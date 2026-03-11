// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"compress/gzip"

	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("cocom.storage.path", "/data/cocom/data/gallery")
	viper.SetDefault("cocom.archive.path", "/data/cocom/data/archive")
	viper.SetDefault("cocom.archive.password", "")
	viper.SetDefault("cocom.archive.cmd", "7z")
	viper.SetDefault("cocom.archive.algorithm", "double")
	viper.SetDefault("archive.algorithm.single.concurrency", 4)
	viper.SetDefault("archive.algorithm.double.concurrency", 4)
	viper.SetDefault("server.access_log.patterns", []string{"/debug", "/api", "/v1", "/v2"})
	// server 中间件配置默认值（默认关闭）
	viper.SetDefault("server.cors.enabled", false)
	viper.SetDefault("server.cors.allow_origins", "*")
	viper.SetDefault("server.cors.allow_methods", "GET,POST,PUT,DELETE,OPTIONS")
	viper.SetDefault("server.cors.allow_headers", "*")
	viper.SetDefault("server.gzip.enabled", false)
	viper.SetDefault("server.gzip.level", gzip.BestSpeed)
	viper.SetDefault("server.ratelimit.enabled", false)
	viper.SetDefault("server.ratelimit.rps", 10)
	viper.SetDefault("server.ratelimit.burst", 20)
	// 调度器配置
	viper.SetDefault("server.scheduler.enabled", false)
	viper.SetDefault("server.scheduler.timezone", "Local")
	viper.SetDefault("server.scheduler.probe_comic.enabled", false)
	viper.SetDefault("server.scheduler.probe_comic.name", "ProbeComic")
	viper.SetDefault("server.scheduler.probe_comic.cron", "0 */10 * * * *")
	viper.SetDefault("server.scheduler.probe_comic.tags", []string{"probe", "comic"})
}

func GetSaveRoot() string {
	return viper.GetString("cocom.storage.path")
}

func GetArchiveRoot() string {
	return viper.GetString("cocom.archive.path")
}

func GetArchiveTempRoot() string {
	return viper.GetString("cocom.archive.temp_path")
}

func GetArchivePassword() string {
	return viper.GetString("cocom.archive.password")
}

func GetArchiveCmd() string {
	return viper.GetString("cocom.archive.cmd")
}

func GetArchiveAlgorithm() string {
	return viper.GetString("cocom.archive.algorithm")
}

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"slices"
	"strings"
	"time"
)

// mongoIndexTypes 是 pkg/archive/manager 注册的 mongo 系索引类型。
// 与 pkg/archive/manager/factory.go 保持一致。
var mongoIndexTypes = []string{"mongo", "mongo-cocom", "mongo-comicInfo"}

// validIndexTypes 是 pkg/archive/manager 注册的全部索引类型。
// 与 pkg/archive/manager/factory.go init() 保持一致。
var validIndexTypes = []string{"", "memory", "file", "mongo", "mongo-cocom", "mongo-comicInfo"}

// IsMongoIndexType 判断 index 类型是否属于 mongo 系（需要在装配归档管理器前初始化 MongoDB 连接）。
func IsMongoIndexType(typ string) bool {
	return slices.Contains(mongoIndexTypes, typ)
}

// Validate 校验配置的语义合法性。
// 返回错误时调用方决定 fail-fast 策略（server/ar/tools 的 initArchiveManager 返回错误）。
func (c *Config) Validate() error {
	// archive.manager.index.type 必须是已注册类型
	typ := c.Archive.Manager.Index.Type
	if !slices.Contains(validIndexTypes, typ) {
		return fmt.Errorf("config: invalid key %q: %q (valid: %v)",
			"archive.manager.index.type", typ, validIndexTypes)
	}

	// mongo-comicInfo 索引归一：语义上它必然写业务 comicInfo 集合（factory.go 的
	// NewComicInfoArchiveIndexStore 固定使用 GetMongoCollection("comicInfo")），
	// 因此未显式配置 db/collection 时归一为业务 Mongo 默认值，避免落错集合。
	if typ == "mongo-comicInfo" {
		if isZeroString(c.Archive.Manager.Index.MongoDatabase) {
			c.Archive.Manager.Index.MongoDatabase = c.Mongo.Database
		}
		if isZeroString(c.Archive.Manager.Index.MongoCollection) {
			c.Archive.Manager.Index.MongoCollection = c.Comic.Mongo.Collections.ComicInfo
		}
		if isZeroString(c.Archive.Manager.Index.MongoDatabase) {
			c.Archive.Manager.Index.MongoDatabase = "cocom"
		}
		if isZeroString(c.Archive.Manager.Index.MongoCollection) {
			c.Archive.Manager.Index.MongoCollection = "comicInfo"
		}
	}

	// 归档算法并发数必须 > 0
	if c.Cocom.Archive.Algorithm.Single.Concurrency <= 0 {
		return fmt.Errorf("config: invalid key %q: %d (want > 0)",
			"cocom.archive.algorithm.single.concurrency", c.Cocom.Archive.Algorithm.Single.Concurrency)
	}
	if c.Cocom.Archive.Algorithm.Double.Concurrency <= 0 {
		return fmt.Errorf("config: invalid key %q: %d (want > 0)",
			"cocom.archive.algorithm.double.concurrency", c.Cocom.Archive.Algorithm.Double.Concurrency)
	}

	// server.listen.http.addr：含冒号时必须可 SplitHostPort（裸 host 允许，serve.go -p 会补端口）
	if addr := c.Server.Listen.HTTP.Addr; strings.Contains(addr, ":") {
		if _, _, err := net.SplitHostPort(addr); err != nil {
			return fmt.Errorf("config: invalid key %q: %v", "server.listen.http.addr", err)
		}
	}

	// server.cors.allow_origins：逐项校验（整体 * 或合法 http/https 来源），
	// 含 * 中缀（如 https://*.example.com）或非 http/https 的 scheme → fail-fast。
	if err := validateAllowOrigins(c.Server.CORS.AllowOrigins); err != nil {
		return err
	}

	// server.shutdown_timeout 必须可解析
	if _, err := time.ParseDuration(c.Server.ShutdownTimeout); err != nil {
		return fmt.Errorf("config: invalid key %q: %v", "server.shutdown_timeout", err)
	}

	// comic.verify.concurrent 必须 > 0
	if c.Comic.Verify.Concurrent <= 0 {
		return fmt.Errorf("config: invalid key %q: %d (want > 0)",
			"comic.verify.concurrent", c.Comic.Verify.Concurrent)
	}

	// download.maxRunning 必须 > 0（负值/0 会触发 downloader.Init 兜底，但配置语义应当显式合法）
	if c.Download.MaxRunning <= 0 {
		return fmt.Errorf("config: invalid key %q: %d (want > 0)",
			"download.maxRunning", c.Download.MaxRunning)
	}

	// mongo 系索引要求 mongo.host 非空
	if IsMongoIndexType(typ) && strings.TrimSpace(c.Mongo.Host) == "" {
		return fmt.Errorf("config: invalid key %q: mongo.host 不能为空（archive.manager.index.type=%q）",
			"mongo.host", typ)
	}

	// mongo.user 非空但 password 为空：本地无认证开发合法，输出 Warn 不阻止启动
	if strings.TrimSpace(c.Mongo.User) != "" && strings.TrimSpace(c.Mongo.Password) == "" {
		slog.Warn("config: mongo.user 非空但 mongo.password 为空——若 MongoDB 开启认证请显式配置口令；本地无认证开发可忽略",
			slog.String("mongo.user", c.Mongo.User),
			slog.String("key", "mongo.password"))
	}

	// cocom.cache.* 必须可解析
	if _, err := time.ParseDuration(c.Cocom.Cache.CleanInterval); err != nil {
		return fmt.Errorf("config: invalid key %q: %v", "cocom.cache.cleanInterval", err)
	}
	if _, err := time.ParseDuration(c.Cocom.Cache.EvictionInterval); err != nil {
		return fmt.Errorf("config: invalid key %q: %v", "cocom.cache.evictionInterval", err)
	}

	return nil
}

// validateAllowOrigins 校验 CORS allow_origins 的每一项。
// 规则：
//   - 整体（trim 后）* 或单项 *：合法，放行所有来源。
//   - 别的项必须可解析为 URL 且 scheme 为 http/https；含 * 中缀（如 https://*.example.com）→ 错误（启动失败）。
//
// 由 Validate fail-fast 前置拦截后，middlewares.CORS 中的 * 中缀 Only if Conditional 保留作防御（已 500）。
func validateAllowOrigins(raw string) error {
	if strings.TrimSpace(raw) == "*" {
		return nil
	}
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			return fmt.Errorf("config: invalid key %q: 空值项", "server.cors.allow_origins")
		}
		if strings.Contains(item, "*") {
			return fmt.Errorf("config: invalid key %q: %q 含 * 中缀通配符不被支持，请用完整域名列表或仅整体 *", "server.cors.allow_origins", item)
		}
		u, err := url.Parse(item)
		// strict://scheme:仅 http/https；host 可缺省由 ServeMux/gin 后续处理，这里仅校验合法性。
		if err != nil || u.Scheme == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return fmt.Errorf("config: invalid key %q: %q 不是合法 http/https 来源", "server.cors.allow_origins", item)
		}
	}
	return nil
}

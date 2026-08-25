// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package mongowrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ErrNotInitialized 表示 mongowrap 未被 Init（或 Init 失败），客户端不可用。
// 作为稳定的哨兵，供调用方 errors.Is 判断，而非依赖脆弱的错误消息匹配。
var ErrNotInitialized = errors.New("mongowrap: not initialized")

var (
	mu     sync.Mutex
	client *mongo.Client
	// initErr 上一次 Init 的失败原因（成功时为 nil）。与 client 在 mu 保护下
	// 一起更新，保证不变式「client 非 nil ⇔ initErr == nil 且 initialized == true」。
	initErr     error
	initialized atomic.Bool
)

// buildMongoDBURI 构造 mongodb:// URI。
// user/password 通过 url.UserPassword 编码 userinfo，保证 @ : / % 等特殊字符
// 不被误解析（旧的 url.PathEscape 无法转义 '@'，会破坏 URI 结构）。
func buildMongoDBURI(cfg Config) string {
	host := cfg.Host
	database := cfg.Database
	authSource := cfg.AuthSource

	if cfg.User == "" {
		return fmt.Sprintf("mongodb://%s/%s?authSource=%s", host, database, authSource)
	}
	userinfo := url.UserPassword(cfg.User, cfg.Password)
	return fmt.Sprintf("mongodb://%s@%s/%s?authSource=%s", userinfo.String(), host, database, authSource)
}

func doConnect(ctx context.Context, cfg Config) (*mongo.Client, error) {
	uri := buildMongoDBURI(cfg)
	slog.InfoContext(ctx, "mongo connecting",
		slog.String("host", cfg.Host),
		slog.String("user", cfg.User),
		slog.String("database", cfg.Database))

	clientOptions := options.Client().ApplyURI(uri)

	c, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		slog.ErrorContext(ctx, "mongo client connect failed", slog.String("errmsg", err.Error()))
		return nil, err
	}

	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()
	if err := c.Ping(pingCtx, nil); err != nil {
		slog.ErrorContext(ctx, "mongo client ping failed", slog.String("errmsg", err.Error()))
		// 契约「client 非 nil ⇔ 无错误」：Ping 失败立即断开，不留
		// "看似已初始化实则不可用" 的 client；WithoutCancel 防止 ctx 取消时
		// Disconnect 半途退出留下连接器 goroutine。
		_ = c.Disconnect(context.WithoutCancel(ctx))
		return nil, err
	}
	slog.InfoContext(ctx, "mongo db connected")
	return c, nil
}

// Init 初始化 MongoDB 连接，可重试：失败后再次调用会重新连接。
// 连接（含 Ping）在锁外完成，仅在状态交换时持锁，避免持锁做网络 I/O。
// 成功后 initialized 置位，Client() 才可用。
func Init(ctx context.Context, cfg Config) error {
	c, err := doConnect(ctx, cfg)
	if err != nil {
		mu.Lock()
		client = nil
		initErr = err
		mu.Unlock()
		return initErr
	}

	mu.Lock()
	client = c
	initErr = nil
	initialized.Store(true)
	mu.Unlock()
	return nil
}

// Client 返回已初始化的 MongoDB 客户端。
// 未初始化、初始化失败或客户端不可用时返回 (nil, ErrNotInitialized)。
func Client() (*mongo.Client, error) {
	if !initialized.Load() {
		return nil, ErrNotInitialized
	}
	mu.Lock()
	defer mu.Unlock()
	if client == nil || initErr != nil {
		return nil, ErrNotInitialized
	}
	// 锁内拷贝指针，避免并发 Init 覆盖后调用方持有被换掉的句柄。
	return client, nil
}

func DB(name string, opts ...*options.DatabaseOptions) (*mongo.Database, error) {
	c, err := Client()
	if err != nil {
		return nil, err
	}
	return c.Database(name, opts...), nil
}

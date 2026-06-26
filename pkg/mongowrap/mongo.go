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

	"github.com/cocomhub/cocom/pkg/logging"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client      *mongo.Client
	initErr     error
	onceInit    sync.Once
	initialized atomic.Bool
)

func buildMongoDBURI(cfg Config) string {
	user := cfg.User
	password := cfg.Password
	host := cfg.Host
	database := cfg.Database
	authSource := cfg.AuthSource

	if user == "" {
		return fmt.Sprintf("mongodb://%s/%s?authSource=%s", host, database, authSource)
	}
	return fmt.Sprintf(
		"mongodb://%s:%s@%s/%s?authSource=%s",
		url.PathEscape(user),
		url.PathEscape(password),
		host,
		database,
		authSource,
	)
}

func initEngine(cfg Config) {
	ctx, cancel := context.WithTimeout(logging.NewTraceCtx("initMongoEngine"), 10*time.Second)
	defer cancel()
	uri := buildMongoDBURI(cfg)
	slog.InfoContext(ctx, "mongo connecting",
		slog.String("host", cfg.Host),
		slog.String("user", cfg.User),
		slog.String("database", cfg.Database))

	clientOptions := options.Client().ApplyURI(uri)

	client, initErr = mongo.Connect(ctx, clientOptions)
	if initErr != nil {
		slog.ErrorContext(ctx, "mongo client connect failed", slog.String("errmsg", initErr.Error()))
		return
	}

	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()
	initErr = client.Ping(pingCtx, nil)
	if initErr != nil {
		slog.ErrorContext(ctx, "mongo client ping failed", slog.String("errmsg", initErr.Error()))
	}
	slog.InfoContext(ctx, "mongo db connected")
}

// Init 初始化 MongoDB 连接。
// 传入 cfg 替代从全局 viper 读取，解耦配置依赖。
func Init(cfg Config) error {
	onceInit.Do(func() {
		initialized.Store(true)
		initEngine(cfg)
	})
	return initErr
}

func Client() (*mongo.Client, error) {
	if !initialized.Load() {
		return nil, errors.New("mongowrap: Init() must be called before Client()")
	}
	return client, initErr
}

func DB(name string, opts ...*options.DatabaseOptions) (*mongo.Database, error) {
	c, err := Client()
	if err != nil {
		return nil, err
	}
	return c.Database(name, opts...), nil
}

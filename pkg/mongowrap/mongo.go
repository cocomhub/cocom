// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package mongowrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/cocomhub/cocom/pkg/logging"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client   *mongo.Client
	initErr  error
	onceInit sync.Once
)

// SetDefault 已迁移到 internal/config/config.go setDefaults()
// 保留空 init() 以保持 import side-effect 兼容。
func init() {}

func buildMongoDBURI() string {
	user := viper.GetString("mongo.user")
	password := viper.GetString("mongo.password")
	host := viper.GetString("mongo.host")
	database := viper.GetString("mongo.database")
	authSource := viper.GetString("mongo.authSource")

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

func initEngine() {
	ctx, cancel := context.WithTimeout(logging.NewTraceCtx("initMongoEngine"), 10*time.Second)
	defer cancel()
	uri := buildMongoDBURI()
	slog.InfoContext(ctx, "mongo connecting",
		slog.String("host", viper.GetString("mongo.host")),
		slog.String("user", viper.GetString("mongo.user")),
		slog.String("database", viper.GetString("mongo.database")))

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

func Init() error {
	onceInit.Do(initEngine)
	return initErr
}

func Client() (*mongo.Client, error) {
	if err := Init(); err != nil {
		return nil, err
	}
	return client, nil
}

func DB(name string, opts ...*options.DatabaseOptions) (*mongo.Database, error) {
	c, err := Client()
	if err != nil {
		return nil, err
	}
	return c.Database(name, opts...), nil
}

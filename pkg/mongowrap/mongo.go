// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package mongowrap

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/cocomhub/cocom/pkg/logging"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client   *mongo.Client
	onceInit sync.Once
)

func init() {
	viper.SetDefault("mongo.user", "cocom")
	viper.SetDefault("mongo.password", "cocom123")
	viper.SetDefault("mongo.host", "localhost:27017")
	viper.SetDefault("mongo.database", "cocom")
	viper.SetDefault("mongo.authSource", "cocom")
}

func buildMongoDBURI() string {
	return fmt.Sprintf("mongodb://%s:%s@%s/%s?authSource=%s",
		viper.GetString("mongo.user"),
		viper.GetString("mongo.password"),
		viper.GetString("mongo.host"),
		viper.GetString("mongo.database"),
		viper.GetString("mongo.authSource"),
	)
}

func initEngine() {
	var err error
	ctx := logging.NewTraceCtx("initMongoEngine")
	uri := buildMongoDBURI()
	slog.InfoContext(ctx, "mongo db uri", slog.String("uri", uri))

	clientOptions := options.Client().ApplyURI(uri)

	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		slog.ErrorContext(ctx, "mongo client connect failed", slog.String("errmsg", err.Error()))
		panic(fmt.Errorf("mongo client connect failed: %w", err))
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		slog.ErrorContext(ctx, "mongo client ping failed", slog.String("errmsg", err.Error()))
		panic(fmt.Errorf("mongo client ping failed: %w", err))
	}
	slog.InfoContext(ctx, "mongo db connected")
}

func Init() {
	go Client()
}

func Client() *mongo.Client {
	onceInit.Do(initEngine)
	return client
}

func DB(name string, opts ...*options.DatabaseOptions) *mongo.Database {
	return Client().Database(name, opts...)
}

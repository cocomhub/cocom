/*
Copyright © 2023 suixibing <suixibing@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package mongowrap

import (
	"context"
	"fmt"
	"sync"

	"github.com/spf13/viper"
	"github.com/suixibing/cocom/pkg/clog"
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
	ctx := clog.NewTraceCtx("initMongoEngine")
	uri := buildMongoDBURI()
	clog.Infof(ctx, "mongo db uri[%s]", uri)

	clientOptions := options.Client().ApplyURI(uri)

	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		clog.Fatalf(ctx, "mongo client connect failed. errmsg: %s", err)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		clog.Fatalf(ctx, "mongo client ping failed. errmsg: %s", err)
	}
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

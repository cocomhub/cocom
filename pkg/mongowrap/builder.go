// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package mongowrap

import (
	"context"

	"github.com/cocomhub/cocom/pkg/conv"
	"github.com/cocomhub/cocom/pkg/errwrap"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrMongoUpdateFailed = errwrap.New(10000, "mongo update failed")
	ErrMongoFindFailed   = errwrap.New(10001, "mongo find failed")
	ErrMongoDeleteFailed = errwrap.New(10002, "mongo delete failed")
	ErrMongoDecodeFailed = errwrap.New(10003, "mongo decode failed")
	ErrMongoCountFailed  = errwrap.New(10004, "mongo count failed")
)

var (
	DefaultOptionLimit int64 = 20
	DefaultOptionSkip  int64 = 0
)

func NewBuilder(collection *mongo.Collection) *Builder {
	return &Builder{
		collection: collection,
		filter:     bson.M{},
		sort:       bson.D{},
		limit:      DefaultOptionLimit,
		skip:       DefaultOptionSkip,
	}
}

type Builder struct {
	collection *mongo.Collection
	filter     bson.M
	sort       bson.D
	limit      int64
	skip       int64
}

func (builder *Builder) FindOptions() *options.FindOptions {
	opts := options.Find()
	if len(builder.sort) != 0 {
		opts.SetSort(builder.sort)
	}
	if builder.limit != 0 {
		opts.SetLimit(builder.limit)
	}
	if builder.skip != 0 {
		opts.SetSkip(builder.skip)
	}
	return opts
}

func (builder *Builder) All(ctx context.Context, info interface{}) error {
	opts := builder.FindOptions()
	cur, err := builder.collection.Find(ctx, builder.filter, opts)
	if cur.Err() != nil {
		return ErrMongoFindFailed.SetIErrF("filter[%s] opts[%s] errmsg[%s]",
			conv.JSON(builder.filter), conv.JSON(opts), cur.Err())
	}

	err = cur.All(ctx, info)
	if err != nil {
		return ErrMongoDecodeFailed.SetIErrF("filter[%s] opts[%s] errmsg[%s]",
			conv.JSON(builder.filter), conv.JSON(opts), err.Error())
	}
	return nil
}

func (builder *Builder) CountOptions() *options.CountOptions {
	opts := options.Count()
	if builder.limit != 0 {
		opts.SetLimit(builder.limit)
	}
	if builder.skip != 0 {
		opts.SetSkip(builder.skip)
	}
	return opts
}

func (builder *Builder) Count(ctx context.Context) (int64, error) {
	opts := builder.CountOptions()
	count, err := builder.collection.CountDocuments(ctx, builder.filter, opts)
	if err != nil {
		return 0, ErrMongoCountFailed.SetIErrF("filter[%s] opts[%s] errmsg[%s]",
			conv.JSON(builder.filter), conv.JSON(opts), err.Error())
	}
	return count, nil
}

func (builder *Builder) Filters(filter ...interface{}) *Builder {
	for i := 0; i+1 < len(filter); i += 2 {
		switch t := filter[i].(type) {
		case string:
			builder.FilterKV(t, filter[i+1])
		default:
			panic(any("filter key must string"))
		}
	}
	return builder
}

func (builder *Builder) FilterKV(key string, val interface{}) *Builder {
	builder.filter[key] = val
	return builder
}

func (builder *Builder) SortKV(key string, val interface{}) *Builder {
	builder.sort = append(builder.sort, bson.E{Key: key, Value: val})
	return builder
}

func (builder *Builder) Aggregate(ctx context.Context, pipeline, info interface{}) error {
	opts := options.Aggregate()
	opts.SetAllowDiskUse(true)
	cur, err := builder.collection.Aggregate(ctx, pipeline, opts)
	if err != nil {
		return ErrMongoFindFailed.SetIErrF("pipeline[%s] opts[%s] errmsg[%s]",
			conv.JSON(pipeline), conv.JSON(opts), err.Error())
	}
	defer cur.Close(ctx)

	err = cur.All(ctx, info)
	if err != nil {
		return ErrMongoDecodeFailed.SetIErrF("pipeline[%s] opts[%s] errmsg[%s]",
			conv.JSON(pipeline), conv.JSON(opts), err.Error())
	}
	return nil
}

func (builder *Builder) Limit(limit int64) *Builder {
	builder.limit = limit
	return builder
}

func (builder *Builder) NoLimit() *Builder {
	builder.limit = 0
	return builder
}

func (builder *Builder) Skip(skip int64) *Builder {
	builder.skip = skip
	return builder
}

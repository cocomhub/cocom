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

func (builder *Builder) All(ctx context.Context, info any) error {
	opts := builder.FindOptions()
	cur, err := builder.collection.Find(ctx, builder.filter, opts)
	if err != nil {
		return ErrMongoFindFailed.SetIErrF("filter[%s] opts[%s] errmsg: %v",
			conv.JSON(builder.filter), conv.JSON(opts), err)
	}
	// 游标关闭使用 WithoutCancel 派生的上下文，保证即使 ctx 已取消，
	// driver 仍有充足机会完成底层连接释放，避免连接泄漏。
	defer cur.Close(context.WithoutCancel(ctx)) //nolint:errcheck

	allErr := cur.All(ctx, info)
	if allErr != nil {
		return ErrMongoDecodeFailed.SetIErrF("filter[%s] opts[%s] errmsg: %v",
			conv.JSON(builder.filter), conv.JSON(opts), allErr)
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
		return 0, ErrMongoCountFailed.SetIErrF("filter[%s] opts[%s] errmsg: %v",
			conv.JSON(builder.filter), conv.JSON(opts), err)
	}
	return count, nil
}

func (builder *Builder) Filters(filter ...any) *Builder {
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

func (builder *Builder) FilterKV(key string, val any) *Builder {
	builder.filter[key] = val
	return builder
}

func (builder *Builder) SortKV(key string, val any) *Builder {
	builder.sort = append(builder.sort, bson.E{Key: key, Value: val})
	return builder
}

func (builder *Builder) Aggregate(ctx context.Context, pipeline, info any) error {
	opts := options.Aggregate()
	opts.SetAllowDiskUse(true)
	cur, err := builder.collection.Aggregate(ctx, pipeline, opts)
	if err != nil {
		return ErrMongoFindFailed.SetIErrF("pipeline[%s] opts[%s] errmsg: %v",
			conv.JSON(pipeline), conv.JSON(opts), err)
	}
	defer cur.Close(context.WithoutCancel(ctx)) //nolint:errcheck

	err = cur.All(ctx, info)
	if err != nil {
		return ErrMongoDecodeFailed.SetIErrF("pipeline[%s] opts[%s] errmsg: %v",
			conv.JSON(pipeline), conv.JSON(opts), err)
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

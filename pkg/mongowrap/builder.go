package mongowrap

import (
	"context"

	"github.com/suixibing/cocom/pkg/conv"
	"github.com/suixibing/cocom/pkg/errwrap"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrMongoUpdateFailed = errwrap.New(10000, "mongo update failed")
	ErrMongoFindFailed   = errwrap.New(10001, "mongo find failed")
	ErrMongoDeleteFailed = errwrap.New(10002, "mongo delete failed")
)

var (
	DefaultFindfilter  = bson.M{}
	DefaultFindsort    = bson.D{}
	DefaultFindOptions = options.Find().SetLimit(10)
)

func NewFindBuilder(collection *mongo.Collection) *FindBuilder {
	return &FindBuilder{
		collection: collection,
		filter:     DefaultFindfilter,
		sort:       DefaultFindsort,
		opts:       DefaultFindOptions,
	}
}

type FindBuilder struct {
	collection *mongo.Collection
	filter     bson.M
	sort       bson.D
	opts       *options.FindOptions
}

func (builder *FindBuilder) All(ctx context.Context, info interface{}) error {
	cur, err := builder.collection.Find(ctx, builder.filter, builder.opts)
	if cur.Err() != nil {
		return ErrMongoFindFailed.SetIErrF("filter[%s] opts[%s] errmsg[%s]",
			conv.JSON(builder.filter), conv.JSON(builder.opts), cur.Err())
	}

	err = cur.All(ctx, info)
	if err != nil {
		return ErrMongoFindFailed.SetIErrF("filter[%s] opts[%s] errmsg[%s]",
			conv.JSON(builder.filter), conv.JSON(builder.opts), err.Error())
	}
	return nil
}

func (builder *FindBuilder) FilterKV(key string, val interface{}) *FindBuilder {
	builder.filter[key] = val
	return builder
}

func (builder *FindBuilder) SortKV(key string, val interface{}) *FindBuilder {
	builder.sort = append(builder.sort, bson.E{Key: key, Value: val})
	builder.opts.SetSort(builder.sort)
	return builder
}

func (builder *FindBuilder) Limit(limit int64) *FindBuilder {
	builder.opts.SetLimit(limit)
	return builder
}

func (builder *FindBuilder) Skip(skip int64) *FindBuilder {
	builder.opts.SetSkip(skip)
	return builder
}

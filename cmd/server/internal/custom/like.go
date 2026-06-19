// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package custom

import (
	"context"

	"github.com/cocomhub/cocom/cmd/server/internal/mongo"
	"github.com/cocomhub/cocom/pkg/conv"
	"github.com/cocomhub/cocom/pkg/mongowrap"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func AddLikeGroup(ctx context.Context, cid int) (err error) {
	if s := defaultStore; s != nil {
		return s.AddLikeGroup(ctx, cid)
	}

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"cid": cid}
	update := bson.M{"$set": bson.M{"like": true}}

	_, err = mongo.Custom().UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return mongowrap.ErrMongoUpdateFailed.SetIErrF("mongo collection update failed. filter[%s] update[%s] opts[%s] errmsg: %s",
			conv.JSON(filter), conv.JSON(update), conv.JSON(opts), err.Error())
	}
	return
}

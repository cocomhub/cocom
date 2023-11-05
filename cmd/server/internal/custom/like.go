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
package custom

import (
	"context"

	"github.com/suixibing/cocom/cmd/server/internal/mongo"
	"github.com/suixibing/cocom/pkg/conv"
	"github.com/suixibing/cocom/pkg/mongowrap"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func AddLikeGroup(ctx context.Context, cid int) (err error) {
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

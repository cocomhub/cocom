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
package comic

import (
	"context"

	"github.com/suixibing/cocom/cmd/server/internal/comic/errs"
	"github.com/suixibing/cocom/cmd/server/internal/comic/mongo"
	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/conv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func UpdateComicInfo(ctx context.Context, cid int, comicInfo map[string]interface{}) error {
	opts := options.Update().SetUpsert(true)
	filter := bson.M{"cid": cid}
	update := bson.M{"$set": comicInfo}
	delete(comicInfo, "_id")

	_, err := mongo.Collection().UpdateOne(ctx, filter, update, opts)
	if err != nil {
		clog.Errorf(ctx, "mongo collection update failed. filter[%s] update[%s] errmsg: %s",
			conv.JSON(filter), conv.JSON(update), err)
		return errs.ErrMongoUpdateFail
	}
	return nil
}

func GetComicInfo(ctx context.Context, cid int) (interface{}, error) {
	opts := options.FindOne()
	filter := bson.M{"cid": cid}

	result := mongo.Collection().FindOne(ctx, filter, opts)
	info := map[string]interface{}{}
	err := result.Decode(&info)
	if err != nil {
		clog.Errorf(ctx, "mongo collection find one failed. filter[%s] errmsg: %s",
			conv.JSON(filter), err)
		return nil, errs.ErrMongoFindOneFail
	}
	return info, nil
}

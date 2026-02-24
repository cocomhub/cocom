package storage

import (
	"context"
	"fmt"

	"github.com/suixibing/cocom/pkg/comic"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoStorage MongoDB 实现的存储
type MongoStorage struct {
	db *mongo.Database
}

// NewMongoStorage 创建 MongoDB 存储实例
func NewMongoStorage(db *mongo.Database) *MongoStorage {
	return &MongoStorage{db: db}
}

// Get 实现 ComicStorage 接口
func (s *MongoStorage) Get(ctx context.Context, id string) (comic.Comic, error) {
	var comic comic.ComicImpl
	err := s.db.Collection("comics").FindOne(ctx, bson.M{"_id": id}).Decode(&comic)
	if err != nil {
		return nil, err
	}
	return &comic, nil
}

// Update 实现 ComicStorage 接口
func (s *MongoStorage) Update(ctx context.Context, obj interface{}) error {
	comic, err := comic.NewComicImplByObject(obj)
	if err != nil {
		return err
	}

	if comic == nil || comic.GetID() == "" {
		return fmt.Errorf("invalid comic info")
	}

	_, err = s.db.Collection("comics").UpdateOne(ctx, bson.M{"cid": comic.GetID()}, comic)
	return err
}

// Find 实现 ComicStorage 接口
func (s *MongoStorage) Find(ctx context.Context, filter *comic.ComicFilter) ([]comic.Comic, error) {
	cursor, err := s.db.Collection("comics").Find(ctx, s.toMongoFilter(filter), &options.FindOptions{
		Sort:  bson.M{"cid": 1},
		Limit: &filter.Limit,
		Skip:  &filter.Skip,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var impls []comic.ComicImpl
	if err := cursor.All(ctx, &impls); err != nil {
		return nil, err
	}

	// 转换为接口类型
	comics := make([]comic.Comic, len(impls))
	for i := range impls {
		comics[i] = &impls[i]
	}
	return comics, nil
}

// FindTotal 列出符合条件的漫画总数
func (s *MongoStorage) FindTotal(ctx context.Context, filter *comic.ComicFilter) (int64, error) {
	return s.db.Collection("comics").CountDocuments(ctx, s.toMongoFilter(filter), &options.CountOptions{
		Limit: &filter.Limit,
		Skip:  &filter.Skip,
	})
}

// FindChannel 列出符合条件的漫画，返回通道
func (s *MongoStorage) FindChannel(ctx context.Context, filter *comic.ComicFilter) (chan comic.Comic, error) {
	comics := make(chan comic.Comic, 100)
	go func() {
		defer close(comics)
		oriLimit := filter.Limit + filter.Skip
		filter.Limit = 100
		for filter.Limit+filter.Skip <= oriLimit {
			impls, err := s.Find(ctx, filter)
			if err != nil {
				return
			}
			for _, c := range impls {
				comics <- c
			}
			filter.Skip += filter.Limit
		}
	}()
	return comics, nil
}

func (s *MongoStorage) toMongoFilter(filter *comic.ComicFilter) bson.M {
	mongoFilter := bson.M{}
	if filter == nil {
		return mongoFilter
	}

	if filter.ID != nil {
		mongoFilter["cid"] = *filter.ID
	}
	if filter.TitlePattern != nil {
		mongoFilter["title"] = bson.M{"$regex": primitive.Regex{Pattern: *filter.TitlePattern, Options: "i"}}
	}
	if filter.PageMin != nil && filter.PageMax != nil {
		mongoFilter["$expr"] = bson.M{
			"$gte": bson.A{bson.M{"$size": "$tags"}, *filter.PageMin},
			"$lte": bson.A{bson.M{"$size": "$tags"}, *filter.PageMax},
		}
	} else if filter.PageMin != nil {
		mongoFilter["$expr"] = bson.M{"$gte": bson.A{bson.M{"$size": "$tags"}, *filter.PageMin}}
	} else if filter.PageMax != nil {
		mongoFilter["$expr"] = bson.M{"$lte": bson.A{bson.M{"$size": "$tags"}, *filter.PageMax}}
	}
	if filter.Valid != nil {
		mongoFilter["verify.valid"] = *filter.Valid
	}
	if filter.HasValid != nil {
		if *filter.HasValid {
			mongoFilter["verify.valid"] = bson.M{"$exists": 1}
		} else {
			mongoFilter["verify.valid"] = bson.M{"$exists": 0}
		}
	}

	return mongoFilter
}

// SaveVerifyResult 实现 ComicStorage 接口
func (s *MongoStorage) SaveVerifyResult(ctx context.Context, result *comic.VerifyResult) error {
	verifyInfo := comic.VerifyInfo{}
	verifyInfo.SetVerifyResult(result)
	_, err := s.db.Collection("comics").UpdateOne(
		ctx,
		bson.M{"_id": result.ComicID},
		bson.M{"$set": bson.M{
			"verify": verifyInfo,
		}},
	)
	if err != nil {
		return err
	}

	_, err = s.db.Collection("verify_results").InsertOne(ctx, result)
	return err
}

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
package api

import (
	"encoding/json"
)

type DownloadComicByIDRequest struct {
	Cid      int  `json:"cid"`
	MaxConn  int  `json:"max_conn"`
	MaxRetry int  `json:"max_retry"`
	Timeout  int  `json:"timeout"`
	IsSync   bool `json:"is_sync"`
}

type ComicInfo struct {
	Oid      string      `json:"_id,omitempty" bson:"_id"`
	Cid      int         `json:"cid,omitempty" bson:"cid"`
	ComicId  interface{} `json:"comic_id,omitempty" bson:"comic_id" `
	ComicUrl string      `json:"comic_url,omitempty" bson:"comic_url"`
	Id       int         `json:"id,omitempty" bson:"id"`
	Images   struct {
		Cover struct {
			H int    `json:"h,omitempty" bson:"h"`
			T string `json:"t,omitempty" bson:"t"`
			W int    `json:"w,omitempty" bson:"w"`
		} `json:"cover,omitempty" bson:"cover"`
		Pages []struct {
			H      int    `json:"h,omitempty" bson:"h"`
			T      string `json:"t,omitempty" bson:"t"`
			W      int    `json:"w,omitempty" bson:"w"`
			Status bool   `json:"status,omitempty" bson:"status"`
		} `json:"pages,omitempty" bson:"pages"`
		Thumbnail struct {
			H int    `json:"h,omitempty" bson:"h"`
			T string `json:"t,omitempty" bson:"t"`
			W int    `json:"w,omitempty" bson:"w"`
		} `json:"thumbnail,omitempty" bson:"thumbnail"`
	} `json:"images,omitempty" bson:"images"`
	MediaId      string `json:"media_id,omitempty" bson:"media_id"`
	NumFavorites int    `json:"num_favorites,omitempty" bson:"num_favorites"`
	NumPages     int    `json:"num_pages,omitempty" bson:"num_pages"`
	Scanlator    string `json:"scanlator,omitempty" bson:"scanlator"`
	Status       bool   `json:"status,omitempty" bson:"status"`
	Tags         []struct {
		Count int    `json:"count,omitempty" bson:"count"`
		Id    int    `json:"id,omitempty" bson:"id"`
		Name  string `json:"name,omitempty" bson:"name"`
		Type  string `json:"type,omitempty" bson:"type"`
		Url   string `json:"url,omitempty" bson:"url"`
	} `json:"tags,omitempty" bson:"tags"`
	Title struct {
		English  string `json:"english,omitempty" bson:"english"`
		Japanese string `json:"japanese,omitempty" bson:"japanese"`
		Pretty   string `json:"pretty,omitempty" bson:"pretty"`
	} `json:"title,omitempty" bson:"title"`
	UploadDate int `json:"upload_date,omitempty" bson:"upload_date"`
}

func (i *ComicInfo) ToMapInfo() (map[string]interface{}, error) {
	data, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	info := map[string]interface{}{}
	err = json.Unmarshal(data, &info)
	if err != nil {
		return nil, err
	}
	return info, nil
}

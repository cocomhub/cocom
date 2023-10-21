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
package view

import (
	"reflect"
	"testing"
)

func TestGalleryIndexPage_PageNumList(t *testing.T) {
	type fields struct {
		PopularNow []*GalleryDetail
		NewUpdates []*GalleryDetail
		CurPage    int
		LastPage   int
	}
	tests := []struct {
		name     string
		fields   fields
		wantList []int
	}{
		{
			name: "cur:0 last:10",
			fields: fields{
				CurPage:  0,
				LastPage: 10,
			},
			wantList: nil,
		},
		{
			name: "cur:1 last:0",
			fields: fields{
				CurPage:  1,
				LastPage: 0,
			},
			wantList: nil,
		},
		{
			name: "cur:20 last:10",
			fields: fields{
				CurPage:  20,
				LastPage: 10,
			},
			wantList: nil,
		},
		{
			name: "cur:1 last:10",
			fields: fields{
				CurPage:  1,
				LastPage: 10,
			},
			wantList: []int{1, 2, 3, 4, 5, 6},
		},
		{
			name: "cur:4 last:20",
			fields: fields{
				CurPage:  4,
				LastPage: 20,
			},
			wantList: []int{1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
		{
			name: "cur:10 last:20",
			fields: fields{
				CurPage:  10,
				LastPage: 20,
			},
			wantList: []int{5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		},
		{
			name: "cur:14 last:20",
			fields: fields{
				CurPage:  14,
				LastPage: 20,
			},
			wantList: []int{9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
		},
		{
			name: "cur:18 last:20",
			fields: fields{
				CurPage:  18,
				LastPage: 20,
			},
			wantList: []int{13, 14, 15, 16, 17, 18, 19, 20},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &GalleryIndexPage{
				PopularNow: tt.fields.PopularNow,
				NewUpdates: tt.fields.NewUpdates,
				CurPage:    tt.fields.CurPage,
				LastPage:   tt.fields.LastPage,
			}
			if gotList := p.PageNumList(); !reflect.DeepEqual(gotList, tt.wantList) {
				t.Errorf("PageNumList() = %v, want %v", gotList, tt.wantList)
			}
		})
	}
}

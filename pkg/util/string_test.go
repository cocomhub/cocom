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
package util

import (
	"reflect"
	"testing"
)

func TestSplitStrRightBySize(t *testing.T) {
	type args struct {
		raw  string
		size int
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "length 0 size -1",
			args: args{
				raw:  "",
				size: -1,
			},
			want: []string{""},
		},
		{
			name: "length 5 size 0",
			args: args{
				raw:  "12345",
				size: 0,
			},
			want: []string{"12345"},
		},
		{
			name: "length 5 size 1",
			args: args{
				raw:  "12345",
				size: 1,
			},
			want: []string{"1", "2", "3", "4", "5"},
		},
		{
			name: "length 5 size 2",
			args: args{
				raw:  "12345",
				size: 2,
			},
			want: []string{"1", "23", "45"},
		},
		{
			name: "length 5 size 3",
			args: args{
				raw:  "12345",
				size: 3,
			},
			want: []string{"12", "345"},
		},
		{
			name: "length 5 size 4",
			args: args{
				raw:  "12345",
				size: 4,
			},
			want: []string{"1", "2345"},
		},
		{
			name: "length 5 size 5",
			args: args{
				raw:  "12345",
				size: 5,
			},
			want: []string{"12345"},
		},
		{
			name: "length 5 size 10",
			args: args{
				raw:  "12345",
				size: 10,
			},
			want: []string{"12345"},
		},
		{
			name: "length 6 size 2",
			args: args{
				raw:  "123456",
				size: 2,
			},
			want: []string{"12", "34", "56"},
		},
		{
			name: "length 6 size 3",
			args: args{
				raw:  "123456",
				size: 3,
			},
			want: []string{"123", "456"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SplitStrRightBySize(tt.args.raw, tt.args.size); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitStrRightBySize() = %v, want %v", got, tt.want)
			}
		})
	}
}

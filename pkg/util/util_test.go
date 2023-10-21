package util

import (
	"reflect"
	"testing"
)

func TestFillIncrNum(t *testing.T) {
	type args struct {
		s     []int
		begin int
		sep   int
	}
	tests := []struct {
		name string
		args args
		want []int
	}{
		{
			name: "begin:1 sep:0",
			args: args{
				s:     make([]int, 5),
				begin: 1,
				sep:   0,
			},
			want: []int{1, 1, 1, 1, 1},
		},
		{
			name: "begin:1 sep:1",
			args: args{
				s:     make([]int, 5),
				begin: 1,
				sep:   1,
			},
			want: []int{1, 2, 3, 4, 5},
		},
		{
			name: "begin:1 sep:2",
			args: args{
				s:     make([]int, 5),
				begin: 1,
				sep:   2,
			},
			want: []int{1, 3, 5, 7, 9},
		},
		{
			name: "begin:1 sep:3",
			args: args{
				s:     make([]int, 5),
				begin: 1,
				sep:   3,
			},
			want: []int{1, 4, 7, 10, 13},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FillIncrNum(tt.args.s, tt.args.begin, tt.args.sep); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FillIncrNum() = %v, want %v", got, tt.want)
			}
		})
	}
}

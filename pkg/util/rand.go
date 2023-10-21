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
	"math/rand"
	"time"
)

var (
	s = rand.New(rand.NewSource(time.Now().UnixMicro()))
)

func NormFloat64() float64 {
	return s.NormFloat64()
}

func Seed(seed int64) {
	s.Seed(seed)
}

func Int63() int64 {
	return s.Int63()
}

func Uint32() uint32 {
	return s.Uint32()
}

func Uint64() uint64 {
	return s.Uint64()
}

func Int31() int32 {
	return s.Int31()
}

func Int() int {
	return s.Int()
}

func Int63n(n int64) int64 {
	return s.Int63n(n)
}

func Int31n(n int32) int32 {
	return s.Int31n(n)
}

func Intn(n int) int {
	return s.Intn(n)
}

func Float64() float64 {
	return s.Float64()
}

func Float32() float32 {
	return s.Float32()
}

func Perm(n int) []int {
	return s.Perm(n)
}

func Shuffle(n int, swap func(i int, j int)) {
	s.Shuffle(n, swap)
}

func Read(p []byte) (n int, err error) {
	return s.Read(p)
}

func ExpFloat64() float64 {
	return s.ExpFloat64()
}

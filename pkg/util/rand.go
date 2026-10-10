// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package util

import "math/rand"

// 本包所有随机函数委托 math/rand 包级函数。
// math/rand 包级函数由全局锁保护，goroutine 安全；
// 不要改回共享 *rand.Rand 实例（实例方法非并发安全，会撕裂 RNG 状态）。

func NormFloat64() float64 {
	return rand.NormFloat64()
}

func Int63() int64 {
	return rand.Int63()
}

func Uint32() uint32 {
	return rand.Uint32()
}

func Uint64() uint64 {
	return rand.Uint64()
}

func Int31() int32 {
	return rand.Int31()
}

func Int() int {
	return rand.Int()
}

func Int63n(n int64) int64 {
	return rand.Int63n(n)
}

func Int31n(n int32) int32 {
	return rand.Int31n(n)
}

func Intn(n int) int {
	return rand.Intn(n)
}

func Float64() float64 {
	return rand.Float64()
}

func Float32() float32 {
	return rand.Float32()
}

func Perm(n int) []int {
	return rand.Perm(n)
}

func Shuffle(n int, swap func(i int, j int)) {
	rand.Shuffle(n, swap)
}

func ExpFloat64() float64 {
	return rand.ExpFloat64()
}

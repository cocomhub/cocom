// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package httpwrap

// ErrCode 标准错误码
type ErrCode int

const (
	ErrCodeUnknown   ErrCode = -1 // 未知错误
	ErrCodeInvalid   ErrCode = -2 // 参数无效
	ErrCodeNotFound  ErrCode = -3 // 资源不存在
	ErrCodeForbidden ErrCode = -4 // 无权限
	ErrCodeInternal  ErrCode = -5 // 内部错误
)
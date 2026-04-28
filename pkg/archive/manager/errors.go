// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"errors"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
)

var (
	// 通用错误
	ErrNotFound        = errors.New("not found")
	ErrAlreadyExists   = errors.New("already exists")
	ErrInvalidArgument = errors.New("invalid argument")
	ErrInternal        = errors.New("internal error")
	ErrNotImplemented  = errors.New("not implemented")
	// 策略相关错误
	ErrUnsupportedPolicy = errors.New("unsupported policy")
)

func IsNotFound(err error) bool {
	return os.IsNotExist(err) || errors.Is(err, mongo.ErrNoDocuments) || errors.Is(err, ErrNotFound)
}

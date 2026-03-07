// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package errs

import (
	"github.com/cocomhub/cocom/pkg/errwrap"
)

var (
	ErrComicAlreadyDownloaded = errwrap.New(1000, "comic already downloaded")
	ErrComicDownloadRetryOver = errwrap.New(1001, "comic download retry over")
	ErrComicDownloadConnOver  = errwrap.New(1002, "comic download conn over")
)

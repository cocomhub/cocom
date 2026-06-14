// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import "testing"

func TestCustomLikeToTag_Skipped(t *testing.T) {
	t.Skip("CustomLikeToTag requires real MongoDB (mongo.ComicInfoCustom), skip in unit tests")
}

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

var indexFactories = map[string]func(IndexConfig) IndexStore{}

func RegisterIndexStoreFactory(typ string, f func(IndexConfig) IndexStore) {
	indexFactories[typ] = f
}

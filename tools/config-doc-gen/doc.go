// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package main (config-doc-gen) scans viper.SetDefault calls and // config-doc:
// annotations to generate a configuration reference document.
//
// Usage:
//
//	go generate ./tools/config-doc-gen
//	make config-doc
//
// Output is written to docs/config-reference.md (not docs/config.md, which is
// hand-maintained for user-friendly reading).
//
//go:generate go run . -o ../../docs/config-reference.md
package main

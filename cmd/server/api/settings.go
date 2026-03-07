// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api

type SetSettingsRequest struct {
	Type     string         `json:"type"`
	Settings map[string]any `json:"settings"`
}

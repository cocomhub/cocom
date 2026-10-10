// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package baidupcs

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cocomhub/cocom/pkg/storage"
)

func init() {
	storage.Register(Type, newFn)
}

func newFn(storageName string, config map[string]any) (storage.Storage, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}
	root, err := stringValue(config, "root")
	if err != nil {
		return nil, err
	}
	tempDir, err := optionalStringValue(config, "temp_dir")
	if err != nil {
		return nil, err
	}
	bduss, err := optionalStringValue(config, "bduss")
	if err != nil {
		return nil, err
	}
	cookies, err := optionalStringValue(config, "cookies")
	if err != nil {
		return nil, err
	}
	stoken, err := optionalStringValue(config, "stoken")
	if err != nil {
		return nil, err
	}
	sboxtkn, err := optionalStringValue(config, "sboxtkn")
	if err != nil {
		return nil, err
	}
	appID, err := optionalIntValue[int](config, "app_id")
	if err != nil {
		return nil, err
	}
	pcsAddr, err := optionalStringValue(config, "pcs_addr")
	if err != nil {
		return nil, err
	}
	pcsUserAgent, err := optionalStringValue(config, "pcs_user_agent")
	if err != nil {
		return nil, err
	}
	panUserAgent, err := optionalStringValue(config, "pan_user_agent")
	if err != nil {
		return nil, err
	}
	uid, err := optionalIntValue[uint64](config, "uid")
	if err != nil {
		return nil, err
	}
	return New(storageName, Config{
		Root:         root,
		TempDir:      tempDir,
		BDUSS:        bduss,
		Cookies:      cookies,
		SToken:       stoken,
		SBoxTKN:      sboxtkn,
		AppID:        appID,
		PCSAddr:      pcsAddr,
		PCSUserAgent: pcsUserAgent,
		PanUserAgent: panUserAgent,
		UID:          uid,
	})
}

func stringValue(config map[string]any, keys ...string) (string, error) {
	for _, key := range keys {
		raw, ok := config[key]
		if !ok {
			continue
		}
		value, ok := raw.(string)
		if !ok {
			return "", fmt.Errorf("%s is not a string", key)
		}
		if value == "" {
			return "", fmt.Errorf("%s is empty", key)
		}
		return value, nil
	}
	if len(keys) == 0 {
		return "", fmt.Errorf("missing string key")
	}
	return "", fmt.Errorf("%s is required", keys[0])
}

func optionalStringValue(config map[string]any, key string) (string, error) {
	raw, ok := config[key]
	if !ok || raw == nil {
		return "", nil
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("%s is not a string", key)
	}
	return value, nil
}

func optionalIntValue[T ~int | ~uint | ~int64 | ~uint64](config map[string]any, key string) (T, error) {
	raw, ok := config[key]
	if !ok || raw == nil {
		return 0, nil
	}
	switch value := raw.(type) {
	case int:
		return T(value), nil
	case int32:
		return T(value), nil
	case int64:
		return T(value), nil
	case float64:
		return T(value), nil
	case string:
		value = strings.TrimSpace(value)
		if value == "" {
			return 0, nil
		}
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%s is not an int", key)
		}
		return T(n), nil
	default:
		return 0, fmt.Errorf("%s is not an int", key)
	}
}

// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package baidupcs

import (
	"fmt"
	"time"

	"github.com/cocomhub/cocom/pkg/storage"
)

func init() {
	storage.Register(Type, newFn)
}

func newFn(storageName string, config map[string]any) (storage.Storage, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}
	command, err := stringValue(config, "command", "commandPath")
	if err != nil {
		return nil, err
	}
	root, err := stringValue(config, "root", "remoteRoot")
	if err != nil {
		return nil, err
	}
	tempDir, err := optionalStringValue(config, "tempDir")
	if err != nil {
		return nil, err
	}
	workDir, err := optionalStringValue(config, "workDir")
	if err != nil {
		return nil, err
	}
	timeout, err := durationValue(config, 30*time.Second, "timeout")
	if err != nil {
		return nil, err
	}
	args, err := stringSliceValue(config, "args", "globalArgs")
	if err != nil {
		return nil, err
	}
	return New(storageName, Config{
		Command: command,
		Root:    root,
		TempDir: tempDir,
		WorkDir: workDir,
		Timeout: timeout,
		Args:    args,
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

func durationValue(config map[string]any, defaultValue time.Duration, keys ...string) (time.Duration, error) {
	for _, key := range keys {
		raw, ok := config[key]
		if !ok || raw == nil {
			continue
		}
		switch value := raw.(type) {
		case time.Duration:
			if value <= 0 {
				return 0, fmt.Errorf("%s must be positive", key)
			}
			return value, nil
		case string:
			d, err := time.ParseDuration(value)
			if err != nil {
				return 0, fmt.Errorf("%s parse duration: %w", key, err)
			}
			if d <= 0 {
				return 0, fmt.Errorf("%s must be positive", key)
			}
			return d, nil
		case int:
			if value <= 0 {
				return 0, fmt.Errorf("%s must be positive", key)
			}
			return time.Duration(value) * time.Millisecond, nil
		case int64:
			if value <= 0 {
				return 0, fmt.Errorf("%s must be positive", key)
			}
			return time.Duration(value) * time.Millisecond, nil
		case float64:
			if value <= 0 {
				return 0, fmt.Errorf("%s must be positive", key)
			}
			return time.Duration(value * float64(time.Millisecond)), nil
		default:
			return 0, fmt.Errorf("%s is not a duration", key)
		}
	}
	return defaultValue, nil
}

func stringSliceValue(config map[string]any, keys ...string) ([]string, error) {
	for _, key := range keys {
		raw, ok := config[key]
		if !ok || raw == nil {
			continue
		}
		switch value := raw.(type) {
		case []string:
			return append([]string(nil), value...), nil
		case []any:
			out := make([]string, 0, len(value))
			for _, item := range value {
				s, ok := item.(string)
				if !ok {
					return nil, fmt.Errorf("%s contains non-string item", key)
				}
				out = append(out, s)
			}
			return out, nil
		default:
			return nil, fmt.Errorf("%s is not a string slice", key)
		}
	}
	return nil, nil
}

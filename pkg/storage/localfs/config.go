// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package localfs

import (
	"fmt"

	"github.com/cocomhub/cocom/pkg/storage"
)

func init() {
	storage.Register("localfs", newFn)
}

func newFn(storageName string, config map[string]any) (storage.Storage, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}
	root, ok := config["root"].(string)
	if !ok {
		return nil, fmt.Errorf("root is not a string")
	}
	if root == "" {
		return nil, fmt.Errorf("root is empty")
	}
	return New(storageName, root), nil
}

// SetFromMap 从 name→root 映射注册 localfs 后端。
// 调用方负责从配置系统读取值后传入。
func SetFromMap(values map[string]string) error {
	for name, root := range values {
		if name == "" {
			continue
		}
		// 空 root 视为配置错误，fail-fast（与 newFn 的 "root is empty" 一致）。
		// v0.0.57 基线行为：storage.Clear() → SetFromViper(keys...) 任一键为空即报错中止。
		if root == "" {
			return fmt.Errorf("root is empty")
		}
		if err := storage.SetFromConfig(storage.Config{
			Name:     name,
			Type:     "localfs",
			MetaData: map[string]any{"root": root},
		}); err != nil {
			return err
		}
	}
	return nil
}

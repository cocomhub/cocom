// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"fmt"
	"io"
)

type MigrateResult struct {
	Success []ObjectMeta
	Failed  map[string]error
}

func Migrate(ctx context.Context, src Storage, dst Storage, keys []string, opts ...Option) (MigrateResult, error) {
	res := MigrateResult{Failed: make(map[string]error)}
	for _, k := range keys {
		rc, meta, err := src.Get(ctx, k)
		if err != nil {
			res.Failed[k] = fmt.Errorf("get: %w", err)
			continue
		}
		func() {
			defer rc.Close()
			// 使用 TeeReader 以支持 ETag 计算选项
			pr, pw := io.Pipe()
			defer pr.Close()
			go func() {
				_, _ = io.Copy(pw, rc)
				_ = pw.Close()
			}()
			dstMeta, err := dst.Put(ctx, k, pr, opts...)
			if err != nil {
				res.Failed[k] = fmt.Errorf("put: %w", err)
				return
			}
			if meta.Size != 0 && dstMeta.Size != meta.Size {
				res.Failed[k] = fmt.Errorf("size mismatch: src=%d dst=%d", meta.Size, dstMeta.Size)
				return
			}
			res.Success = append(res.Success, *dstMeta)
		}()
	}
	if len(res.Failed) > 0 && len(res.Success) == 0 {
		return res, fmt.Errorf("all migrations failed")
	}
	return res, nil
}

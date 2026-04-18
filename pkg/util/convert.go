// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"encoding/json"
	"fmt"
	"reflect"
)

func ToMap(v any) (map[string]any, error) {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Pointer {
		return ToMap(reflect.ValueOf(v).Elem().Interface())
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input must be a struct: %T", v)
	}

	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	info := map[string]any{}
	err = json.Unmarshal(data, &info)
	if err != nil {
		return nil, err
	}
	return info, nil
}

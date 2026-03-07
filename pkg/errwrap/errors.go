// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package errwrap

import "strings"

func NewErrors(errs ...error) Errors {
	return errs
}

type Errors []error

func (e *Errors) Add(err error) {
	*e = append(*e, err)
}

func (e Errors) Count() int {
	return len(e)
}

func (e Errors) Err() error {
	if e.Count() == 0 {
		return nil
	}
	return e
}

func (e Errors) Error() string {
	if len(e) == 0 {
		return ""
	}

	buf := strings.Builder{}
	buf.WriteString("Errors[")
	for i, err := range e {
		if err == nil {
			continue
		}
		if i > 0 {
			buf.WriteString(" | ")
		}
		buf.WriteString(err.Error())
	}
	buf.WriteString("]")
	return buf.String()
}

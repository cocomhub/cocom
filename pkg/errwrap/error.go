// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package errwrap

import (
	"fmt"
)

var (
	ErrInvalidArgs      = New(1000, "invalid args")
	ErrImageOpen        = New(2000, "image open failed")
	ErrImageSave        = New(2001, "image save failed")
	ErrImageFormat      = New(2002, "unsupported image format")
	ErrImageSubsampling = New(2003, "luma/chroma subsampling ratio error")
	ErrImageBatch       = New(2004, "batch process failed")
	ErrImageEmpty       = New(2005, "empty image source")
	ErrImageDir         = New(2006, "directory operation failed")
	ErrImageConv        = New(2007, "format conversion failed")
)

func New(code int, msg string) *Error {
	return &Error{
		code: code,
		iErr: nil,
		msg:  msg,
	}
}

type Error struct {
	code int
	iErr error
	msg  string
}

func (e Error) String() string {
	return fmt.Sprintf("code[%d] msg[%s] ierr[%v]", e.Code(), e.Msg(), e.IErr())
}

func (e Error) GoString() string {
	return e.String()
}

func (e Error) Error() string {
	if e.code == 0 {
		return ""
	}
	return e.String()
}

func (e Error) Code() int {
	return e.code
}

func (e Error) Msg() string {
	if err, ok := e.iErr.(*Error); ok {
		return fmt.Sprintf("%s: %s", e.msg, err.Msg())
	}
	return e.msg
}

func (e Error) IErr() error {
	if err, ok := e.iErr.(*Error); ok {
		return err.IErr()
	}
	return e.iErr
}

func (e Error) SetIErr(err error) *Error {
	n := e
	n.iErr = err
	return &n
}

func (e Error) SetIErrF(format string, a ...interface{}) *Error {
	n := e
	n.iErr = fmt.Errorf(format, a...)
	return &n
}

func (e Error) Is(target error) bool {
	if err, ok := target.(*Error); ok {
		return e.code == err.code
	}
	return false
}

func (e Error) Unwrap() error {
	return e.iErr
}

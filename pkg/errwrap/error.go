/*
Copyright © 2023 suixibing <suixibing@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package errwrap

import (
	"fmt"
)

var (
	ErrInvalidArgs = New(1000, "invalid args")
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
	return e.msg
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

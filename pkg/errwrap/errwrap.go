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

import "strings"

func NewWrap() Wrap {
	return nil
}

type Wrap []error

func (e *Wrap) Add(err error) {
	*e = append(*e, err)
}

func (e Wrap) Count() int {
	return len(e)
}

func (e Wrap) Err() error {
	if e.Count() == 0 {
		return nil
	}
	return e
}

func (e Wrap) Error() string {
	if len(e) == 0 {
		return ""
	}

	buf := strings.Builder{}
	buf.WriteString("errsWrap[")
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

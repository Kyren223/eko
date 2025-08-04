// Eko: A terminal-native social media platform
// Copyright (C) 2025 Kyren223
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package assert

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime/debug"
	"sync"
)

var (
	writer io.Writer = os.Stderr

	flushes = []io.Closer{}
	flushMu sync.Mutex

	assertData = map[string]any{}
	mapMu      sync.Mutex
)

func AddData(key string, value any) {
	mapMu.Lock()
	assertData[key] = value
	mapMu.Unlock()
}

func RemoveData(key string) {
	mapMu.Lock()
	delete(assertData, key)
	mapMu.Unlock()
}

func AddFlush(flusher io.Closer) {
	flushMu.Lock()
	flushes = append(flushes, flusher)
	flushMu.Unlock()
}

func SetWriter(w io.Writer) {
	writer = w
}

func runAssert(message string, args ...any) {
	flushMu.Lock()
	for len(flushes) != 0 {
		flusher := flushes[len(flushes)-1]
		_ = flusher.Close()
		flushes = flushes[:len(flushes)-1]
	}
	flushMu.Unlock()

	values := []any{
		"msg", message,
	}
	values = append(values, args...)
	mapMu.Lock()
	for k, v := range assertData {
		values = append(values, k, v)
	}
	mapMu.Unlock()

	fmt.Fprintf(writer, "ARGS: %+v\n", args)
	fmt.Fprintf(writer, "ASSERT\n")
	for i := 0; i < len(values); i += 2 {
		fmt.Fprintf(writer, "   %s=%v\n", values[i], values[i+1])
	}
	fmt.Fprintln(writer, string(debug.Stack()))

	os.Exit(1)
}

func Assert(assertion bool, message string, args ...any) {
	if !assertion {
		runAssert(message, args...)
	}
}

func NoError(err error, message string, args ...any) {
	if err != nil {
		args = append(args, "error", err)
		runAssert(message, args...)
	}
}

func Never(message string, args ...any) {
	runAssert(message, args...)
}

func Abort(message string, args ...any) {
	runAssert(message, args...)
}

func NotNil(value any, message string, args ...any) {
	if value == nil || reflect.ValueOf(value).Kind() == reflect.Ptr && reflect.ValueOf(value).IsNil() {
		runAssert(message, args...)
	}
}

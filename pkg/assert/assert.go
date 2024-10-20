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
		flusher.Close()
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

func NotNil(value any, message string, args ...any) {
	if value == nil || reflect.ValueOf(value).Kind() == reflect.Ptr && reflect.ValueOf(value).IsNil() {
		runAssert(message, args...)
	}
}

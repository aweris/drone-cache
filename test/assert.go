package test

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TODO: Reconsider if it's not used or useful!

// Assert fails the test if the condition is false.
func Assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		tb.Fatalf("%s:%d: "+msg+"\n", append([]interface{}{filepath.Base(file), line}, v...)...)
	}
}

// Ok fails the test if an err is not nil.
func Ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		tb.Fatalf("%s:%d: unexpected error: %s\n", filepath.Base(file), line, err.Error())
	}
}

// NotOk fails the test if an err is nil.
func NotOk(tb testing.TB, err error) {
	if err == nil {
		_, file, line, _ := runtime.Caller(1)
		tb.Fatalf("%s:%d: expected error, got nothing\n", filepath.Base(file), line)
	}
}

// Equals fails the test if want is not equal to got.
func Equals(tb testing.TB, want, got interface{}, v ...interface{}) {
	if diff := cmp.Diff(want, got); diff != "" {
		_, file, line, _ := runtime.Caller(1)

		var msg string
		if len(v) > 0 {
			msg = fmt.Sprintf(v[0].(string), v[1:]...)
		}

		tb.Fatalf("%s:%d:"+msg+"\n\n\t (-want +got):\n%s", filepath.Base(file), line, diff)
	}
}

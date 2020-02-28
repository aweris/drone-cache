package test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Assert fails the test if the condition is false.
func Assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	tb.Helper()
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		tb.Fatalf("%s:%d: "+msg+"\n", append([]interface{}{filepath.Base(file), line}, v...)...)
	}
}

// Ok fails the test if an err is not nil.
func Ok(tb testing.TB, err error) {
	tb.Helper()
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		tb.Fatalf("%s:%d: unexpected error: %s\n", filepath.Base(file), line, err.Error())
	}
}

// NotOk fails the test if an err is nil.
func NotOk(tb testing.TB, err error) {
	tb.Helper()
	if err == nil {
		_, file, line, _ := runtime.Caller(1)
		tb.Fatalf("%s:%d: expected error, got nothing\n", filepath.Base(file), line)
	}
}

// Expected TODO
func Expected(tb testing.TB, got, want error) {
	tb.Helper()
	NotOk(tb, got)

	if errors.Is(got, want) {
		return
	}

	_, file, line, _ := runtime.Caller(1)
	tb.Fatalf("%s:%d: got unexpected error: %v\n", filepath.Base(file), line, got.Error())
}

// Exists TODO
func Exists(tb testing.TB, path string) {
	tb.Helper()
	_, err := os.Stat(path)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		if os.IsNotExist(err) {
			tb.Fatalf("%s:%d: should exists: %s\n", filepath.Base(file), line, err.Error())
		}
	}
}

// Equals fails the test if want is not equal to got.
func Equals(tb testing.TB, want, got interface{}, v ...interface{}) {
	tb.Helper()
	if diff := cmp.Diff(want, got); diff != "" {
		_, file, line, _ := runtime.Caller(1)

		var msg string
		if len(v) > 0 {
			msg = fmt.Sprintf(v[0].(string), v[1:]...)
		}

		tb.Fatalf("%s:%d:"+msg+"\n\n\t (-want +got):\n%s", filepath.Base(file), line, diff)
	}
}

// EqualDirs TODO
func EqualDirs(tb testing.TB, dst string, srcs []string) {
	tb.Helper()
	srcList := []string{}
	for _, src := range srcs {
		if IsDir(src) {
			paths, err := Expand(src)
			if err != nil {
				tb.Fatalf("expand %s: %v\n", src, err)
			}
			srcList = append(srcList, paths...)
			continue
		}

		srcList = append(srcList, src)
	}

	dstList, err := Expand(dst)
	if err != nil {
		tb.Fatalf("expand %s: %v\n", dst, err)
	}

	relDstList := []string{}
	for _, p := range dstList {
		rel, err := filepath.Rel(dst, p)
		if err != nil {
			tb.Fatalf("relative path %q: %q %v\n", p, rel, err)
		}
		relDstList = append(relDstList, filepath.Join("/", filepath.Clean(rel)))
		// relDstList = append(relDstList, p)
	}

	sort.Strings(srcList)
	sort.Strings(relDstList)

	Equals(tb, srcList, relDstList)
	_, file, line, _ := runtime.Caller(1)

	for i := 0; i < len(srcList); i++ {
		wContent, err := ioutil.ReadFile(srcList[i])
		if err != nil {
			tb.Fatalf("%s:%d: unexpected error, path <%s>: %s\n", filepath.Base(file), line, srcList[i], err.Error())
		}

		gContent, err := ioutil.ReadFile(dstList[i])
		if err != nil {
			tb.Fatalf("%s:%d: unexpected error, path <%s>: %s\n", filepath.Base(file), line, dstList[i], err.Error())
		}

		Equals(tb, wContent, gContent)
	}
}

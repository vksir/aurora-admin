package errutil

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
)

var (
	ErrNotFound = errors.New("not found")
	ErrBusy     = errors.New("busy")
	ErrErrType  = errors.New("error type")
)

func wrap(err error) error {
	var pcs [1]uintptr
	// skip [runtime.Callers, this function]
	runtime.Callers(3, pcs[:])
	f, _ := runtime.CallersFrames(pcs[:]).Next()
	dir, file := filepath.Split(f.File)
	return fmt.Errorf("%s/%s:%d %w", filepath.Base(dir), file, f.Line, err)
}

func Wrap(err error) error {
	return wrap(err)
}

func WrapF(format string, a ...any) error {
	return wrap(fmt.Errorf(format, a...))
}

func WrapNotFound(dst string) error {
	return wrap(fmt.Errorf("%w: %s", ErrNotFound, dst))
}

func WrapErrType(v any) error {
	return wrap(fmt.Errorf("%w: %T", ErrErrType, v))
}

func Fatal(err error) {
	if err != nil {
		panic(wrap(err))
	}
}

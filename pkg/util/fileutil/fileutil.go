package fileutil

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"aurora-admin/pkg/util/errutil"
)

var Home = func() string {
	p, err := os.UserHomeDir()
	errutil.Fatal(err)
	return p
}()

var Executable = func() string {
	p, err := os.Executable()
	errutil.Fatal(err)
	return p
}()

func Exist(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}

func Read(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func Write(path string, content []byte) error {
	return os.WriteFile(path, content, 0640)
}

func Remove(path string) error {
	return os.RemoveAll(path)
}

func MkDir(paths ...string) error {
	for _, p := range paths {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return fmt.Errorf("MkdirAll %s: %w", p, err)
		}
	}
	return nil
}

func MkdirAndReturn(path string) (string, error) {
	if !Exist(path) {
		err := MkDir(path)
		if err != nil {
			return "", errutil.Wrap(err)
		}
	}
	return path, nil
}

func ClearDir(path string) error {
	dirs, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, d := range dirs {
		err = os.RemoveAll(filepath.Join(path, d.Name()))
		if err != nil {
			return err
		}
	}
	return nil
}

func Cp(src, dst string) error {
	srcStat, err := os.Stat(src)
	if err != nil {
		return errutil.Wrap(err)
	}

	if srcStat.IsDir() {
		srcFs := os.DirFS(src)
		err = os.CopyFS(dst, srcFs)
		if err != nil {
			return errutil.Wrap(err)
		}
		return nil
	}

	// 如果目标是目录，则将文件复制到目录中
	newDst := dst
	dstStat, err := os.Stat(dst)
	if err == nil && dstStat.IsDir() {
		newDst = filepath.Join(dst, srcStat.Name())
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return errutil.Wrap(err)
	}
	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.OpenFile(newDst, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, srcStat.Mode())
	if err != nil {
		return errutil.Wrap(err)
	}
	defer func() { _ = dstFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

func Mv(src, dst string) error {
	return os.Rename(src, dst)
}

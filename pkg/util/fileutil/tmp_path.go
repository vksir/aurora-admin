package fileutil

import (
	"aurora-admin/pkg/log"
	"aurora-admin/pkg/util/errutil"
	"os"
	"path/filepath"
)

type TempDir string

func NewTempDir(dir, pattern string) (TempDir, error) {
	path, err := os.MkdirTemp(dir, pattern)
	if err != nil {
		return "", errutil.Wrap(err)
	}
	return TempDir(path), nil
}

func (p TempDir) String() string {
	return string(p)
}

func (p TempDir) Join(elem ...string) string {
	return filepath.Join(append([]string{string(p)}, elem...)...)
}

func (p TempDir) Clear() {
	err := os.RemoveAll(string(p))
	if err != nil {
		log.Error("clear tmp path failed", "path", p, "err", err)
	}
}

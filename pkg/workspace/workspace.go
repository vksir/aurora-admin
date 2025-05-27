package workspace

import (
	"github.com/vksir/vkiss-lib/pkg/util/errutil"
	"github.com/vksir/vkiss-lib/pkg/util/fileutil"
	"os"
	"path/filepath"
)

var ws string

func Ws() string {
	return ws
}

func ServiceDir() string {
	return filepath.Join(ws, "service")
}

func LogDir() string {
	return filepath.Join(ws, "log")
}

func CachePath() string {
	return filepath.Join(ws, "cache.json")
}

func DBPath() string {
	return filepath.Join(ws, "aurora-admin.db")
}

func LogPath() string {
	return filepath.Join(LogDir(), "aurora-admin.log")
}

func TempDir() string {
	return filepath.Join(ws, "tmp")
}

func Init(workspace string) {
	ws = workspace
	dirs := []string{ws, ServiceDir(), LogDir(), TempDir()}
	for _, d := range dirs {
		err := os.MkdirAll(d, 0o755)
		errutil.Check(err)
	}
	err := fileutil.ClearDir(TempDir())
	errutil.Check(err)
}

package workspace

import (
	"github.com/vksir/vkiss-lib/pkg/util/errutil"
	"os"
	"path/filepath"
)

var ws string
var home string

func Ws() string {
	return ws
}

func Home() string {
	return home
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

func Init(workspace string) {
	ws = workspace
	dirs := []string{ws, ServiceDir(), LogDir()}
	for _, d := range dirs {
		err := os.MkdirAll(d, 0o755)
		errutil.Check(err)
	}

	var err error
	home, err = os.UserHomeDir()
	errutil.Check(err)
}

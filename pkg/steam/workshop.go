package steam

import (
	"path/filepath"
	"sync"

	"aurora-admin/pkg/log"
	"aurora-admin/pkg/workspace"
	"aurora-admin/pkg/util/fileutil"
)

// workshopDir 只创建一次目录并缓存结果，
// 避免每次调用 WorkshopItemPath 都重复执行 mkdir 系统调用。
var workshopDir = sync.OnceValue(func() string {
	dir, err := fileutil.MkdirAndReturn(filepath.Join(workspace.ServiceDir(), "workshop"))
	if err != nil {
		log.Error("MkdirAndReturn", "err", err)
		return ""
	}
	return dir
})

func WorkshopDir() string {
	return workshopDir()
}

func WorkshopItemPath(appId, workshopId string) string {
	return filepath.Join(WorkshopDir(), "steamapps/workshop/content", appId, workshopId)
}

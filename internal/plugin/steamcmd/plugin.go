package steamcmd

import (
	"aurora-admin/pkg/workspace"
	"github.com/vksir/vkiss-lib/pkg/log"
	"github.com/vksir/vkiss-lib/pkg/util/fileutil"
	"path/filepath"
)

func WorkshopDir() string {
	dir, err := fileutil.MkdirAndReturn(filepath.Join(workspace.ServiceDir(), "workshop"))
	if err != nil {
		log.Error("MkdirAndReturn", "err", err)
	}
	return dir
}

func WorkshopItemPath(appId, workshopId string) string {
	return filepath.Join(WorkshopDir(), "steamapps/workshop/content", appId, workshopId)
}

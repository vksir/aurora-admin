package dontstarve

import (
	"aurora-admin/pkg/steam"
	"aurora-admin/pkg/workspace"
	"fmt"
	"path/filepath"

	"aurora-admin/pkg/log"
	"aurora-admin/pkg/util/fileutil"
)

const (
	PluginName            = "dontstarve"
	AutoUpdateCronJobName = "dontstarve_autoupdate"
	ModUpdateCronJobName  = "dontstarve_modupdate"

	AppId       = "322330"
	ServerAppId = "343050"

	ShardMaster = "Master"
	ShardCaves  = "Caves"

	AutoupdateCronJobInterval = 1800
	ModUpdateCronJobInterval  = 1800
	AutoupdateNotifyInterval  = 300
)

// 自动更新期间的游戏内公告文案
const (
	msgServerUpdateLogout = iconEye + " The server requires an update. Please log out immediately. " + iconEye
	msgModUpdateLogout    = iconEye + " The server requires a mod update. Please log out immediately. " + iconEye
)

// --- path functions (replaces pathHelper struct hierarchy) ---

func programDir() string {
	return filepath.Join(workspace.ServiceDir(), PluginName)
}

func clusterDir() string {
	dir, err := fileutil.MkdirAndReturn(filepath.Join(fileutil.Home, ".klei/DoNotStarveTogether"))
	if err != nil {
		log.Error("MkdirAndReturn", "err", err)
	}
	return dir
}

func dstServPath() string {
	return filepath.Join(programDir(), "bin64/dontstarve_dedicated_server_nullrenderer_x64")
}

func appManifestPath() string {
	return filepath.Join(programDir(), "steamapps/appmanifest_343050.acf")
}

func modSetupPath() string {
	return filepath.Join(programDir(), "mods/dedicated_server_mods_setup.lua")
}

func ugcModDir() string {
	return steam.WorkshopItemPath(AppId, "")
}

func clusterRoot(archID string) string {
	return filepath.Join(clusterDir(), archID)
}

func clusterIni(archID string) string {
	return filepath.Join(clusterRoot(archID), "cluster.ini")
}

func adminListTxt(archID string) string {
	return filepath.Join(clusterRoot(archID), "adminlist.txt")
}

func clusterTokenTxt(archID string) string {
	return filepath.Join(clusterRoot(archID), "cluster_token.txt")
}

func shardRoot(archID, shard string) (string, error) {
	if shard != "Master" && shard != "Caves" {
		return "", fmt.Errorf("invalid shard %s", shard)
	}
	return filepath.Join(clusterRoot(archID), shard), nil
}

func shardWorldOverrideLua(archID, shard string) (string, error) {
	root, err := shardRoot(archID, shard)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "leveldataoverride.lua"), nil
}

func shardModOverrideLua(archID, shard string) (string, error) {
	root, err := shardRoot(archID, shard)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "modoverrides.lua"), nil
}

func shardServerIni(archID, shard string) (string, error) {
	root, err := shardRoot(archID, shard)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "server.ini"), nil
}

func worldTypeShard(worldType string) (string, error) {
	switch worldType {
	case "Master":
		return ShardMaster, nil
	case "Caves":
		return ShardCaves, nil
	default:
		return "", fmt.Errorf("unknown world type: %s", worldType)
	}
}

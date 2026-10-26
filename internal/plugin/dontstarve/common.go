package dontstarve

import (
	"aurora-admin/pkg/steam"
	"aurora-admin/pkg/workspace"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// 服务启动完成的判定：DST 世界加载完成后会在分片输出里打印模拟运行/暂停标记。
const (
	shardActiveMarker = "Sim unpaused"
	shardPausedMarker = "Sim paused"

	// waitActiveTimeout 是等待启动完成的上限，首次生成世界、下载模组可能较慢。
	waitActiveTimeout = 10 * time.Minute
	// waitActivePollInterval 是等待期间检查分片进程是否仍存活的间隔。
	waitActivePollInterval = time.Second
)

// shardFailureMarkers 是 DST 启动流程中明确表示无法启动的输出，
// 命中后不必再等到超时（此时进程可能仍然存活）。
var shardFailureMarkers = []string{
	"Your Server Will Not Start",
	"[Shard] Invalid shard id",
	"[Shard] Missing shard name",
}

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

// shardFIFOPath 返回分片控制台命名的 stdin 管道路径（relay 模式）。
func shardFIFOPath(shard string) string {
	return filepath.Join(programDir(), "run", "dst-"+strings.ToLower(shard)+".fifo")
}

// shardLogPath 返回分片 stdout/stderr 重定向的日志文件路径（relay 模式）。
func shardLogPath(shard string) string {
	return filepath.Join(programDir(), "log", "dst-"+strings.ToLower(shard)+".log")
}

// resetShardLog 清空分片服务日志，使日志只反映即将启动的这一次运行。
func resetShardLog(shard string) error {
	path := shardLogPath(shard)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}
	return f.Close()
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

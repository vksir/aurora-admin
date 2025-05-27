package dontstarve

import (
	"aurora-admin/internal/plugin/steamcmd"
	"aurora-admin/pkg/workspace"
	"fmt"
	"path/filepath"

	"github.com/vksir/vkiss-lib/pkg/log"
	"github.com/vksir/vkiss-lib/pkg/util/fileutil"
)

const (
	PluginName            = "dontstarve"
	AutoUpdateCronJobName = "dontstarve_autoupdate"

	AppId       = "322330"
	ServerAppId = "343050"

	ShardMaster = "Master"
	ShardCaves  = "Caves"

	AutoupdateCronJobInterval = 1800
	AutoupdateNotifyInterval  = 300
)

type pathHelper struct {
	home   string
	svcDir string
}

func newPathHelper() *pathHelper {
	return &pathHelper{
		home:   fileutil.Home,
		svcDir: workspace.ServiceDir(),
	}
}

func (p *pathHelper) ProgramDir() string {
	return filepath.Join(p.svcDir, PluginName)
}

func (p *pathHelper) ClusterDir() string {
	dir, err := fileutil.MkdirAndReturn(filepath.Join(p.home, ".klei/DoNotStarveTogether"))
	if err != nil {
		log.Error("MkdirAndReturn", "err", err)
	}
	return dir
}

func (p *pathHelper) DstServPath() string {
	return filepath.Join(p.ProgramDir(), "bin64/dontstarve_dedicated_server_nullrenderer_x64")
}

func (p *pathHelper) AppManifestPath() string {
	return filepath.Join(p.ProgramDir(), "steamapps/appmanifest_343050.acf")
}

func (p *pathHelper) ModSetupPath() string {
	return filepath.Join(p.ProgramDir(), "mods/dedicated_server_mods_setup.lua")
}

func (p *pathHelper) UgcModDir() string {
	return steamcmd.WorkshopItemPath(AppId, "")
}

type clusterPathHelper struct {
	archID string
	root   string
}

func newClusterPathHelper(ph *pathHelper, archId string) *clusterPathHelper {
	cp := &clusterPathHelper{
		archID: archId,
		root:   filepath.Join(ph.ClusterDir(), archId),
	}
	return cp
}

func (p *clusterPathHelper) SetRoot(path string) *clusterPathHelper {
	p.root = path
	return p
}

func (p *clusterPathHelper) Root() string {
	return p.root
}

func (p *clusterPathHelper) ClusterIni() string {
	return filepath.Join(p.root, "cluster.ini")
}

func (p *clusterPathHelper) AdminListTxt() string {
	return filepath.Join(p.root, "adminlist.txt")
}

func (p *clusterPathHelper) ClusterTokenTxt() string {
	return filepath.Join(p.root, "cluster_token.txt")
}

func (p *clusterPathHelper) Shard(shard string) (*shardPathHelper, error) {
	if shard != "Master" && shard != "Caves" {
		return nil, fmt.Errorf("invalid shard %s", shard)
	}
	return &shardPathHelper{root: filepath.Join(p.root, shard)}, nil
}

type shardPathHelper struct {
	root string
}

func (p *shardPathHelper) Root() string {
	return p.root
}

func (p *shardPathHelper) WorldOverrideLua() string {
	return filepath.Join(p.root, "leveldataoverride.lua")
}

func (p *shardPathHelper) ModOverrideLua() string {
	return filepath.Join(p.root, "modoverrides.lua")
}

func (p *shardPathHelper) ServerIni() string {
	return filepath.Join(p.root, "server.ini")
}

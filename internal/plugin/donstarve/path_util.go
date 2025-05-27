package donstarve

import (
	"aurora-admin/pkg/workspace"
	"fmt"
	"github.com/vksir/vkiss-lib/pkg/log"
	"github.com/vksir/vkiss-lib/pkg/util/fileutil"
	"path/filepath"
)

const (
	name = "dontstarve"
)

type PathHelper struct {
	home   string
	svcDir string
}

func NewPathHelper() *PathHelper {
	return &PathHelper{
		home:   workspace.Home(),
		svcDir: workspace.ServiceDir(),
	}
}

func (p *PathHelper) ProgramDir() string {
	return filepath.Join(p.svcDir, name)
}

func (p *PathHelper) ClusterDir() string {
	res := filepath.Join(p.home, ".klei/DoNotStarveTogether")
	if !fileutil.Exist(res) {
		err := fileutil.MkDir(res)
		if err != nil {
			log.ErrorF("mkdir %s failed: %v", res, err)
		}
	}
	return res
}

func (p *PathHelper) DstServPath() string {
	return filepath.Join(p.ProgramDir(), "bin64/dontstarve_dedicated_server_nullrenderer_x64")
}

func (p *PathHelper) ModSetupPath() string {
	return filepath.Join(p.ProgramDir(), "mods/dedicated_server_mods_setup.lua")
}

type ClusterPath struct {
	ArchId        string
	Root          string
	ClusterIni    string
	AdminListFile string
	TokenFile     string

	Master ShardPath
	Caves  ShardPath
}

type ShardPath struct {
	Root              string
	WorldOverrideFile string
	ModOverrideFile   string
	ServerFile        string
}

func NewClusterPath(ph *PathHelper, archId string) *ClusterPath {
	var cp ClusterPath

	cp.ArchId = archId
	cp.Root = filepath.Join(ph.ClusterDir(), cp.ArchId)
	cp.ClusterIni = filepath.Join(cp.Root, "cluster.ini")
	cp.AdminListFile = filepath.Join(cp.Root, "adminlist.txt")
	cp.TokenFile = filepath.Join(cp.Root, "cluster_token.txt")

	cp.Master.Root = filepath.Join(cp.Root, "Master")
	cp.Master.WorldOverrideFile = filepath.Join(cp.Master.Root, "leveldataoverride.lua")
	cp.Master.ModOverrideFile = filepath.Join(cp.Master.Root, "modoverrides.lua")
	cp.Master.ServerFile = filepath.Join(cp.Master.Root, "server.ini")

	cp.Caves.Root = filepath.Join(cp.Root, "Caves")
	cp.Caves.WorldOverrideFile = filepath.Join(cp.Caves.Root, "leveldataoverride.lua")
	cp.Caves.ModOverrideFile = filepath.Join(cp.Caves.Root, "modoverrides.lua")
	cp.Caves.ServerFile = filepath.Join(cp.Master.Root, "server.ini")

	return &cp
}

func (p *ClusterPath) Dirs() []string {
	return []string{
		p.Root, p.Master.Root, p.Caves.Root,
	}
}

func (p *ClusterPath) GetShardPath(shard string) (*ShardPath, error) {
	if shard == "Master" {
		return &p.Master, nil
	} else if shard == "Caves" {
		return &p.Caves, nil
	} else {
		return nil, fmt.Errorf("invalid shard: %s", shard)
	}
}

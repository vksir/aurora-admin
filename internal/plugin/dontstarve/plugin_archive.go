package dontstarve

import (
	"aurora-admin/assets"
	"aurora-admin/internal/entity/dstety"
	"aurora-admin/internal/store"
	"context"
	"database/sql"
	"io"
	"os"

	"aurora-admin/pkg/log"
	"aurora-admin/pkg/util/errutil"
	"aurora-admin/pkg/util/fileutil"
	"gopkg.in/ini.v1"
)

const (
	iconEye = "󰀅"
)

func (p *Plugin) UploadCreateArchive(ctx context.Context, r io.Reader, archName string) (*store.Archive, error) {
	var arch *store.Archive
	err := p.uploadArchive(ctx, r, archName, func(ctx context.Context, path string, archName string) error {
		var err error
		arch, err = p.createArchiveFromDisk(ctx, path, archName)
		if err != nil {
			return errutil.Wrap(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return arch, nil
}

func (p *Plugin) UploadUpdateArchive(ctx context.Context, r io.Reader, id string, fileName string) (*store.Archive, error) {
	var arch *store.Archive
	err := p.uploadArchive(ctx, r, fileName, func(ctx context.Context, path string, fileName string) error {
		var err error
		arch, err = p.updateArchiveFromDisk(ctx, path, id)
		if err != nil {
			return errutil.Wrap(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return arch, nil
}

func (p *Plugin) uploadArchive(ctx context.Context, r io.Reader, archName string,
	callback func(ctx context.Context, path string, fileName string) error) error {

	ph := newPathHelper()
	tempDir, err := fileutil.NewTempDir(ph.ClusterDir(), "archive-*")
	if err != nil {
		return errutil.Wrap(err)
	}
	defer tempDir.Clear()
	uploadFile := tempDir.Join("upload")
	w, err := os.Create(uploadFile)
	if err != nil {
		return errutil.Wrap(err)
	}
	defer log.Close(w)
	_, err = io.Copy(w, r)
	if err != nil {
		return errutil.Wrap(err)
	}
	log.Close(w)
	unzipDir := tempDir.Join("unzip")
	err = fileutil.Unzip(uploadFile, unzipDir)
	if err != nil {
		return errutil.Wrap(err)
	}

	err = callback(ctx, unzipDir, archName)
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

func (p *Plugin) CreateArchive(ctx context.Context) (*store.Archive, error) {
	arch := &store.Archive{
		ClusterName:        "Aurora",
		ClusterPassword:    "6666",
		ClusterDescription: "Just Have Fun",
		MaxPlayers:         6,
		Pvp:                false,
		World: []dstety.DontStarveWorld{
			{
				Type:          dstety.WorldTypeMaster,
				ServerConfig:  assets.DontStarveMasterServerIni,
				WorldOverride: assets.DontStarveMasterOverride,
			},
			{
				Type:          dstety.WorldTypeCaves,
				ServerConfig:  assets.DontStarveCavesServerIni,
				WorldOverride: assets.DontStarveCavesOverride,
			},
		},
	}
	arch, err := p.db.CreateArchive(ctx, arch)
	if err != nil {
		return nil, errutil.Wrap(err)
	}

	d := newDeployer(arch.ID, p.db, p.state, p.logger)
	err = d.DeployCluster(ctx)
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	return arch, nil
}

func (p *Plugin) DownloadArchive(ctx context.Context, id string, w io.Writer) error {
	_, err := p.db.GetArchive(ctx, id)
	if err != nil {
		return errutil.Wrap(err)
	}

	cph := newClusterPathHelper(p.ph, id)
	if !fileutil.Exist(cph.Root()) {
		return errutil.WrapNotFound(cph.Root())
	}

	err = fileutil.ZipToWriter(cph.Root(), w)
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

func (p *Plugin) DeleteArchive(ctx context.Context, id string) error {
	cph := newClusterPathHelper(p.ph, id)
	err := fileutil.Remove(cph.Root())
	if err != nil {
		return errutil.Wrap(err)
	}

	err = p.db.DeleteArchive(ctx, id)
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

func (p *Plugin) UpdateArchive(ctx context.Context, arch *store.Archive) (*store.Archive, error) {
	_, err := p.db.UpdateArchive(ctx, arch)
	if err != nil {
		return nil, errutil.Wrap(err)
	}

	d := newDeployer(arch.ID, p.db, p.state, p.logger)
	err = d.DeployCluster(ctx)
	if err != nil {
		return nil, errutil.Wrap(err)
	}

	return p.GetArchive(ctx, arch.ID)
}

func (p *Plugin) GetArchive(ctx context.Context, id string) (*store.Archive, error) {
	return p.db.GetArchive(ctx, id)
}

func (p *Plugin) createArchiveFromDisk(ctx context.Context, path string, archName string) (*store.Archive, error) {
	if !fileutil.Exist(path) {
		return nil, errutil.WrapNotFound(path)
	}

	var newArch *store.Archive
	err := p.db.WithTx(ctx, func(ctx context.Context, _ *sql.Tx) error {
		arch := &store.Archive{Remark: archName}
		arch, err := p.db.CreateArchive(ctx, arch)
		if err != nil {
			return errutil.Wrap(err)
		}

		cph := newClusterPathHelper(p.ph, arch.ID).SetRoot(path)
		err = p.loadArchive(ctx, cph, arch)
		if err != nil {
			return errutil.Wrap(err)
		}

		_, err = p.db.UpdateArchive(ctx, arch)
		if err != nil {
			return errutil.Wrap(err)
		}

		newArch = arch

		newCph := newClusterPathHelper(p.ph, arch.ID)
		err = fileutil.Mv(cph.Root(), newCph.Root())
		if err != nil {
			return errutil.Wrap(err)
		}
		return nil
	})
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	return newArch, nil
}

func (p *Plugin) updateArchiveFromDisk(ctx context.Context, path string, archID string) (*store.Archive, error) {
	if !fileutil.Exist(path) {
		return nil, errutil.WrapNotFound(path)
	}

	arch, err := p.db.GetArchive(ctx, archID)
	if err != nil {
		return nil, errutil.Wrap(err)
	}

	cph := newClusterPathHelper(p.ph, arch.ID).SetRoot(path)
	err = p.loadArchive(ctx, cph, arch)
	if err != nil {
		return nil, errutil.Wrap(err)
	}

	newArch, err := p.UpdateArchive(ctx, arch)
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	return newArch, nil
}

func (p *Plugin) loadArchive(ctx context.Context, cph *clusterPathHelper, arch *store.Archive) error {
	master, err := cph.Shard(dstety.WorldTypeMaster)
	if err != nil {
		return errutil.Wrap(err)
	}
	caves, err := cph.Shard(dstety.WorldTypeCaves)
	if err != nil {
		return errutil.Wrap(err)
	}

	rawModBytes, err := fileutil.Read(master.ModOverrideLua())
	if err != nil {
		return errutil.Wrap(err)
	}
	mods, err := p.createModFromModOverrides(ctx, rawModBytes)
	if err != nil {
		return errutil.Wrap(err)
	}
	arch.Mods = mods

	masterWorldOverridesBytes, err := fileutil.Read(master.WorldOverrideLua())
	if err != nil {
		return errutil.Wrap(err)
	}
	masterServerConfigBytes, err := fileutil.Read(master.ServerIni())
	if err != nil {
		return errutil.Wrap(err)
	}
	cavesWorldOverridesBytes, err := fileutil.Read(caves.WorldOverrideLua())
	if err != nil {
		return errutil.Wrap(err)
	}
	cavesServerConfigBytes, err := fileutil.Read(caves.ServerIni())
	if err != nil {
		return errutil.Wrap(err)
	}
	arch.World = []dstety.DontStarveWorld{{
		Type:          dstety.WorldTypeMaster,
		WorldOverride: string(masterWorldOverridesBytes),
		ServerConfig:  string(masterServerConfigBytes),
	}, {
		Type:          dstety.WorldTypeCaves,
		WorldOverride: string(cavesWorldOverridesBytes),
		ServerConfig:  string(cavesServerConfigBytes),
	}}

	clusterConfigBytes, err := fileutil.Read(cph.ClusterIni())
	if err != nil {
		return errutil.Wrap(err)
	}
	conf, err := ini.Load(clusterConfigBytes)
	if err != nil {
		return errutil.Wrap(err)
	}
	arch.ClusterName = conf.Section("NETWORK").Key("cluster_name").String()
	arch.ClusterPassword = conf.Section("NETWORK").Key("cluster_password").String()
	arch.ClusterDescription = conf.Section("NETWORK").Key("cluster_description").String()
	arch.MaxPlayers = conf.Section("GAMEPLAY").Key("max_players").MustInt(6)
	arch.Pvp = conf.Section("GAMEPLAY").Key("pvp").MustBool(false)
	return nil
}

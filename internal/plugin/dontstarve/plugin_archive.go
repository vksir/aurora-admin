package dontstarve

import (
	"aurora-admin/assets"
	"aurora-admin/ent"
	"aurora-admin/ent/dontstarvearchive"
	"aurora-admin/internal/entity/dstety"
	"aurora-admin/pkg/database"
	"context"
	"io"
	"os"

	"github.com/vksir/vkiss-lib/pkg/log"
	"github.com/vksir/vkiss-lib/pkg/util/errutil"
	"github.com/vksir/vkiss-lib/pkg/util/fileutil"
	"gopkg.in/ini.v1"
)

const (
	iconEye = "󰀅"
)

func (p *Plugin) UploadCreateArchive(ctx context.Context, r io.Reader, archName string) (*ent.DontStarveArchive, error) {
	var arch *ent.DontStarveArchive
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

func (p *Plugin) UploadUpdateArchive(ctx context.Context, r io.Reader, id string, fileName string) (*ent.DontStarveArchive, error) {
	var arch *ent.DontStarveArchive
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

	// TODO: save at tmp path
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

func (p *Plugin) CreateArchive(ctx context.Context) (*ent.DontStarveArchive, error) {
	arch, err := p.db.DontStarveArchive.Create().
		SetClusterName("Aurora").
		SetClusterPassword("6666").
		SetClusterDescription("Just Have Fun").
		SetMaxPlayers(6).
		SetPvp(false).
		SetWorld([]dstety.DontStarveWorld{
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
		}).
		Save(ctx)
	if err != nil {
		return nil, errutil.Wrap(err)
	}

	d := newDeployer(arch.ID, p.db, p.logger)
	err = d.DeployCluster(ctx)
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	return arch, nil
}

func (p *Plugin) DownloadArchive(ctx context.Context, id string, w io.Writer) error {
	_, err := p.db.DontStarveArchive.Query().Where(dontstarvearchive.ID(id)).WithMods().Only(ctx)
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

	err = p.db.DontStarveArchive.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

func (p *Plugin) UpdateArchive(ctx context.Context, arch *ent.DontStarveArchive) (*ent.DontStarveArchive, error) {
	_, err := p.db.DontStarveArchive.UpdateOne(arch).
		SetRemark(arch.Remark).
		SetClusterName(arch.ClusterName).
		SetClusterPassword(arch.ClusterPassword).
		SetClusterDescription(arch.ClusterDescription).
		SetMaxPlayers(arch.MaxPlayers).
		SetPvp(arch.Pvp).
		SetWorld(arch.World).
		ClearMods().
		AddMods(arch.Edges.Mods...).
		Save(ctx)
	if err != nil {
		return nil, errutil.Wrap(err)
	}

	d := newDeployer(arch.ID, p.db, p.logger)
	err = d.DeployCluster(ctx)
	if err != nil {
		return nil, errutil.Wrap(err)
	}

	return p.GetArchive(ctx, arch.ID)
}

func (p *Plugin) GetArchive(ctx context.Context, id string) (*ent.DontStarveArchive, error) {
	return p.db.DontStarveArchive.Query().Where(dontstarvearchive.ID(id)).WithMods().Only(ctx)
}

func (p *Plugin) createArchiveFromDisk(ctx context.Context, path string, archName string) (*ent.DontStarveArchive, error) {
	if !fileutil.Exist(path) {
		return nil, errutil.WrapNotFound(path)
	}

	var newArch *ent.DontStarveArchive
	err := database.WithTx(ctx, p.db, func(ctx context.Context, tx *ent.Tx) error {
		arch, err := tx.DontStarveArchive.Create().Save(ctx)
		if err != nil {
			return errutil.Wrap(err)
		}

		arch.Remark = archName
		cph := newClusterPathHelper(p.ph, arch.ID).SetRoot(path)
		err = p.loadArchive(ctx, cph, arch)
		if err != nil {
			return errutil.Wrap(err)
		}

		_, err = tx.DontStarveArchive.UpdateOne(arch).
			SetRemark(arch.Remark).
			SetClusterName(arch.ClusterName).
			SetClusterPassword(arch.ClusterPassword).
			SetClusterDescription(arch.ClusterDescription).
			SetMaxPlayers(arch.MaxPlayers).
			SetPvp(arch.Pvp).
			SetWorld(arch.World).
			ClearMods().
			AddMods(arch.Edges.Mods...).
			Save(ctx)
		if err != nil {
			return errutil.Wrap(err)
		}
		newArch, err = tx.DontStarveArchive.Query().Where(dontstarvearchive.ID(arch.ID)).WithMods().Only(ctx)
		if err != nil {
			return errutil.Wrap(err)
		}

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

func (p *Plugin) updateArchiveFromDisk(ctx context.Context, path string, archID string) (*ent.DontStarveArchive, error) {
	if !fileutil.Exist(path) {
		return nil, errutil.WrapNotFound(path)
	}

	arch, err := p.db.DontStarveArchive.Query().Where(dontstarvearchive.ID(archID)).WithMods().Only(ctx)
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

func (p *Plugin) loadArchive(ctx context.Context, cph *clusterPathHelper, arch *ent.DontStarveArchive) error {
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
	arch.Edges.Mods = mods

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

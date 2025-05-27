package donstarve

import (
	"aurora-admin/assets"
	"aurora-admin/internal/entity/dstety"
	"context"
	"errors"
	"github.com/vksir/vkiss-lib/pkg/util/errutil"
	"github.com/vksir/vkiss-lib/pkg/util/fileutil"
	"gopkg.in/ini.v1"
	"gorm.io/gorm"
)

const (
	iconEye = "󰀅"
)

func (d *Plugin) GetArchive(ctx context.Context, id string) (dstety.DontStarveArchive, error) {
	return d.archRepo.GetOne(ctx, id)
}

func (d *Plugin) ListArchive(ctx context.Context) ([]dstety.DontStarveArchive, error) {
	return d.archRepo.GetAll(ctx)
}

func (d *Plugin) CreateArchive(ctx context.Context) (dstety.DontStarveArchive, error) {
	archive := dstety.DontStarveArchive{
		ClusterName:        "Aurora",
		ClusterPassword:    "6666",
		ClusterDescription: "Just Have Fun",
		MaxPlayers:         "6",
		Pvp:                "false",
		Worlds: []dstety.DontStarveWorld{
			{
				Type:          dstety.DontStarveWorldTypeMaster,
				ServerConfig:  assets.DontStarveMasterServerIni,
				WorldOverride: assets.DontStarveMasterOverride,
			},
			{
				Type:          dstety.DontStarveWorldTypeCaves,
				ServerConfig:  assets.DontStarveCavesServerIni,
				WorldOverride: assets.DontStarveCavesOverride,
			},
		},
		Mods: make([]dstety.DontStarveModInArchive, 0),
	}

	err := d.archRepo.Create(ctx, &archive)
	if err != nil {
		d.logger.ErrorC(ctx, "arch repo create failed", "archive", archive, "err", err)
		return archive, errutil.Wrap(err)
	}

	err = d.deployCluster(ctx, &archive)
	if err != nil {
		d.logger.ErrorC(ctx, "deploy cluster failed, begin clean repo", "archive", archive, "err", err)
		if err := d.archRepo.Delete(ctx, archive.ID); err != nil {
			d.logger.ErrorC(ctx, "arch repo delete failed", "archive", archive, "err", err)
		}
		return archive, errutil.Wrap(err)
	}
	d.logger.InfoC(ctx, "arch repo create success", "archive", archive)
	return archive, nil
}

func (d *Plugin) createArchiveFromDisk(ctx context.Context, archId *string) error {
	cp := NewClusterPath(d.ph, *archId)
	if !fileutil.Exist(cp.Root) {
		return errutil.WrapOsNotExistErr(cp.Root)
	}

	mods, err := d.loadModInfo(ctx, cp)
	if err != nil {
		return errutil.Wrap(err)
	}

	masterWld, err := d.loadShardInfo(dstety.DontStarveWorldTypeMaster, &cp.Master)
	if err != nil {
		return errutil.Wrap(err)
	}
	cavesWld, err := d.loadShardInfo(dstety.DontStarveWorldTypeCaves, &cp.Caves)
	if err != nil {
		return errutil.Wrap(err)
	}

	clusterIniContent, err := fileutil.Read(cp.ClusterIni)
	if err != nil {
		return errutil.Wrap(err)
	}
	conf, err := ini.Load(clusterIniContent)
	if err != nil {
		return errutil.Wrap(err)
	}
	archive := dstety.DontStarveArchive{
		ClusterName:        conf.Section("NETWORK").Key("cluster_name").Value(),
		ClusterPassword:    conf.Section("NETWORK").Key("cluster_password").Value(),
		ClusterDescription: conf.Section("NETWORK").Key("cluster_description").Value(),
		MaxPlayers:         conf.Section("GAMEPLAY").Key("max_players").Value(),
		Pvp:                conf.Section("GAMEPLAY").Key("Pvp").Value(),
		Worlds:             []dstety.DontStarveWorld{*masterWld, *cavesWld},
		Mods:               mods,
	}

	err = d.archRepo.Create(ctx, &archive)
	if err != nil {
		d.logger.ErrorC(ctx, "arch repo create failed", "archive", archive, "err", err)
		return errutil.Wrap(err)
	}

	newCp := NewClusterPath(d.ph, archive.ID)
	err = fileutil.Mv(cp.Root, newCp.Root)
	if err != nil {
		d.logger.ErrorC(ctx, "cluster mv failed, begin clean repo", "src", cp.Root, "dst", newCp.Root, "err", err)
		if err := d.archRepo.Delete(ctx, archive.ID); err != nil {
			d.logger.ErrorC(ctx, "arch repo delete failed", "archive", archive, "err", err)
		}
		return errutil.Wrap(err)
	}
	d.logger.InfoC(ctx, "rename archive %s to %s", *archId, archive.ID)
	*archId = archive.ID

	d.logger.InfoC(ctx, "create archive from disk success", "archive", archive)
	return nil
}

func (d *Plugin) loadShardInfo(shardType string, sp *ShardPath) (*dstety.DontStarveWorld, error) {
	servCfg, err := fileutil.Read(sp.ServerFile)
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	wldCfg, err := fileutil.Read(sp.WorldOverrideFile)
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	world := dstety.DontStarveWorld{
		Type:          shardType,
		ServerConfig:  string(servCfg),
		WorldOverride: string(wldCfg),
	}
	return &world, nil
}

func (d *Plugin) loadModInfo(ctx context.Context, cp *ClusterPath) ([]dstety.DontStarveModInArchive, error) {
	master, err := fileutil.Read(cp.Master.ModOverrideFile)
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	mods, err := parseModLua(string(master))
	if err != nil {
		return nil, errutil.Wrap(err)
	}

	// 若数据库中不存在 Mod，则创建。为 Belongs To 关系做准备
	for i, mod := range mods {
		modInDb, err := d.modRepo.GetOneByKey(ctx, "workshop_id", mod.WorkshopId)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = d.modRepo.Create(ctx, mod)
			if err != nil {
				return nil, errutil.Wrap(err)
			}
		} else {
			mods[i] = &modInDb
		}
	}

	var modInArchives []dstety.DontStarveModInArchive
	for _, mod := range mods {
		modInArchives = append(modInArchives, mod.ToInArchive(cp.ArchId))
	}
	return modInArchives, nil
}

func (d *Plugin) deployCluster(ctx context.Context, arch *dstety.DontStarveArchive) error {
	cp := NewClusterPath(d.ph, arch.ID)

	err := fileutil.MkDir(cp.Root)
	if err != nil {
		return errutil.Wrap(err)
	}

	err = d.deployClusterIni(ctx, arch)
	if err != nil {
		return errutil.Wrap(err)
	}

	for _, world := range arch.Worlds {
		err = d.deployWorld(ctx, arch, &world)
		if err != nil {
			return errutil.Wrap(err)
		}
	}

	d.logger.InfoC(ctx, "deploy cluster success")
	return nil
}

func (d *Plugin) deployClusterIni(ctx context.Context, arch *dstety.DontStarveArchive) error {
	cp := NewClusterPath(d.ph, arch.ID)

	conf, err := ini.Load([]byte(assets.DontStarveClusterIni))
	if err != nil {
		return errutil.Wrap(err)
	}

	conf.Section("NETWORK").Key("cluster_name").
		SetValue(iconEye + arch.ClusterName + iconEye)
	conf.Section("NETWORK").Key("cluster_password").
		SetValue(arch.ClusterPassword)
	conf.Section("NETWORK").Key("cluster_description").
		SetValue(arch.ClusterDescription)

	conf.Section("GAMEPLAY").Key("max_players").SetValue(arch.MaxPlayers)
	conf.Section("GAMEPLAY").Key("Pvp").SetValue(arch.Pvp)

	err = conf.SaveTo(cp.ClusterIni)
	if err != nil {
		return errutil.Wrap(err)
	}
	d.logger.InfoC(ctx, "deploy cluster ini success")
	return nil
}

func (d *Plugin) deployWorld(ctx context.Context, arch *dstety.DontStarveArchive, world *dstety.DontStarveWorld) error {
	cp := NewClusterPath(d.ph, arch.ID)

	sp, err := cp.GetShardPath(world.Type)
	if err != nil {
		return errutil.Wrap(err)
	}

	err = fileutil.MkDir(sp.Root)
	if err != nil {
		return errutil.Wrap(err)
	}

	err = fileutil.Write(sp.ServerFile, []byte(world.ServerConfig))
	if err != nil {
		return errutil.Wrap(err)
	}

	err = fileutil.Write(sp.WorldOverrideFile, []byte(world.WorldOverride))
	if err != nil {
		return errutil.Wrap(err)
	}

	content := generateModLua(arch.Mods)
	err = fileutil.Write(sp.ModOverrideFile, []byte(content))
	if err != nil {
		return errutil.Wrap(err)
	}

	d.logger.InfoC(ctx, "deploy world success")
	return nil
}

func (d *Plugin) deployModSetup(ctx context.Context, arch *dstety.DontStarveArchive) error {
	content := generateModSetup(arch.Mods)
	err := fileutil.Write(d.ph.ModSetupPath(), []byte(content))
	if err != nil {
		return errutil.Wrap(err)
	}
	d.logger.InfoC(ctx, "deploy mod setup success")
	return nil
}

func (d *Plugin) deployToken(ctx context.Context, arch *dstety.DontStarveArchive) error {
	cp := NewClusterPath(d.ph, arch.ID)
	err := fileutil.Write(cp.TokenFile, []byte(cacheToken.Get()))
	if err != nil {
		return errutil.Wrap(err)
	}
	d.logger.InfoC(ctx, "deploy token success")
	return nil
}

func (d *Plugin) DeleteArchive(ctx context.Context, id string) error {
	cp := NewClusterPath(d.ph, id)
	err := fileutil.Rm(cp.Root)
	if err != nil {
		return errutil.Wrap(err)
	}

	err = d.archRepo.Delete(ctx, id)
	if err != nil {
		d.logger.ErrorC(ctx, "repo delete failed: id=%s, err=%v", id, err)
		return errutil.Wrap(err)
	}

	return nil
}

func (d *Plugin) UpdateArchive(ctx context.Context, archive dstety.DontStarveArchive) (dstety.DontStarveArchive, error) {
	err := d.deployCluster(ctx, &archive)
	if err != nil {
		return archive, errutil.Wrap(err)
	}

	err = d.archRepo.Update(ctx, &archive)
	if err != nil {
		d.logger.ErrorC(ctx, "repo update failed: archive=%+v, err=%v", archive, err)
		return archive, errutil.Wrap(err)
	}
	return archive, err
}

package dontstarve

import (
	"aurora-admin/assets"
	"aurora-admin/ent"
	"aurora-admin/ent/dontstarvearchive"
	"aurora-admin/internal/entity/dstety"
	"aurora-admin/pkg/cache"
	"context"
	"fmt"
	"github.com/vksir/vkiss-lib/pkg/log"
	"github.com/vksir/vkiss-lib/pkg/template"
	"github.com/vksir/vkiss-lib/pkg/util/convutil"
	"github.com/vksir/vkiss-lib/pkg/util/errutil"
	"github.com/vksir/vkiss-lib/pkg/util/fileutil"
	"gopkg.in/ini.v1"
	"strconv"
)

type deployer struct {
	db     *ent.Client
	logger *log.Logger
	archID string
	ph     *pathHelper
	cph    *clusterPathHelper
}

func newDeployer(archID string, db *ent.Client, logger *log.Logger) *deployer {
	d := &deployer{
		db:     db,
		archID: archID,
		logger: logger,
	}
	d.ph = newPathHelper()
	d.cph = newClusterPathHelper(d.ph, archID)
	return d
}

func (d *deployer) DeployBeforeStart(ctx context.Context) error {
	arch, err := d.db.DontStarveArchive.Query().Where(dontstarvearchive.ID(d.archID)).WithMods().Only(ctx)
	if err != nil {
		return errutil.Wrap(err)
	}
	admins, err := d.db.DontStarveAdmin.Query().All(ctx)
	if err != nil {
		return errutil.Wrap(err)
	}
	if cache.G.DontStarve.Token == "" {
		return fmt.Errorf("token not exist")
	}

	err = template.ExecuteFiles(map[string]template.Template{
		d.ph.ModSetupPath():     &deployModSetup{Mods: arch.Edges.Mods},
		d.cph.ClusterTokenTxt(): &deployToken{Token: cache.G.DontStarve.Token},
		d.cph.AdminListTxt():    &deployAdmin{Admin: admins},
	})
	if err != nil {
		return errutil.Wrap(err)
	}

	err = d.deployClusterIni(ctx, arch)
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

func (d *deployer) DeployCluster(ctx context.Context) error {
	arch, err := d.db.DontStarveArchive.Query().Where(dontstarvearchive.ID(d.archID)).WithMods().Only(ctx)
	if err != nil {
		return errutil.Wrap(err)
	}

	err = fileutil.MkDir(d.cph.Root())
	if err != nil {
		return errutil.Wrap(err)
	}

	err = d.deployClusterIni(ctx, arch)
	if err != nil {
		return errutil.Wrap(err)
	}

	for _, world := range arch.World {
		err = d.deployWorld(ctx, arch, &world)
		if err != nil {
			return errutil.Wrap(err)
		}
	}
	return nil
}

func (d *deployer) deployClusterIni(ctx context.Context, arch *ent.DontStarveArchive) error {
	clusterConfigBytes := []byte(assets.DontStarveClusterIni)
	if fileutil.Exist(d.cph.ClusterIni()) {
		var err error
		clusterConfigBytes, err = fileutil.Read(d.cph.ClusterIni())
		if err != nil {
			return errutil.Wrap(err)
		}
	}

	conf, err := ini.Load(clusterConfigBytes)
	if err != nil {
		return errutil.Wrap(err)
	}

	conf.Section("NETWORK").Key("cluster_name").
		SetValue(iconEye + arch.ClusterName + iconEye)
	conf.Section("NETWORK").Key("cluster_password").
		SetValue(cache.G.DontStarve.PassWord)
	conf.Section("NETWORK").Key("cluster_description").
		SetValue(arch.ClusterDescription)
	conf.Section("NETWORK").Key("tick_rate").
		SetValue(strconv.Itoa(cache.G.DontStarve.TickRate))
	conf.Section("NETWORK").Key("lan_only_cluster").
		SetValue("false")

	conf.Section("GAMEPLAY").Key("max_players").
		SetValue(convutil.String(arch.MaxPlayers))
	conf.Section("GAMEPLAY").Key("Pvp").
		SetValue(convutil.String(arch.Pvp))

	err = conf.SaveTo(d.cph.ClusterIni())
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

func (d *deployer) deployWorld(ctx context.Context, arch *ent.DontStarveArchive, world *dstety.DontStarveWorld) error {
	sph, err := d.cph.Shard(world.Type)
	if err != nil {
		return errutil.Wrap(err)
	}

	err = fileutil.MkDir(sph.Root())
	if err != nil {
		return errutil.Wrap(err)
	}

	err = fileutil.Write(sph.ServerIni(), []byte(world.ServerConfig))
	if err != nil {
		return errutil.Wrap(err)
	}

	err = fileutil.Write(sph.WorldOverrideLua(), []byte(world.WorldOverride))
	if err != nil {
		return errutil.Wrap(err)
	}

	err = d.deployMod(ctx, arch, sph)
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

func (d *deployer) deployMod(ctx context.Context, arch *ent.DontStarveArchive, sph *shardPathHelper) error {
	var rawConfigs []string
	for _, m := range arch.Edges.Mods {
		rawConfigs = append(rawConfigs, m.RawConfig)
	}
	modOverridesContent := genModOverrideContent(rawConfigs)
	err := fileutil.Write(sph.ModOverrideLua(), []byte(modOverridesContent))
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

type deployModSetup struct {
	Mods []*ent.DontStarveMod
}

func (d *deployModSetup) Template() string {
	return assets.DontStarveModsSetupTmpl
}

type deployToken struct {
	Token string
}

func (d *deployToken) Template() string {
	return assets.DontStarveClusterTokenTmpl
}

type deployAdmin struct {
	Admin []*ent.DontStarveAdmin
}

func (d *deployAdmin) Template() string {
	return assets.DontStarveAdminListTmpl
}

type deployModOverrides struct {
	Mods []*ent.DontStarveMod
}

func (d *deployModOverrides) Template() string {
	return assets.DontStarveModOverridesTmpl
}

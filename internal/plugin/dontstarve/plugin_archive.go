package dontstarve

import (
	"aurora-admin/assets"
	"aurora-admin/internal/entity/dstety"
	"aurora-admin/internal/store"
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"

	"aurora-admin/pkg/log"
	"aurora-admin/pkg/util/errutil"
	"aurora-admin/pkg/util/fileutil"
	"gopkg.in/ini.v1"
)

const iconEye = "󰀅"

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

	tempDir, err := fileutil.NewTempDir(clusterDir(), "archive-*")
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
				Type:          "Master",
				ServerConfig:  assets.DontStarveMasterServerIni,
				WorldOverride: assets.DontStarveMasterOverride,
			},
			{
				Type:          "Caves",
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

	root := clusterRoot(id)
	if !fileutil.Exist(root) {
		return errutil.WrapNotFound(root)
	}

	err = fileutil.ZipToWriter(root, w)
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

func (p *Plugin) DeleteArchive(ctx context.Context, id string) error {
	err := fileutil.Remove(clusterRoot(id))
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
	arch, err := p.db.GetArchive(ctx, id)
	if err != nil {
		return nil, err
	}
	p.fillArchiveModsFromDisk(ctx, arch)
	return arch, nil
}

// fillArchiveModsFromDisk 兼容旧存档：早期版本的存档没有写入 archive_mods 关联，
// 只按数据库关联判断会把已启用的模组显示为全部禁用，保存时还会把关联清空。
// 这里在读取存档时按需从存档目录的 Master/modoverrides.lua 回填关联（只补不删）。
func (p *Plugin) fillArchiveModsFromDisk(ctx context.Context, arch *store.Archive) {
	if len(arch.Mods) > 0 {
		return
	}
	mods, err := p.backfillArchiveMods(ctx, arch)
	if err != nil {
		log.WarnC(ctx, "backfill archive mods from disk failed", "archive", arch.ID, "err", err)
		return
	}
	if len(mods) > 0 {
		arch.Mods = mods
	}
}

// backfillArchiveMods 解析存档目录中的 modoverrides.lua，把其中启用的模组写入
// 模组库并与存档建立关联。模组已在库中时不覆盖其配置（配置按模组维度共享）。
func (p *Plugin) backfillArchiveMods(ctx context.Context, arch *store.Archive) ([]*store.Mod, error) {
	root := clusterRoot(arch.ID)
	if !fileutil.Exist(root) {
		return nil, nil
	}

	rawModOverrides, err := fileutil.Read(filepath.Join(root, ShardMaster, "modoverrides.lua"))
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	workshopID2Config, err := parseModLuaByRegex(string(rawModOverrides))
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	if len(workshopID2Config) == 0 {
		return nil, nil
	}

	var mods []*store.Mod
	err = p.db.WithTx(ctx, func(ctx context.Context, _ *sql.Tx) error {
		for workshopID, rawConfig := range workshopID2Config {
			mod, err := p.db.GetModByWorkshopID(ctx, workshopID)
			if err == sql.ErrNoRows {
				mod = &store.Mod{WorkshopID: workshopID, RawConfig: rawConfig}
				mod, err = p.db.CreateMod(ctx, mod)
			}
			if err != nil {
				return errutil.Wrap(err)
			}
			mods = append(mods, mod)
		}
		return nil
	})
	if err != nil {
		return nil, errutil.Wrap(err)
	}

	modIDs := make([]string, 0, len(mods))
	for _, mod := range mods {
		modIDs = append(modIDs, mod.ID)
	}
	if err := p.db.AddArchiveMods(ctx, arch.ID, modIDs); err != nil {
		return nil, errutil.Wrap(err)
	}
	return mods, nil
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

		err = p.loadArchive(ctx, path, arch)
		if err != nil {
			return errutil.Wrap(err)
		}

		_, err = p.db.UpdateArchive(ctx, arch)
		if err != nil {
			return errutil.Wrap(err)
		}

		newArch = arch

		err = fileutil.Mv(path, clusterRoot(arch.ID))
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

	err = p.loadArchive(ctx, path, arch)
	if err != nil {
		return nil, errutil.Wrap(err)
	}

	newArch, err := p.UpdateArchive(ctx, arch)
	if err != nil {
		return nil, errutil.Wrap(err)
	}
	return newArch, nil
}

func (p *Plugin) loadArchive(ctx context.Context, rootDir string, arch *store.Archive) error {
	archID := arch.ID

	masterModOverride := filepath.Join(rootDir, "Master", "modoverrides.lua")
	rawModBytes, err := fileutil.Read(masterModOverride)
	if err != nil {
		return errutil.Wrap(err)
	}
	mods, err := p.createModFromModOverrides(ctx, rawModBytes)
	if err != nil {
		return errutil.Wrap(err)
	}
	arch.Mods = mods

	masterOverride := filepath.Join(rootDir, "Master", "leveldataoverride.lua")
	masterWorldOverridesBytes, err := fileutil.Read(masterOverride)
	if err != nil {
		return errutil.Wrap(err)
	}
	masterServerIni := filepath.Join(rootDir, "Master", "server.ini")
	masterServerConfigBytes, err := fileutil.Read(masterServerIni)
	if err != nil {
		return errutil.Wrap(err)
	}
	cavesOverride := filepath.Join(rootDir, "Caves", "leveldataoverride.lua")
	cavesWorldOverridesBytes, err := fileutil.Read(cavesOverride)
	if err != nil {
		return errutil.Wrap(err)
	}
	cavesServerIni := filepath.Join(rootDir, "Caves", "server.ini")
	cavesServerConfigBytes, err := fileutil.Read(cavesServerIni)
	if err != nil {
		return errutil.Wrap(err)
	}

	arch.World = []dstety.DontStarveWorld{{
		Type:          "Master",
		WorldOverride: string(masterWorldOverridesBytes),
		ServerConfig:  string(masterServerConfigBytes),
	}, {
		Type:          "Caves",
		WorldOverride: string(cavesWorldOverridesBytes),
		ServerConfig:  string(cavesServerConfigBytes),
	}}

	clusterConfigBytes, err := fileutil.Read(clusterIni(archID))
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

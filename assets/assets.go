package assets

import (
	"embed"
	"fmt"
	"os"
	"path"
)

//go:embed config.toml
var DefaultConfig string

//go:embed tmodloader/serverconfig.txt
var TModLoaderServerConfig string

//go:embed dontstarve/Cluster_1/cluster.ini
var DontStarveClusterIni string

//go:embed dontstarve/Cluster_1/Master/leveldataoverride.lua
var DontStarveMasterOverride string

//go:embed dontstarve/Cluster_1/Master/server.ini
var DontStarveMasterServerIni string

//go:embed dontstarve/Cluster_1/Caves/leveldataoverride.lua
var DontStarveCavesOverride string

//go:embed dontstarve/Cluster_1/Caves/server.ini
var DontStarveCavesServerIni string

//go:embed dontstarve/template/modoverrides.lua.tmpl
var DontStarveModOverridesTmpl string

//go:embed dontstarve/template/mods_setup.lua.tmpl
var DontStarveModsSetupTmpl string

//go:embed dontstarve/template/Cluster/cluster_token.txt.tmpl
var DontStarveClusterTokenTmpl string

//go:embed dontstarve/template/Cluster/adminlist.txt.tmpl
var DontStarveAdminListTmpl string

//go:embed web/dist/*
var StaticFS embed.FS

func Cp(fs embed.FS, src string, dst string) error {
	stat, err := os.Stat(dst)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dst, 0o755)
		if err != nil {
			return fmt.Errorf("mkdir %s failed: %v", dst, err)
		}
	} else if !stat.IsDir() {
		return fmt.Errorf("%s is not dir", dst)
	}

	entries, err := fs.ReadDir(src)
	if err != nil {
		return fmt.Errorf("embed fs read dir failed: %v", err)
	}

	for _, entry := range entries {
		oneSrc := path.Join(src, entry.Name())
		oneDst := path.Join(dst, entry.Name())
		if entry.IsDir() {
			err = Cp(fs, oneSrc, oneDst)
			if err != nil {
				return err
			}
		} else {
			bytes, err := fs.ReadFile(oneSrc)
			if err != nil {
				return fmt.Errorf("embed fs read file %s failed: %v", oneSrc, err)
			}
			err = os.WriteFile(oneDst, bytes, 0o640)
			if err != nil {
				return fmt.Errorf("os write file %s failed: %v", oneDst, err)
			}
		}
	}

	return nil
}

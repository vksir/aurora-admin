package main

import (
	"aurora-admin/assets"
	"aurora-admin/internal/cmd"
	"aurora-admin/pkg/database"
	"aurora-admin/pkg/workspace"
	"github.com/spf13/cobra"
	"github.com/vksir/vkiss-lib/pkg/cache"
	"github.com/vksir/vkiss-lib/pkg/cfg"
	"github.com/vksir/vkiss-lib/pkg/log"
	"github.com/vksir/vkiss-lib/pkg/util/errutil"
	"os"
	"path/filepath"
)

func main() {
	root := cmd.NewRootCmd()

	execPath, err := os.Executable()
	errutil.Check(err)
	execDir := filepath.Dir(execPath)

	defaultConfPath := filepath.Join(execDir, "aurora-admin.toml")
	workspaceDir := execDir

	var cfgFile string
	root.PersistentFlags().StringVarP(&cfgFile, "config", "c", defaultConfPath,
		"config file")
	cfg.LogLevel.Bind(root)

	cobra.OnInitialize(func() {
		// Init Config
		cfg.Init(cfgFile, assets.DefaultConfig)

		// Init Workspace
		workspace.Init(workspaceDir)

		// Init Log
		log.Init(workspace.LogPath(), cfg.LogLevel.Get())

		// Init Others
		cache.Init(workspace.CachePath())
		database.Init(workspace.DBPath())
	})
	err = root.Execute()
	errutil.Check(err)
}

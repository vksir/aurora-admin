package main

import (
	"aurora-admin/assets"
	"aurora-admin/internal/app"
	"aurora-admin/pkg/cache"
	"aurora-admin/pkg/cfg"
	"aurora-admin/pkg/constant"
	"aurora-admin/pkg/database"
	"aurora-admin/pkg/workspace"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
	"github.com/vksir/vkiss-lib/pkg/log"
	"github.com/vksir/vkiss-lib/pkg/util/errutil"
	"github.com/vksir/vkiss-lib/pkg/util/fileutil"
	"github.com/vksir/vkiss-lib/pkg/util/installutil"
	"github.com/vksir/vkiss-lib/thirdpkg/systemctl"
)

func main() {
	cmd := &cli.Command{
		Name: "aurora",
		Commands: []*cli.Command{
			{
				Name: "install",
				Action: func(ctx context.Context, command *cli.Command) error {
					log.Init("", "debug")
					svc := &systemctl.Service{
						Name:             "aurora-admin",
						Description:      "aurora admin",
						ExecStart:        fmt.Sprintf("%s serve -c %s", constant.ExecPath, constant.ConfPath),
						RestartOnFailure: true,
					}
					return installutil.InstallService(svc, constant.ExecPath)
				},
			},
			{
				Name: "serve",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "config",
						Aliases: []string{"c"},
						Usage:   "config file",
					},
					&cli.StringFlag{
						Name:    "workspace",
						Aliases: []string{"w"},
						Value:   filepath.Dir(fileutil.Executable),
						Usage:   "workspace directory",
					},
				},
				Action: serve,
			},
		},
	}
	errutil.Check(cmd.Run(context.Background(), os.Args))
}

func serve(ctx context.Context, cmd *cli.Command) error {
	cfgFile := cmd.String("config")
	ws := cmd.String("workspace")

	// Init Config
	if cfgFile == "" {
		cfgFile = filepath.Join(ws, "config.toml")
	}
	err := cfg.Init(cfgFile, assets.DefaultConfig)
	if err != nil {
		return errutil.Wrap(err)
	}

	// Init Workspace
	workspace.Init(ws)

	// Init Log
	log.Init(workspace.LogPath(), cfg.G.LogLevel)
	log.Warn("init cfg", "path", cfgFile)
	log.Warn("init workspace", "path", ws)

	// Init Others
	cache.Init(workspace.CachePath())
	database.Init(workspace.DBPath())

	app.Run()
	return nil
}

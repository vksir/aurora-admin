package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/urfave/cli/v3"

	"aurora-admin/assets"
	"aurora-admin/internal/app"
	"aurora-admin/internal/store"
	"aurora-admin/pkg/cache"
	"aurora-admin/pkg/cfg"
	"aurora-admin/pkg/log"
	"aurora-admin/pkg/util/errutil"
	"aurora-admin/pkg/util/fileutil"
	"aurora-admin/pkg/workspace"
)

//go:embed assets/service.tmpl
var serviceTmpl string

type systemdService struct {
	Name             string
	Description      string
	WorkingDirectory string
	ExecStart        string
	RestartOnFailure bool
	User             string
	Group            string
}

func (s *systemdService) Template() string {
	return serviceTmpl
}

const (
	execPath = "/root/aurora-admin/aurora-admin"
	confPath = "/root/aurora-admin/config.toml"
)

func (s *systemdService) serviceName() string {
	return s.Name + ".service"
}

func (s *systemdService) deploy() error {
	path := filepath.Join("/etc/systemd/system/", s.serviceName())
	tmpl, err := template.New("").Parse(s.Template())
	if err != nil {
		return errutil.Wrap(err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640)
	if err != nil {
		return errutil.Wrap(err)
	}
	if err := tmpl.Execute(f, s); err != nil {
		f.Close()
		return errutil.Wrap(err)
	}
	return f.Close()
}

func (s *systemdService) systemctl(args ...string) error {
	cmd := exec.Command("systemctl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %v failed: %w\nout: %s", args, err, out)
	}
	return nil
}

func installService(svc *systemdService, execPath string) error {
	// 停止旧服务（忽略错误）
	_ = svc.systemctl("stop", svc.serviceName())

	// 复制二进制
	src, err := os.Executable()
	if err != nil {
		return errutil.Wrap(err)
	}
	if err := fileutil.MkDir(filepath.Dir(execPath)); err != nil {
		return errutil.Wrap(err)
	}
	_ = fileutil.Remove(execPath)
	if err := fileutil.Cp(src, execPath); err != nil {
		return errutil.Wrap(err)
	}
	log.Info("copy exec", "src", src, "dst", execPath)
	if err := os.Chmod(execPath, 0o755); err != nil {
		return errutil.Wrap(err)
	}
	_ = fileutil.Remove(src)

	// 部署 systemd 服务
	if err := svc.deploy(); err != nil {
		return errutil.Wrap(err)
	}
	if err := svc.systemctl("daemon-reload"); err != nil {
		return errutil.Wrap(err)
	}
	if err := svc.systemctl("enable", svc.serviceName()); err != nil {
		return errutil.Wrap(err)
	}
	if err := svc.systemctl("start", svc.serviceName()); err != nil {
		log.Error("start server failed", "cmd", "systemctl restart "+svc.Name, "err", err)
	} else {
		log.Info("install and start server success", "cmd", "systemctl status "+svc.Name)
	}
	return nil
}

func main() {
	cmd := &cli.Command{
		Name: "aurora",
		Commands: []*cli.Command{
			{
				Name: "install",
				Action: func(ctx context.Context, command *cli.Command) error {
					log.Init("", "debug")
					svc := &systemdService{
						Name:             "aurora-admin",
						Description:      "aurora admin",
						ExecStart:        fmt.Sprintf("%s serve -c %s", execPath, confPath),
						RestartOnFailure: true,
					}
					return installService(svc, execPath)
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
	errutil.Fatal(cmd.Run(context.Background(), os.Args))
}

func serve(ctx context.Context, cmd *cli.Command) error {
	cfgFile := cmd.String("config")
	ws := cmd.String("workspace")

	// Init Config
	if cfgFile == "" {
		cfgFile = filepath.Join(ws, "config.toml")
	}
	config, err := cfg.Init(cfgFile, assets.DefaultConfig)
	if err != nil {
		return errutil.Wrap(err)
	}

	// Init Workspace
	workspace.Init(ws)

	// Init Log
	log.Init(workspace.LogPath(), config.LogLevel)
	log.Warn("init cfg", "path", cfgFile)
	log.Warn("init workspace", "path", ws)

	// Init Others
	state := cache.Init(workspace.CachePath())
	db, err := store.Open(workspace.DBPath())
	if err != nil {
		return errutil.Wrap(err)
	}
	if err := db.Migrate(ctx); err != nil {
		return errutil.Wrap(err)
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Fatal: %v\n", r)
			os.Exit(1)
		}
	}()

	app.Run(config, state, db)
	return nil
}

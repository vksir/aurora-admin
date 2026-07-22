package steam

import (
	"context"
	"time"

	"aurora-admin/pkg/log"
	"aurora-admin/pkg/service"
	"aurora-admin/pkg/util/errutil"
	"os/exec"
	"path/filepath"
)

type Steamcmd struct {
	executePath     string
	logger          *log.Logger
	forceInstallDir string
}

func NewSteamcmd(path string, log *log.Logger) *Steamcmd {
	return &Steamcmd{executePath: path, logger: log.With("tag", "steamcmd")}
}

func (s *Steamcmd) SetForceInstallDir(dir string) *Steamcmd {
	s.forceInstallDir = dir
	return s
}

func (s *Steamcmd) DoWorkShopDownloadItem(ctx context.Context, appId string, publishedFileId ...string) error {
	s.logger.InfoC(ctx, "begin DoWorkShopDownloadItem", "appId", appId, "publishedFileId", publishedFileId)
	var extArgs []string
	for _, id := range publishedFileId {
		extArgs = append(extArgs, "+workshop_download_item", appId, id)
	}
	err := s.execute(ctx, extArgs)
	if err != nil {
		s.logger.ErrorC(ctx, "DoWorkShopDownloadItem failed", "appId", appId, "publishedFileId", publishedFileId, "err", err)
		return errutil.Wrap(err)
	}
	s.logger.InfoC(ctx, "DoWorkShopDownloadItem success", "appId", appId, "publishedFileId", publishedFileId)
	return nil
}

func (s *Steamcmd) DoAppUpdate(ctx context.Context, appId string) error {
	s.logger.InfoC(ctx, "begin DoAppUpdate", "appId", appId)
	extArgs := []string{"+app_update", appId, "validate"}
	err := s.execute(ctx, extArgs)
	if err != nil {
		s.logger.ErrorC(ctx, "DoAppUpdate failed", "appId", appId, "err", err)
		return errutil.Wrap(err)
	}
	s.logger.InfoC(ctx, "DoAppUpdate success", "appId", appId)
	return nil
}

func (s *Steamcmd) execute(ctx context.Context, extArgs []string) error {
	args := []string{"+login", "anonymous"}
	if s.forceInstallDir != "" {
		args = append(args, "+force_install_dir", s.forceInstallDir)
	}
	args = append(args, extArgs...)
	args = append(args, "+quit")

	exe, err := exec.LookPath(s.executePath)
	if err != nil {
		return errutil.Wrap(err)
	}
	// 使用独立的 context.Background() 而非 HTTP 请求 ctx，避免客户端超时断开时
	// exec.CommandContext 级联 kill 子进程。steamcmd 下载大型游戏服务端可能耗时数分钟。
	// 30 分钟超时作为兜底保护。
	process := service.NewSubprocess("steamcmd", exe, args).
		SetLogger(s.logger).
		SetDir(filepath.Dir(exe)).
		SetTimeout(30 * time.Minute)

	logger := s.logger.With("output", "steamcmd")
	process.RegisterOutputFunc("log", func(line []byte) {
		logger.Debug(string(line))
	})

	err = process.Start(context.Background())
	if err != nil {
		return errutil.Wrap(err)
	}

	err = process.Wait()
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

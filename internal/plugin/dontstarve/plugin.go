package dontstarve

import (
	"aurora-admin/internal/store"
	"aurora-admin/pkg/cache"
	"aurora-admin/pkg/cfg"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"aurora-admin/pkg/log"
	"aurora-admin/pkg/service"
	"aurora-admin/pkg/util/errutil"
	"aurora-admin/pkg/util/fileutil"
	"aurora-admin/pkg/steam"
)

type Plugin struct {
	svc          *service.Service
	autoupdate   *autoupdateHelper
	modupdate    *modUpdateHelper
	logger       *log.Logger
	db           *store.DB
	config       *cfg.Config
	state        *cache.State
	transferLock *sync.Mutex
	updateMu     sync.Mutex
}

func NewPlugin(db *store.DB, config *cfg.Config, state *cache.State, logger *log.Logger) *Plugin {
	p := &Plugin{
		db:           db,
		config:       config,
		state:        state,
		logger:       logger.With("tag", "dst"),
		transferLock: &sync.Mutex{},
	}
	p.svc = service.New(p, logger)
	p.autoupdate = newAutoupdateHelper(p, logger)
	p.modupdate = newModUpdateHelper(p, logger)

	p.autoupdate.RegisterCronJob()
	p.modupdate.RegisterCronJob()
	return p
}

func (p *Plugin) Name() string {
	return PluginName
}

func (p *Plugin) Service() *service.Service {
	return p.svc
}

func (p *Plugin) Install(ctx context.Context) error {
	err := steam.NewSteamcmd(p.config.SteamCmdPath, p.logger).
		SetForceInstallDir(programDir()).
		DoAppUpdate(ctx, ServerAppId)
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

func (p *Plugin) Uninstall(ctx context.Context) error {
	err := fileutil.Remove(programDir())
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

func (p *Plugin) Update(ctx context.Context) error {
	err := p.Install(ctx)
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

func (p *Plugin) RunCmd(ctx context.Context, process *service.SubProcess, cmd string) (string, error) {
	out, err := process.SendCommand(cmd, service.DefaultCommandTimeout)
	if err != nil {
		return "", errutil.Wrap(err)
	}
	p.logger.InfoC(ctx, "run cmd success", "cmd", cmd, "out", out)
	return out, nil
}

func (p *Plugin) PrepareProcess(ctx context.Context,
	processCtx context.Context) (map[string]*service.SubProcess, error) {

	archID := p.state.DontStarve.EnabledArchiveID
	if archID == "" {
		return nil, fmt.Errorf("empty archID")
	}

	_, err := p.db.GetArchive(ctx, archID)
	if err == sql.ErrNoRows {
		root := clusterRoot(archID)
		if !fileutil.Exist(root) {
			return nil, fmt.Errorf("archID %s not exist neither in db nor on disk", archID)
		}
		arch, err := p.createArchiveFromDisk(ctx, root, filepath.Base(root))
		if err != nil {
			return nil, errutil.Wrap(err)
		}

		p.logger.InfoC(ctx, "create arch from disk, change enable_arch_id",
			"old_arch", archID, "new_arch", archID)
		archID = arch.ID
		p.state.DontStarve.EnabledArchiveID = arch.ID
		p.state.Save()
	} else if err != nil {
		return nil, errutil.Wrap(err)
	}

	d := newDeployer(archID, p.db, p.state, p.logger)
	err = d.DeployBeforeStart(ctx)
	if err != nil {
		return nil, errutil.Wrap(err)
	}

	// 记录本次启动对应的游戏 buildid，供自动更新检测
	// “文件已更新但运行中进程仍旧版”的场景
	if buildID, err := readCurrentBuildID(); err == nil {
		p.state.DontStarve.LastRunBuildID = buildID
		p.state.Save()
	} else {
		p.logger.WarnC(ctx, "record last run build id failed", "err", err)
	}

	process := make(map[string]*service.SubProcess)
	for _, shard := range []string{ShardMaster, ShardCaves} {
		args := []string{"-console", "-cluster", archID, "-shard", shard}
		pp := service.NewSubprocess(shard, dstServPath(), args).
			SetLogger(p.logger).
			SetDir(filepath.Join(programDir(), "bin64"))

		logger := p.logger.With("out", shard)
		pp.RegisterOutputFunc("log", func(line []byte) {
			logger.Debug(string(line))
		})
		process[shard] = pp
	}
	return process, nil
}

func (p *Plugin) WaitActive(waitActiveCtx context.Context) bool {
	return true
}

func (p *Plugin) GracefulShutdown(ctx context.Context, process map[string]*service.SubProcess) time.Duration {
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 12*time.Second)
	defer timeoutCancel()

	wg := &sync.WaitGroup{}
	wg.Add(len(process))
	for _, proc := range process {
		go p.shutdownProcess(timeoutCtx, proc, wg)
	}
	wg.Wait()
	return 3 * time.Second
}

func (p *Plugin) shutdownProcess(ctx context.Context, process *service.SubProcess, wg *sync.WaitGroup) {
	defer wg.Done()

	subCtx, subCancel := context.WithCancel(ctx)
	defer subCancel()

	process.RegisterOutputFunc("shutdown", func(line []byte) {
		if bytes.Contains(line, []byte("Shutting down")) {
			subCancel()
		}
	})
	defer process.UnregisterOutputFunc("shutdown")

	// 进入 shutdown 流程，保存存档
	err := process.Interrupt()
	if err != nil {
		p.logger.ErrorC(ctx, "interrupt process failed", "err", err)
	}

	// 等候 shutdown 完成，存档已保存
	select {
	case <-subCtx.Done():
	}

	// 终止进程
	err = process.Interrupt()
	if err != nil {
		p.logger.ErrorC(ctx, "interrupt process failed", "err", err)
	}
}

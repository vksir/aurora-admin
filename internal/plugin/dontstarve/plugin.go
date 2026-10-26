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
	"sync/atomic"
	"time"

	"aurora-admin/pkg/log"
	"aurora-admin/pkg/service"
	"aurora-admin/pkg/steam"
	"aurora-admin/pkg/util/errutil"
	"aurora-admin/pkg/util/fileutil"
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
	// logGeneration 在每次启动前递增，用于让已建立的服务日志流丢弃上一轮内容
	logGeneration atomic.Int64
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

	// 旧存档兼容：若数据库里没有模组关联（早期版本数据），先按存档目录的
	// modoverrides.lua 回填，避免启动时把已启用的模组全部关掉。
	_, err := p.GetArchive(ctx, archID)
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

	// 启动前先确认没有同一集群的服务端进程在跑，避免面板重启后
	// 误点启动又拉起一套进程（端口冲突 → 僵进程）。
	if err := p.ensureNoRunningServer(archID); err != nil {
		return nil, err
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

		// 每次启动清空服务日志：Web「服务日志」页展示的是本次运行，不是归档。
		// 这里已经过了 ensureNoRunningServer / DeployBeforeStart，前置步骤失败不会误清。
		if err := resetShardLog(shard); err != nil {
			return nil, errutil.Wrap(err)
		}

		// relay 模式：stdin 用命名管道，stdout/stderr 落到独立日志文件，
		// 使面板重启后能重新接管控制台；非 Linux 平台自动回退到管道模式。
		pp.SetRelay(shardFIFOPath(shard), shardLogPath(shard))

		// relay 模式下输出已经完整落在独立日志文件里，不再重复写应用日志
		if pp.LogPath() == "" {
			logger := p.logger.With("out", shard)
			pp.RegisterOutputFunc("log", func(line []byte) {
				logger.Debug(string(line))
			})
		}
		process[shard] = pp
	}
	// 服务日志代际 +1：已连接的服务日志流据此丢弃上一轮内容
	p.logGeneration.Add(1)
	return process, nil
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

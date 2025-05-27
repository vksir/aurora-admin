package dontstarve

import (
	"aurora-admin/ent"
	"aurora-admin/pkg/cache"
	"aurora-admin/pkg/cfg"
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/vksir/vkiss-lib/pkg/log"
	"github.com/vksir/vkiss-lib/pkg/service"
	"github.com/vksir/vkiss-lib/pkg/util/errutil"
	"github.com/vksir/vkiss-lib/pkg/util/fileutil"
	"github.com/vksir/vkiss-lib/thirdpkg/steam"
)

type Plugin struct {
	svc          *service.Service
	autoupdate   *autoupdateHelper
	logger       *log.Logger
	db           *ent.Client
	ph           *pathHelper
	transferLock *sync.Mutex
}

func NewPlugin(db *ent.Client, logger *log.Logger) *Plugin {
	p := &Plugin{
		db:           db,
		logger:       logger.With("tag", "dst"),
		ph:           newPathHelper(),
		transferLock: &sync.Mutex{},
	}
	p.svc = service.New(p, logger)
	p.autoupdate = newAutoupdateHelper(p, logger)

	p.autoupdate.RegisterCronJob()
	return p
}

func (p *Plugin) Name() string {
	return PluginName
}

func (p *Plugin) Service() *service.Service {
	return p.svc
}

func (p *Plugin) Install(ctx context.Context) error {
	err := steam.NewSteamcmd(cfg.G.SteamCmdPath, p.logger).
		SetForceInstallDir(p.ph.ProgramDir()).
		DoAppUpdate(ctx, ServerAppId)
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

func (p *Plugin) Uninstall(ctx context.Context) error {
	err := fileutil.Remove(p.ph.ProgramDir())
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
	builder := strings.Builder{}
	process.RegisterOutputFunc("run_cmd", func(line []byte) {
		builder.Write(line)
	})

	cmd = strings.TrimRight(cmd, "\n")
	_, err := process.Write([]byte(fmt.Sprintf("[BEGIN_CMD]\n%s\n[END_CMD]\n", cmd)))
	if err != nil {
		process.UnregisterOutputFunc("run_cmd")
		return "", errutil.Wrap(err)
	}
	time.Sleep(500 * time.Millisecond)
	process.UnregisterOutputFunc("run_cmd")

	rawOut := builder.String()
	pattern := regexp.MustCompile(`(?ms)BEGIN_CMD(.*?)END_CMD`)
	res := pattern.FindStringSubmatch(rawOut)
	if len(res) == 0 {
		return "", fmt.Errorf("regex cmd out failed: %s", rawOut)
	}
	out := res[1]
	p.logger.InfoC(ctx, "run cmd success", "cmd", cmd, "out", out)
	return out, nil
}

func (p *Plugin) PrepareProcess(ctx context.Context,
	processCtx context.Context) (map[string]*service.SubProcess, error) {

	archID := cache.G.DontStarve.EnabledArchiveID
	if archID == "" {
		return nil, fmt.Errorf("empty archID")
	}

	_, err := p.db.DontStarveArchive.Get(ctx, archID)
	if ent.IsNotFound(err) {
		cph := newClusterPathHelper(p.ph, archID)
		if !fileutil.Exist(cph.Root()) {
			return nil, fmt.Errorf("archID %s not exist neither in db nor on disk", archID)
		}
		arch, err := p.createArchiveFromDisk(ctx, cph.Root(), filepath.Base(cph.Root()))
		if err != nil {
			return nil, errutil.Wrap(err)
		}

		p.logger.InfoC(ctx, "create arch from disk, change enable_arch_id",
			"old_arch", archID, "new_arch", archID)
		archID = arch.ID
		cache.G.DontStarve.EnabledArchiveID = arch.ID
		cache.Save()
	} else if err != nil {
		return nil, errutil.Wrap(err)
	}

	d := newDeployer(archID, p.db, p.logger)
	err = d.DeployBeforeStart(ctx)
	if err != nil {
		return nil, errutil.Wrap(err)
	}

	process := make(map[string]*service.SubProcess)
	for _, shard := range []string{ShardMaster, ShardCaves} {
		args := []string{"-console", "-cluster", archID, "-shard", shard}
		pp := service.NewSubprocess(shard, p.ph.DstServPath(), args).
			SetLogger(p.logger).
			SetDir(filepath.Join(p.ph.ProgramDir(), "bin64"))

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

package donstarve

import (
	"aurora-admin/internal/controller/api/svcapi"
	"aurora-admin/internal/entity/dstety"
	"aurora-admin/internal/repo"
	"context"
	"errors"
	"fmt"
	"github.com/vksir/vkiss-lib/pkg/log"
	"github.com/vksir/vkiss-lib/pkg/subprocess"
	"github.com/vksir/vkiss-lib/pkg/util/errutil"
	"github.com/vksir/vkiss-lib/pkg/util/fileutil"
	"github.com/vksir/vkiss-lib/thirdpkg/steamcmd"
	"gorm.io/gorm"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Plugin struct {
	process  map[string]*subprocess.SubProcess
	archRepo *repo.Repo[dstety.DontStarveArchive]
	modRepo  *repo.Repo[dstety.DontStarveMod]
	logger   *log.Logger
	ph       *PathHelper
}

func NewPlugin(db *gorm.DB, l *log.Logger) *Plugin {
	return &Plugin{
		process:  make(map[string]*subprocess.SubProcess),
		archRepo: repo.NewRepo[dstety.DontStarveArchive](db),
		modRepo:  repo.NewRepo[dstety.DontStarveMod](db),
		logger:   l.With("tag", "dst"),
		ph:       NewPathHelper(),
	}
}

func (d *Plugin) Status() svcapi.Status {
	//TODO implement me
	panic("implement me")
}

func (d *Plugin) Start(ctx context.Context) error {
	archId := cacheEnabledArchiveId.Get()

	if !d.archRepo.Exist(ctx, archId) {
		cp := NewClusterPath(d.ph, archId)
		if !fileutil.Exist(cp.Root) {
			return fmt.Errorf("archId %s not exist neither in db nor on disk", archId)
		}

		d.logger.InfoC(ctx, "begin createArchiveFromDisk")
		err := d.createArchiveFromDisk(ctx, &archId)
		if err != nil {
			return errutil.Wrap(err)
		}
		cacheEnabledArchiveId.Set(archId)
	}

	arch, err := d.archRepo.GetOneWithNestedPreload(ctx, archId, "Mods.Mod")
	if err != nil {
		return errutil.Wrap(err)
	}

	err = d.deployModSetup(ctx, &arch)
	if err != nil {
		return errutil.Wrap(err)
	}
	err = d.deployToken(ctx, &arch)
	if err != nil {
		return errutil.Wrap(err)
	}

	return d.start(ctx, archId)
}

func (d *Plugin) start(ctx context.Context, archId string) error {
	d.process = make(map[string]*subprocess.SubProcess)

	for _, shard := range []string{"Master", "Caves"} {
		args := []string{"-console", "-cluster", archId, "-shard", shard}
		p := subprocess.New(shard, d.ph.DstServPath(), args).
			SetLogger(shard, d.logger).
			SetDir(filepath.Join(d.ph.ProgramDir(), "bin64"))

		logger := d.logger.With("out", shard)
		p.RegisterOutFunc("log", func(line *string) {
			logger.Debug(*line)
		})

		err := p.Start(ctx)
		if err != nil {
			d.logger.ErrorC(ctx, "start failed, begin kill all processes", "shard", shard)
			if err := d.Stop(ctx); err != nil {
				d.logger.ErrorC(ctx, "stop failed")
			}
			return errutil.Wrap(err)
		}
		d.process[shard] = p
	}
	return nil
}

func (d *Plugin) Stop(ctx context.Context) error {
	d.logger.InfoC(ctx, "begin graceful shutdown process")
	for _, p := range d.process {
		err := p.Interrupt()
		if err != nil {
			d.logger.ErrorC(ctx, "interrupt failed", "process", p.Name)
		}
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	for _, p := range d.process {
		select {
		case <-p.Ctx().Done():
		case <-ctx.Done():
		}
	}

	if err := ctx.Err(); !errors.Is(err, context.DeadlineExceeded) {
		d.process = make(map[string]*subprocess.SubProcess)
		d.logger.InfoC(ctx, "graceful shutdown process success", "err", err)
		return nil
	}

	d.logger.InfoC(ctx, "graceful shutdown process timeout, begin kill")
	for _, p := range d.process {
		if err := p.Kill(); err != nil {
			d.logger.ErrorC(ctx, "kill failed", "process", p.Name)
		}
	}
	d.process = make(map[string]*subprocess.SubProcess)
	return errutil.Wrap(subprocess.ErrGraceFulShutdownTimeout)
}

func (d *Plugin) Restart(ctx context.Context) error {
	err := d.Stop(ctx)
	if err != nil {
		return errutil.Wrap(err)
	}
	return d.Start(ctx)
}

func (d *Plugin) Install(ctx context.Context) error {
	d.logger.WarnC(ctx, "begin install")
	exe, err := exec.LookPath(steamcmd.CfgSteamcmdPath.Get())
	if err != nil {
		return errutil.Wrap(err)
	}
	args := []string{"+force_install_dir", d.ph.ProgramDir(),
		"+login", "anonymous",
		"+app_update", "343050", "validate",
		"+quit"}

	p := subprocess.New("steam", exe, args).
		SetLogger("steam", d.logger).
		SetDir(filepath.Dir(exe))

	logger := d.logger.With("out", "steam")
	p.RegisterOutFunc("log", func(line *string) {
		logger.Debug(*line)
	})

	err = p.Start(ctx)
	if err != nil {
		return errutil.Wrap(err)
	}

	go func() {
		err = p.Wait()
		if err != nil {
			d.logger.ErrorC(ctx, "install failed", "err", err)
			return
		}
		d.logger.InfoC(ctx, "install success")
	}()
	return nil
}

func (d *Plugin) Uninstall(ctx context.Context) error {
	d.logger.WarnC(ctx, "begin install")
	err := fileutil.Rm(d.ph.ProgramDir())
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

func (d *Plugin) Upgrade(ctx context.Context) error {
	d.logger.WarnC(ctx, "begin upgrade")
	return d.Install(ctx)
}

func (d *Plugin) RunCmd(ctx context.Context, p *subprocess.SubProcess, cmd string) (string, error) {

	builder := strings.Builder{}
	p.RegisterOutFunc("run_cmd", func(line *string) {
		builder.WriteString(*line)
	})

	cmd = strings.TrimRight(cmd, "\n")
	_, err := p.Write([]byte(fmt.Sprintf("[BEGIN_CMD]\n%s\n[END_CMD]\n", cmd)))
	if err != nil {
		p.UnregisterOutFunc("run_cmd")
		return "", errutil.Wrap(err)
	}
	time.Sleep(500 * time.Millisecond)
	p.UnregisterOutFunc("run_cmd")

	rawOut := builder.String()
	pattern := regexp.MustCompile(`(?ms)BEGIN_CMD(.*?)END_CMD`)
	res := pattern.FindStringSubmatch(rawOut)
	if len(res) == 0 {
		return "", fmt.Errorf("regex cmd out failed: %s", rawOut)
	}
	out := res[1]
	d.logger.InfoC(ctx, "run cmd success", "cmd", cmd, "out", out)
	return out, nil
}

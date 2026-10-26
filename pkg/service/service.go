package service

import (
	"aurora-admin/pkg/log"
	"aurora-admin/pkg/util/errutil"
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrBusy           = errors.New("busy")
	ErrRunningNow     = errors.New("running now")
	ErrAlreadyStarted = errors.New("already started")
	ErrNotStartedYet  = errors.New("not started yet")
)

type Status int

var statusNames = []string{
	"Inactive",
	"Active",
	"Starting",
	"Stopping",
	"WaitingActive",
	"Abnormal",
	"External",
}

func (s Status) String() string {
	if int(s) < 0 || int(s) >= len(statusNames) {
		return "Unknown"
	}
	return statusNames[s]
}

const (
	StatusInactive Status = iota
	StatusActive
	StatusStarting
	StatusStopping
	// StatusWaitingActive 已不再使用：启动流程统一停留在 StatusStarting，
	// 直到 WaitActive 确认服务就绪后才转为 StatusActive。
	// 保留该值仅为兼容旧版本前端的状态文案。
	StatusWaitingActive
	StatusAbnormal
	// StatusExternal 表示进程在系统里运行，但不归当前面板实例管理
	// （例如面板重启后遗留的孤儿进程），只能停止，不能下发控制台命令。
	StatusExternal
)

type Interface interface {
	PrepareProcess(ctx context.Context, processCtx context.Context) (map[string]*SubProcess, error)
	// WaitActive 等待服务真正启动完成：就绪返回 true，启动失败/进程退出返回 false。
	WaitActive(waitActiveCtx context.Context, process map[string]*SubProcess) bool
	GracefulShutdown(ctx context.Context, process map[string]*SubProcess) time.Duration
	Install(ctx context.Context) error
	Uninstall(ctx context.Context) error
	Update(ctx context.Context) error
}

// Detached 由实现了「脱离内存管理但仍可停止」能力的插件实现。
// 面板重启后遗留的孤儿进程靠它兜底停止。
type Detached interface {
	DetachedRunning() bool
	StopDetached(ctx context.Context) error
}

// Lifecycle 由需要在服务启动/停止时持久化运行态的插件实现。
type Lifecycle interface {
	OnServiceStarted(ctx context.Context, process map[string]*SubProcess) error
	OnServiceStopped(ctx context.Context)
}

type serviceRuntime struct {
	process          map[string]*SubProcess
	processCtx       context.Context
	processCancel    context.CancelFunc
	waitActiveCtx    context.Context
	waitActiveCancel context.CancelFunc
}

func (r *serviceRuntime) Close() {
	r.processCancel()
	r.waitActiveCancel()
}

func newServiceRuntime() *serviceRuntime {
	processCtx, processCancel := context.WithCancel(context.Background())
	waitActiveCtx, waitActiveCancel := context.WithCancel(context.Background())
	return &serviceRuntime{
		processCtx:       processCtx,
		processCancel:    processCancel,
		waitActiveCtx:    waitActiveCtx,
		waitActiveCancel: waitActiveCancel,
	}
}

type Service struct {
	instance Interface
	logger   *log.Logger

	runtime  *serviceRuntime
	status   Status
	busyLock sync.Mutex
}

func New(ins Interface, logger *log.Logger) *Service {
	return &Service{instance: ins, logger: logger}
}

func (s *Service) Status() Status {
	st := s.status
	rt := s.runtime
	if st == StatusActive && rt != nil {
		alive := false
		for _, p := range rt.process {
			if p.Alive() {
				alive = true
				break
			}
		}
		if !alive {
			st = StatusInactive
		}
	}
	if st == StatusInactive && s.detachedRunning() {
		return StatusExternal
	}
	return st
}

func (s *Service) Running() bool {
	return s.runtime != nil
}

func (s *Service) detachedRunning() bool {
	d, ok := s.instance.(Detached)
	return ok && d.DetachedRunning()
}

// Adopt 接管已经在运行的进程：直接把 runtime 装上并置为 Active。
// 由插件在面板启动时调用。
func (s *Service) Adopt(process map[string]*SubProcess) {
	if len(process) == 0 {
		return
	}
	s.busyLock.Lock()
	defer s.busyLock.Unlock()
	if s.Running() {
		return
	}
	svcRt := newServiceRuntime()
	svcRt.process = process
	s.runtime = svcRt
	s.status = StatusActive
	s.logger.Info("adopted running processes", "count", len(process))
	go s.watchAdopted(svcRt)
}

// watchAdopted 监控接管过来的进程：全部退出后释放 runtime，
// 使面板可以重新启动服务；部分退出则标记 Abnormal。
func (s *Service) watchAdopted(rt *serviceRuntime) {
	for _, proc := range rt.process {
		go func(p *SubProcess) {
			_ = p.Wait()

			s.busyLock.Lock()
			defer s.busyLock.Unlock()
			if s.runtime != rt {
				return
			}

			for _, other := range rt.process {
				if other != p && other.Alive() {
					s.status = StatusAbnormal
					s.logger.Warn("adopted subprocess exited, service abnormal", "subprocess", p.Name())
					return
				}
			}

			s.logger.Warn("all adopted subprocesses exited, mark inactive")
			rt.Close()
			s.runtime = nil
			s.status = StatusInactive
			if lc, ok := s.instance.(Lifecycle); ok {
				lc.OnServiceStopped(context.Background())
			}
		}(proc)
	}
}

func (s *Service) Control(f func(process map[string]*SubProcess) error) error {
	if !s.busyLock.TryLock() {
		return ErrBusy
	}
	defer s.busyLock.Unlock()
	if !s.Running() {
		return ErrNotStartedYet
	}
	return f(s.runtime.process)
}

func (s *Service) Start(ctx context.Context) error {
	if !s.busyLock.TryLock() {
		return ErrBusy
	}
	defer s.busyLock.Unlock()
	ctx = log.AppendCtx(ctx, "tag", "starting")
	return s.start(ctx)
}

func (s *Service) Stop(ctx context.Context) error {
	if !s.busyLock.TryLock() {
		return ErrBusy
	}
	defer s.busyLock.Unlock()
	ctx = log.AppendCtx(ctx, "tag", "stopping")
	return s.stopAny(ctx)
}

func (s *Service) Restart(ctx context.Context) error {
	if !s.busyLock.TryLock() {
		return ErrBusy
	}
	defer s.busyLock.Unlock()
	ctx = log.AppendCtx(ctx, "tag", "restarting")
	return s.restart(ctx)
}

func (s *Service) Install(ctx context.Context) error {
	if !s.busyLock.TryLock() {
		return ErrBusy
	}
	defer s.busyLock.Unlock()
	ctx = log.AppendCtx(ctx, "tag", "installing")
	s.logger.InfoC(ctx, "begin install")
	return s.instance.Install(ctx)
}

func (s *Service) Uninstall(ctx context.Context) error {
	if !s.busyLock.TryLock() {
		return ErrBusy
	}
	defer s.busyLock.Unlock()

	ctx = log.AppendCtx(ctx, "tag", "uninstalling")
	s.logger.InfoC(ctx, "begin uninstall")
	if err := s.stopAny(ctx); err != nil {
		return errutil.Wrap(err)
	}
	err := s.instance.Uninstall(ctx)
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

func (s *Service) Update(ctx context.Context) error {
	if !s.busyLock.TryLock() {
		return ErrBusy
	}
	defer s.busyLock.Unlock()

	ctx = log.AppendCtx(ctx, "tag", "upgrading")
	wasRunning := s.Running() || s.detachedRunning()
	if wasRunning {
		if err := s.stopAny(ctx); err != nil {
			return errutil.Wrap(err)
		}
	}
	err := s.instance.Update(ctx)
	if err != nil {
		return errutil.Wrap(err)
	}
	if wasRunning {
		err = s.start(ctx)
		if err != nil {
			return errutil.Wrap(err)
		}
	}
	return nil
}

func (s *Service) start(ctx context.Context) error {
	if s.Running() {
		return ErrAlreadyStarted
	}

	s.logger.InfoC(ctx, "begin start")
	var err error
	s.status = StatusStarting
	svcRt := newServiceRuntime()
	svcRt.process, err = s.instance.PrepareProcess(ctx, svcRt.processCtx)
	if err != nil {
		// 准备阶段失败时回滚状态，避免界面一直显示 "Starting"
		s.status = StatusInactive
		svcRt.Close()
		return errutil.Wrap(err)
	}

	err = s.startProcess(ctx, svcRt)
	if err != nil {
		// 进程拉起失败时回滚状态，避免界面一直显示 "Starting"
		s.status = StatusInactive
		svcRt.Close()
		return errutil.Wrap(err)
	}

	// 进程已拉起，但世界还在加载；保持 Starting 直到 WaitActive 确认就绪。
	go s.waitActive(svcRt)

	s.runtime = svcRt
	if lc, ok := s.instance.(Lifecycle); ok {
		if err := lc.OnServiceStarted(ctx, svcRt.process); err != nil {
			s.logger.WarnC(ctx, "persist runtime state failed", "err", err)
		}
	}
	return nil
}

// waitActive 使用上下文 serviceRuntime.waitActiveCtx
// 如何确保没有线程冲突？
// 1. waitActive 启动后，在正常退出前只可能遇到 stopProcess 线程
// 2. stopProcess 会先调用 serviceRuntime.waitActiveCancel 关闭 waitActive 线程然后再去停进程
func (s *Service) waitActive(svcRt *serviceRuntime) {
	waitActiveCtx := svcRt.waitActiveCtx
	ok := s.instance.WaitActive(waitActiveCtx, svcRt.process)
	if errors.Is(waitActiveCtx.Err(), context.Canceled) {
		s.logger.Warn("context canceled, waitActive exited")
		return
	}

	if !ok {
		s.status = StatusAbnormal
		s.logger.Warn("set status to abnormal, waitActive exited")
		return
	}

	s.status = StatusActive
	s.logger.Warn("set status to active, waitActive exited")
}

func (s *Service) stop(ctx context.Context) {
	if !s.Running() {
		s.logger.DebugC(ctx, "process has not started yet, no need stop")
		return
	}

	s.logger.InfoC(ctx, "begin stop")
	s.status = StatusStopping
	s.stopProcess(ctx, s.runtime)
	s.status = StatusInactive
	s.runtime.Close()
	s.runtime = nil
	if lc, ok := s.instance.(Lifecycle); ok {
		lc.OnServiceStopped(ctx)
	}
}

// stopAny 优先停内存中管理的进程；面板重启后遗留的孤儿进程走插件兜底停止。
func (s *Service) stopAny(ctx context.Context) error {
	if s.Running() {
		s.stop(ctx)
		return nil
	}
	d, ok := s.instance.(Detached)
	if ok && d.DetachedRunning() {
		s.logger.InfoC(ctx, "service not managed in memory, stop detached processes")
		return d.StopDetached(ctx)
	}
	s.logger.DebugC(ctx, "process has not started yet, no need stop")
	return nil
}

func (s *Service) restart(ctx context.Context) error {
	s.logger.InfoC(ctx, "begin restart")
	if err := s.stopAny(ctx); err != nil {
		return errutil.Wrap(err)
	}
	err := s.start(ctx)
	if err != nil {
		s.logger.ErrorC(ctx, "restart failed at starting", "err", err)
		return errutil.Wrap(err)
	}
	s.logger.InfoC(ctx, "restart success")
	return nil
}

func (s *Service) startProcess(ctx context.Context, svcRt *serviceRuntime) error {
	s.logger.InfoC(ctx, "begin start process")
	for name, p := range svcRt.process {
		err := p.Start(svcRt.processCtx)
		if err != nil {
			s.logger.ErrorC(ctx, "start failed, begin kill all processes", "process", name, "err", err)
			s.stopProcess(ctx, svcRt)
			return errutil.Wrap(err)
		}
	}
	s.logger.InfoC(ctx, "start process success")
	return nil
}

func (s *Service) stopProcess(ctx context.Context, svcRt *serviceRuntime) {
	s.logger.InfoC(ctx, "begin stop process")
	waitTime := s.instance.GracefulShutdown(ctx, svcRt.process)

	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), waitTime)
	defer timeoutCancel()
	for _, p := range svcRt.process {
		if p.Ctx() != nil {
			select {
			case <-p.Ctx().Done():
			case <-timeoutCtx.Done():
			}
		}
	}
	if timeoutCtx.Err() == nil {
		s.logger.InfoC(ctx, "graceful shutdown process success")
		return
	}

	s.logger.ErrorC(ctx, "graceful shutdown process timeout, begin kill")
	for _, p := range svcRt.process {
		_ = p.Kill()
	}
	svcRt.processCancel()
}

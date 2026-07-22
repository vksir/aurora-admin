package service

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"aurora-admin/pkg/log"
	"aurora-admin/pkg/util/errutil"
)

type OutputFunc func([]byte)

type SubProcess struct {
	name string
	exec string
	args []string
	env  []string
	dir  string

	log      *log.Logger
	timeout  time.Duration
	outFuncs map[string]OutputFunc
	mu       sync.RWMutex

	cmd      *exec.Cmd
	ctx      context.Context
	cancel   context.CancelFunc
	stdin    io.Writer
	stdout   io.Reader
	stderr   io.Reader
	waitChan chan error
}

func NewSubprocess(name, exec string, args []string) *SubProcess {
	p := &SubProcess{
		name:     name,
		exec:     exec,
		args:     args,
		outFuncs: make(map[string]OutputFunc),
		waitChan: make(chan error, 1),
	}
	return p
}

func (p *SubProcess) SetTimeout(d time.Duration) *SubProcess {
	p.timeout = d
	return p
}

func (p *SubProcess) SetLogger(l *log.Logger) *SubProcess {
	p.log = l.With("subprocess", p.name)
	return p
}

func (p *SubProcess) SetEnv(env []string) *SubProcess {
	p.env = env
	return p
}

func (p *SubProcess) SetDir(dir string) *SubProcess {
	p.dir = dir
	return p
}

func (p *SubProcess) RegisterOutputFunc(name string, f OutputFunc) {
	p.log.Info("begin register outFunc", "name", name, "outFunc", f)
	p.mu.Lock()
	p.outFuncs[name] = f
	p.mu.Unlock()
	p.log.Info("end register outFunc", "name", name, "outFunc", f)
}

func (p *SubProcess) UnregisterOutputFunc(name string) {
	p.log.Info("begin unregister outFunc", "name", name)
	p.mu.Lock()
	delete(p.outFuncs, name)
	p.mu.Unlock()
	p.log.Info("end unregister outFunc", "name", name)
}

func (p *SubProcess) Name() string {
	return p.name
}

func (p *SubProcess) RawCmd() string {
	return strings.Join(append([]string{p.exec}, p.args...), " ")
}

func (p *SubProcess) Ctx() context.Context {
	return p.ctx
}

func (p *SubProcess) Start(ctx context.Context) error {
	if p.cmd != nil && p.cmd.Process != nil {
		return errutil.Wrap(ErrAlreadyStarted)
	}

	if p.timeout != 0 {
		p.ctx, p.cancel = context.WithTimeout(ctx, p.timeout)
	} else {
		p.ctx, p.cancel = context.WithCancel(ctx)
	}
	p.cmd = exec.CommandContext(p.ctx, p.exec, p.args...)
	if len(p.env) != 0 {
		p.cmd.Env = p.env
	}
	if p.dir != "" {
		p.cmd.Dir = p.dir
	}

	p.log.Info("begin start subprocess",
		"path", p.cmd.Path,
		"args", p.cmd.Args,
		"env", p.env,
		"dir", p.dir,
		"timeout", p.timeout)

	var err error
	p.stdin, err = p.cmd.StdinPipe()
	if err != nil {
		return errutil.Wrap(err)
	}
	p.stdout, err = p.cmd.StdoutPipe()
	if err != nil {
		return errutil.Wrap(err)
	}
	p.stderr, err = p.cmd.StderrPipe()
	if err != nil {
		return errutil.Wrap(err)
	}

	go p.loopOutput()
	if err = p.cmd.Start(); err != nil {
		return errutil.Wrap(err)
	}
	go p.blockWait()
	return nil
}

func (p *SubProcess) Interrupt() error {
	if p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	p.log.Info("begin interrupt subprocess")
	err := p.cmd.Process.Signal(os.Interrupt)
	if err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

func (p *SubProcess) Kill() error {
	if p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	p.log.Info("begin stop subprocess")
	p.cancel()
	return p.ctx.Err()
}

func (p *SubProcess) Write(content []byte) (n int, err error) {
	if p.cmd.Process == nil {
		return 0, errutil.Wrap(ErrNotStartedYet)
	}
	return p.stdin.Write(content)
}

func (p *SubProcess) Wait() error {
	if p.cmd.Process == nil {
		return errutil.Wrap(ErrNotStartedYet)
	}
	p.log.Info("begin wait subprocess")
	return <-p.waitChan
}

func (p *SubProcess) blockWait() {
	log.Info("begin block wait subprocess")
	err := p.cmd.Wait()
	p.waitChan <- err
	p.cancel()
	log.Warn("subprocess stopped", "err", err)
}

func (p *SubProcess) loopOutput() {
	p.log.Info("begin loop output")

	scanner := bufio.NewScanner(io.MultiReader(p.stdout, p.stderr))

	for scanner.Scan() {
		out := scanner.Bytes()

		select {
		case <-p.ctx.Done():
			p.log.Warn("exit loop output", "err", p.ctx.Err())
			return
		default:
		}

		p.mu.RLock()
		for _, f := range p.outFuncs {
			f(out)
		}
		p.mu.RUnlock()
	}
}

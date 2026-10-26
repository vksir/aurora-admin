package service

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"aurora-admin/pkg/log"
	"aurora-admin/pkg/util/errutil"
)

type OutputFunc func([]byte)

// 单个 shard 的控制台日志上限，超过后截断，避免无限增长。
const maxShardLogSize = 64 << 20 // 64MB

const (
	// tailPollInterval 日志 tail 的空闲轮询间隔
	tailPollInterval = 80 * time.Millisecond
	// relayPollInterval 重连模式下进程存活检查间隔
	relayPollInterval = 300 * time.Millisecond
)

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
	stdin    io.WriteCloser
	stdout   io.Reader
	stderr   io.Reader
	waitChan chan error

	// relay 模式：stdin 用命名管道、stdout/stderr 重定向到日志文件，
	// 这样面板重启后可以重新接管（attached）。
	relay      bool
	attached   bool
	fifoPath   string
	logPath    string
	pid        int
	startToken string
}

func NewSubprocess(name, exec string, args []string) *SubProcess {
	p := &SubProcess{
		name:     name,
		exec:     exec,
		args:     args,
		log:      log.DefaultLogger(),
		outFuncs: make(map[string]OutputFunc),
		waitChan: make(chan error, 1),
	}
	return p
}

// NewAttachedSubprocess 构造一个用于「接管已运行进程」的实例。
// 需要先调用 Attach 才会真正建立 FIFO 写端和日志 tail。
func NewAttachedSubprocess(name string, pid int, startToken, fifoPath, logPath string) *SubProcess {
	p := NewSubprocess(name, "", nil)
	p.relay = true
	p.attached = true
	p.pid = pid
	p.startToken = startToken
	p.fifoPath = fifoPath
	p.logPath = logPath
	return p
}

// SetRelay 启用 relay 模式（FIFO stdin + 日志文件 stdout/stderr）。
// 非 Linux 平台会在 Start 时自动回退到普通管道模式。
func (p *SubProcess) SetRelay(fifoPath, logPath string) *SubProcess {
	p.relay = true
	p.fifoPath = fifoPath
	p.logPath = logPath
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
	p.mu.Lock()
	p.outFuncs[name] = f
	p.mu.Unlock()
}

func (p *SubProcess) UnregisterOutputFunc(name string) {
	p.mu.Lock()
	delete(p.outFuncs, name)
	p.mu.Unlock()
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

// Pid 返回进程 PID（未启动时为 0）。
func (p *SubProcess) Pid() int {
	if p.relay {
		return p.pid
	}
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	return 0
}

// StartToken 返回进程启动标识，用于重连时防 PID 复用。
func (p *SubProcess) StartToken() string {
	return p.startToken
}

// FIFOPath 返回 relay 模式下 stdin 命名管道路径。
func (p *SubProcess) FIFOPath() string { return p.fifoPath }

// LogPath 返回 relay 模式下 stdout/stderr 日志文件路径。
func (p *SubProcess) LogPath() string { return p.logPath }

// Alive 判断进程是否仍在运行。
func (p *SubProcess) Alive() bool {
	if p.relay {
		return p.pid > 0 && processAliveWithToken(p.pid, p.startToken)
	}
	return p.ctx != nil && p.ctx.Err() == nil
}

func (p *SubProcess) Start(ctx context.Context) error {
	if p.attached {
		return errutil.Wrap(ErrAlreadyStarted)
	}
	if p.cmd != nil && p.cmd.Process != nil {
		return errutil.Wrap(ErrAlreadyStarted)
	}

	if p.timeout != 0 {
		p.ctx, p.cancel = context.WithTimeout(ctx, p.timeout)
	} else {
		p.ctx, p.cancel = context.WithCancel(ctx)
	}

	if p.relay {
		err := p.startRelay()
		if err == nil {
			return nil
		}
		if !errors.Is(err, ErrRelayUnsupported) {
			return errutil.Wrap(err)
		}
		// 非 Linux（本地开发）回退到普通管道模式
		p.relay = false
	}

	return p.startPiped()
}

func (p *SubProcess) startPiped() error {
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
	p.pid = p.cmd.Process.Pid
	go p.blockWait()
	return nil
}

func (p *SubProcess) startRelay() error {
	if err := os.MkdirAll(filepath.Dir(p.fifoPath), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p.logPath), 0o755); err != nil {
		return err
	}
	if err := makeFIFO(p.fifoPath); err != nil {
		return err
	}

	// 启动前清理过大的历史日志，保证 tail 从头读且文件不会无限增长
	if stat, err := os.Stat(p.logPath); err == nil && stat.Size() > maxShardLogSize {
		if err := os.Truncate(p.logPath, 0); err != nil {
			return err
		}
	}

	logFile, err := os.OpenFile(p.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640)
	if err != nil {
		return err
	}

	fifoFile, err := openFIFOReadWrite(p.fifoPath)
	if err != nil {
		_ = logFile.Close()
		return err
	}

	p.cmd = exec.CommandContext(p.ctx, p.exec, p.args...)
	if len(p.env) != 0 {
		p.cmd.Env = p.env
	}
	if p.dir != "" {
		p.cmd.Dir = p.dir
	}
	p.cmd.Stdin = fifoFile
	p.cmd.Stdout = logFile
	p.cmd.Stderr = logFile

	p.log.Info("begin start subprocess (relay)",
		"path", p.cmd.Path,
		"args", p.cmd.Args,
		"dir", p.dir,
		"fifo", p.fifoPath,
		"log", p.logPath)

	if err := p.cmd.Start(); err != nil {
		_ = logFile.Close()
		_ = fifoFile.Close()
		_ = os.Remove(p.fifoPath)
		return err
	}

	// 子进程已持有日志文件句柄，父进程可以关闭自己的副本
	_ = logFile.Close()

	// 面板侧复用同一个 O_RDWR 句柄写 FIFO；子进程也持有写端，不会 EOF
	p.stdin = fifoFile
	p.pid = p.cmd.Process.Pid
	if token, err := processStartToken(p.pid); err == nil {
		p.startToken = token
	} else {
		p.log.Warn("read process start token failed", "err", err)
	}

	p.startTail(false)
	go p.blockWaitRelay()
	return nil
}

// Attach 接管一个已经在运行的进程：重开 FIFO 写端并从日志末尾开始 tail。
func (p *SubProcess) Attach(ctx context.Context) error {
	if !p.attached || !p.relay {
		return errutil.Wrap(ErrNotStartedYet)
	}
	if !processAliveWithToken(p.pid, p.startToken) {
		return errutil.Wrap(ErrNotStartedYet)
	}

	p.ctx, p.cancel = context.WithCancel(ctx)

	fifoFile, err := openFIFOWrite(p.fifoPath)
	if err != nil {
		p.cancel()
		return errutil.Wrap(err)
	}
	p.stdin = fifoFile

	p.startTail(true)
	go p.blockWaitAttached()
	return nil
}

func (p *SubProcess) Interrupt() error {
	if p.relay {
		if p.pid <= 0 {
			return errutil.Wrap(ErrNotStartedYet)
		}
		p.log.Info("begin interrupt subprocess (relay)")
		if err := signalInterrupt(p.pid); err != nil {
			return errutil.Wrap(err)
		}
		return nil
	}
	if p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	p.log.Info("begin interrupt subprocess")
	if err := p.cmd.Process.Signal(os.Interrupt); err != nil {
		return errutil.Wrap(err)
	}
	return nil
}

func (p *SubProcess) Kill() error {
	p.log.Info("begin kill subprocess")
	if p.relay {
		if p.cancel != nil {
			p.cancel()
		}
		if p.pid > 0 {
			_ = signalKill(p.pid)
		}
		return nil
	}
	if p.cancel != nil {
		p.cancel()
	}
	return p.ctx.Err()
}

func (p *SubProcess) Write(content []byte) (n int, err error) {
	if p.stdin == nil {
		return 0, errutil.Wrap(ErrNotStartedYet)
	}
	return p.stdin.Write(content)
}

func (p *SubProcess) Wait() error {
	if p.ctx == nil {
		return errutil.Wrap(ErrNotStartedYet)
	}
	p.log.Info("begin wait subprocess")
	return <-p.waitChan
}

func (p *SubProcess) blockWait() {
	err := p.cmd.Wait()
	p.waitChan <- err
	p.cancel()
	p.log.Warn("subprocess stopped", "err", err)
}

func (p *SubProcess) blockWaitRelay() {
	err := p.cmd.Wait()
	p.closeRelay()
	p.waitChan <- err
	p.cancel()
	p.log.Warn("subprocess stopped", "err", err)
}

func (p *SubProcess) blockWaitAttached() {
	ticker := time.NewTicker(relayPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-p.ctx.Done():
			// ctx 被取消（例如强制杀进程）时也唤醒 Wait 调用方
			select {
			case p.waitChan <- context.Canceled:
			default:
			}
			p.closeRelay()
			return
		case <-ticker.C:
			if processAliveWithToken(p.pid, p.startToken) {
				continue
			}
			p.closeRelay()
			p.waitChan <- nil
			p.cancel()
			p.log.Warn("attached subprocess stopped")
			return
		}
	}
}

func (p *SubProcess) closeRelay() {
	if p.stdin != nil {
		_ = p.stdin.Close()
		p.stdin = nil
	}
	if p.fifoPath != "" {
		_ = os.Remove(p.fifoPath)
	}
}

const DefaultCommandTimeout = 2 * time.Second

// SendCommand writes a command to stdin and collects output for the given timeout.
// It filters out the command echo line (which DST console always echoes back).
func (p *SubProcess) SendCommand(cmd string, timeout time.Duration) (string, error) {
	if p.stdin == nil || p.ctx == nil {
		return "", errutil.Wrap(ErrNotStartedYet)
	}

	cmd = strings.TrimRight(cmd, "\n")

	var mu sync.Mutex
	var lines []string

	name := fmt.Sprintf("sendcmd_%d", time.Now().UnixNano())
	p.RegisterOutputFunc(name, func(line []byte) {
		mu.Lock()
		lines = append(lines, string(line))
		mu.Unlock()
	})

	_, err := p.Write([]byte(cmd + "\n"))
	if err != nil {
		p.UnregisterOutputFunc(name)
		return "", errutil.Wrap(err)
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-p.ctx.Done():
		p.UnregisterOutputFunc(name)
		return "", p.ctx.Err()
	}

	p.UnregisterOutputFunc(name)

	// Filter out the command echo line
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == cmd {
			continue
		}
		filtered = append(filtered, line)
	}

	return strings.Join(filtered, "\n"), nil
}

// loopOutput 用于普通管道模式：直接读取子进程 stdout/stderr。
func (p *SubProcess) loopOutput() {
	scanner := bufio.NewScanner(io.MultiReader(p.stdout, p.stderr))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		out := scanner.Bytes()
		select {
		case <-p.ctx.Done():
			return
		default:
		}
		p.dispatch(out)
	}
}

// startTail 从日志文件读取子进程输出；fromEnd 为 true 时从文件末尾开始（接管场景）。
func (p *SubProcess) startTail(fromEnd bool) {
	go func() {
		f, err := os.Open(p.logPath)
		if err != nil {
			p.log.Error("open shard log failed", "path", p.logPath, "err", err)
			return
		}
		defer f.Close()

		if fromEnd {
			if _, err := f.Seek(0, io.SeekEnd); err != nil {
				p.log.Error("seek shard log failed", "err", err)
				return
			}
		}

		var pending []byte
		buf := make([]byte, 32*1024)
		for {
			select {
			case <-p.ctx.Done():
				return
			default:
			}

			n, readErr := f.Read(buf)
			if n > 0 {
				pending = append(pending, buf[:n]...)
				for {
					idx := bytes.IndexByte(pending, '\n')
					if idx < 0 {
						break
					}
					line := append([]byte(nil), pending[:idx]...)
					pending = pending[idx+1:]
					p.dispatch(bytes.TrimRight(line, "\r"))
				}
				// 防御异常超长行
				if len(pending) > 1024*1024 {
					pending = pending[len(pending)-1024*1024:]
				}
			}

			if readErr != nil && !errors.Is(readErr, io.EOF) {
				p.log.Warn("read shard log failed", "err", readErr)
			}

			p.rotateShardLog(f)

			select {
			case <-p.ctx.Done():
				return
			case <-time.After(tailPollInterval):
			}
		}
	}()
}

// rotateShardLog 在日志超过上限时截断；子进程以 O_APPEND 写入，截断后会继续从 0 追加。
func (p *SubProcess) rotateShardLog(f *os.File) {
	stat, err := f.Stat()
	if err != nil || stat.Size() <= maxShardLogSize {
		return
	}
	if err := f.Truncate(0); err != nil {
		p.log.Warn("truncate shard log failed", "err", err)
		return
	}
	offset, err := f.Seek(0, io.SeekCurrent)
	if err == nil && offset > 0 {
		_, _ = f.Seek(0, io.SeekStart)
	}
}

func (p *SubProcess) dispatch(line []byte) {
	p.mu.RLock()
	funcs := make([]OutputFunc, 0, len(p.outFuncs))
	for _, f := range p.outFuncs {
		funcs = append(funcs, f)
	}
	p.mu.RUnlock()

	for _, f := range funcs {
		f(line)
	}
}

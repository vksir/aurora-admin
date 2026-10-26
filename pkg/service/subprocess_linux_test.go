//go:build linux

package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 模拟：面板启动游戏服 → 面板退出（只关闭自己的 FIFO 写端）→ 新面板重新接管控制台。
func TestRelayStartAndAttach(t *testing.T) {
	dir := t.TempDir()
	fifoPath := filepath.Join(dir, "in.fifo")
	logPath := filepath.Join(dir, "out.log")

	script := `while IFS= read -r line; do echo "got:$line"; done`
	p := NewSubprocess("test", "/bin/sh", []string{"-c", script}).SetRelay(fifoPath, logPath)
	require.NoError(t, p.Start(context.Background()))
	t.Cleanup(func() { _ = p.Kill() })

	out, err := p.SendCommand("hello", 2*time.Second)
	require.NoError(t, err)
	assert.Contains(t, out, "got:hello")

	// 模拟面板退出：关闭自己的 FIFO 写端，但不杀子进程
	pid := p.Pid()
	token := p.StartToken()
	require.NotZero(t, pid)
	require.NotEmpty(t, token)
	require.NoError(t, p.stdin.Close())

	// 新面板接管
	q := NewAttachedSubprocess("test", pid, token, fifoPath, logPath)
	require.NoError(t, q.Attach(context.Background()))
	t.Cleanup(func() { _ = q.Kill() })
	assert.True(t, q.Alive())

	out, err = q.SendCommand("world", 2*time.Second)
	require.NoError(t, err)
	assert.Contains(t, out, "got:world")

	// 停止：SIGINT 后进程应退出，FIFO 被清理
	require.NoError(t, q.Interrupt())
	select {
	case <-q.Ctx().Done():
	case <-time.After(5 * time.Second):
		t.Fatal("attached process did not stop in time")
	}
	assert.False(t, q.Alive())
	_, err = os.Stat(fifoPath)
	assert.True(t, os.IsNotExist(err), "fifo should be removed after stop")
}

func TestAttachRejectsDeadProcess(t *testing.T) {
	dir := t.TempDir()
	q := NewAttachedSubprocess("test", 1<<30, "dead-token",
		filepath.Join(dir, "in.fifo"), filepath.Join(dir, "out.log"))
	err := q.Attach(context.Background())
	assert.ErrorIs(t, err, ErrNotStartedYet)
}

package dontstarve

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"aurora-admin/pkg/log"
	"aurora-admin/pkg/service"

	"github.com/stretchr/testify/assert"
)

func TestIsShardActiveLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{
			name:     "模拟开始运行",
			line:     "[00:00:05]: Sim unpaused",
			expected: true,
		},
		{
			name:     "启动后无人连接直接暂停",
			line:     "[00:00:20]: Sim paused",
			expected: true,
		},
		{
			name:     "启动过程中的普通日志",
			line:     "[00:00:05]: LOADING LUA SUCCESS",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isShardActiveLine([]byte(tt.line)))
		})
	}
}

func TestScanShardLog(t *testing.T) {
	dir := t.TempDir()

	write := func(name, content string) string {
		path := filepath.Join(dir, name)
		assert.Nil(t, os.WriteFile(path, []byte(content), 0o600))
		return path
	}

	assert.Equal(t,
		shardLogActive,
		scanShardLog(write("active.log",
			"[00:00:00]: Starting Up\n"+
				"[00:00:01]: LOADING LUA SUCCESS\n"+
				"[00:00:05]: Sim unpaused\n")),
	)

	assert.Equal(t,
		shardLogFailed,
		scanShardLog(write("failed.log",
			"[00:00:00]: Starting Up\n"+
				"[00:00:50]: !!!! Your Server Will Not Start !!!!\n"+
				"[00:00:50]: No auth token could be found.\n")),
	)

	assert.Equal(t,
		shardLogUnknown,
		scanShardLog(write("starting.log", "[00:00:00]: Starting Up\n")),
	)

	assert.Equal(t, shardLogUnknown, scanShardLog(filepath.Join(dir, "not-exist.log")))
}

// WaitActive 需要补扫已写入日志的启动完成标记：注册输出监听前
// 服务端可能已经打印过 "Sim unpaused"，只监听后续输出会永远等不到。
func TestWaitActiveDetectsMarkerInExistingLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dst-master.log")
	assert.Nil(t, os.WriteFile(path, []byte(
		"[00:00:00]: Starting Up\n"+
			"[00:00:05]: Sim unpaused\n",
	), 0o600))

	pid := os.Getpid()
	token, _ := service.ProcessStartToken(pid)
	proc := service.NewAttachedSubprocess(ShardMaster, pid, token, "", path)

	p := &Plugin{logger: log.NewLogger("")}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	assert.True(t, p.WaitActive(ctx, map[string]*service.SubProcess{ShardMaster: proc}))
}

// 启动失败且进程未退出时（例如缺少 cluster token），应立即判定失败，
// 而不是一直停在 WaitingActive 直到超时。
func TestWaitActiveDetectsFailureInExistingLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dst-master.log")
	assert.Nil(t, os.WriteFile(path, []byte(
		"[00:00:00]: Starting Up\n"+
			"[00:00:50]: !!!! Your Server Will Not Start !!!!\n",
	), 0o600))

	pid := os.Getpid()
	token, _ := service.ProcessStartToken(pid)
	proc := service.NewAttachedSubprocess(ShardMaster, pid, token, "", path)

	p := &Plugin{logger: log.NewLogger("")}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	assert.False(t, p.WaitActive(ctx, map[string]*service.SubProcess{ShardMaster: proc}))
}

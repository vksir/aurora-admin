package cache

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimePersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")

	state := Init(path)
	rt := &Runtime{
		ArchID:    "arch-1",
		StartedAt: 123,
		Shards: map[string]RuntimeShard{
			"Master": {PID: 10, StartToken: "100", FIFO: "/run/master.fifo", Log: "/log/master.log"},
			"Caves":  {PID: 11, StartToken: "101", FIFO: "/run/caves.fifo", Log: "/log/caves.log"},
		},
	}
	state.SetRuntime(rt)

	reloaded := Init(path)
	got := reloaded.GetRuntime()
	require.NotNil(t, got)
	assert.Equal(t, rt, got)

	// 返回的应是副本，修改它不影响持久化状态
	got.Shards["Master"] = RuntimeShard{PID: 99}
	assert.Equal(t, 10, reloaded.GetRuntime().Shards["Master"].PID)

	reloaded.ClearRuntime()
	assert.Nil(t, Init(path).GetRuntime())
}

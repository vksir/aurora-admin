package dontstarve

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// collect 启动 StreamLogFile 并把结果送进 channel。
func collect(t *testing.T, path string, backfill int, generation func() int64) (<-chan LogFileItem, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	items := make(chan LogFileItem, 4096)
	go func() {
		defer close(items)
		StreamLogFile(ctx, path, backfill, generation, func(item LogFileItem) bool {
			select {
			case items <- item:
				return true
			case <-ctx.Done():
				return false
			}
		})
	}()
	t.Cleanup(cancel)
	return items, cancel
}

func next(t *testing.T, items <-chan LogFileItem) LogFileItem {
	t.Helper()
	select {
	case item, ok := <-items:
		require.True(t, ok, "stream closed")
		return item
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for log item")
		return LogFileItem{}
	}
}

func writeLog(t *testing.T, path, content string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640)
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
}

func TestStreamLogFileBackfillAndTail(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dst-master.log")

	var b strings.Builder
	for i := range 500 {
		fmt.Fprintf(&b, "line-%d\n", i)
	}
	require.NoError(t, os.WriteFile(path, []byte(b.String()), 0o640))

	items, _ := collect(t, path, 300, nil)

	// 回灌最近 300 行
	first := next(t, items)
	assert.False(t, first.Reset)
	assert.Equal(t, "line-200", first.Line)
	for i := 201; i < 500; i++ {
		assert.Equal(t, fmt.Sprintf("line-%d", i), next(t, items).Line)
	}

	// 追加内容被逐行读出
	writeLog(t, path, "appended-a\nappended-b\n")
	assert.Equal(t, "appended-a", next(t, items).Line)
	assert.Equal(t, "appended-b", next(t, items).Line)
}

func TestStreamLogFileBackfillZeroSkipsHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dst-master.log")
	require.NoError(t, os.WriteFile(path, []byte("old-1\nold-2\n"), 0o640))

	items, _ := collect(t, path, 0, nil)
	// 等流完成首次打开并定位到文件末尾后再追加
	time.Sleep(100 * time.Millisecond)
	writeLog(t, path, "new-1\n")

	assert.Equal(t, "new-1", next(t, items).Line)
}

func TestStreamLogFileWaitsForFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log", "dst-caves.log")

	items, _ := collect(t, path, 300, nil)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	writeLog(t, path, "created-1\n")

	assert.Equal(t, "created-1", next(t, items).Line)
}

func TestStreamLogFileTruncateEmitsReset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dst-master.log")
	require.NoError(t, os.WriteFile(path, []byte("before-1\nbefore-2\n"), 0o640))

	items, _ := collect(t, path, 300, nil)
	assert.Equal(t, "before-1", next(t, items).Line)
	assert.Equal(t, "before-2", next(t, items).Line)

	require.NoError(t, os.Truncate(path, 0))
	writeLog(t, path, "after-1\n")

	assert.True(t, next(t, items).Reset)
	assert.Equal(t, "after-1", next(t, items).Line)
}

func TestStreamLogFileGenerationChangeEmitsReset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dst-master.log")
	require.NoError(t, os.WriteFile(path, []byte("run-1\n"), 0o640))

	var gen int64
	items, _ := collect(t, path, 300, func() int64 { return gen })
	assert.Equal(t, "run-1", next(t, items).Line)

	// 新一轮启动：清空文件并让代际 +1，即使新内容长度超过旧偏移也要复位
	require.NoError(t, os.Truncate(path, 0))
	gen++
	writeLog(t, path, strings.Repeat("x", 4096)+"\n")

	assert.True(t, next(t, items).Reset)
	assert.Equal(t, strings.Repeat("x", 4096), next(t, items).Line)
}

func TestStreamLogFileStopsOnCancel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dst-master.log")

	items, cancel := collect(t, path, 300, nil)
	writeLog(t, path, "line-1\n")
	assert.Equal(t, "line-1", next(t, items).Line)

	cancel()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case _, ok := <-items:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("stream did not stop after cancel")
		}
	}
}

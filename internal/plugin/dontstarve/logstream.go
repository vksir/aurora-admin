package dontstarve

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"aurora-admin/pkg/util/fileutil"
)

const (
	// LogBackfillLines 是一次日志流连接建立时回灌的历史行数。
	LogBackfillLines = 300
	// logBackfillBytes 是回灌时最多读取的文件尾部字节数，避免整读大文件。
	logBackfillBytes = 256 << 10
	// logPollInterval 是日志文件追加内容的轮询间隔。
	logPollInterval = 200 * time.Millisecond
	// logPingInterval 是心跳间隔，避免反向代理或浏览器因长时间无数据断开连接。
	logPingInterval = 15 * time.Second
	// logMaxPendingBytes 是单行未收尾时的缓冲上限，防御异常超长行。
	logMaxPendingBytes = 1 << 20
)

// LogEventType 是服务日志流的事件类型。
type LogEventType string

const (
	// LogEventTypeMeta 在连接建立时发送一次，告知各分片日志文件是否存在。
	LogEventTypeMeta LogEventType = "meta"
	// LogEventTypeLine 是一行服务日志。
	LogEventTypeLine LogEventType = "line"
	// LogEventTypeReset 表示日志文件被截断或重建，前端应清空已有内容。
	LogEventTypeReset LogEventType = "reset"
	// LogEventTypePing 是心跳，HTTP 层写成 SSE 注释行。
	LogEventTypePing LogEventType = "ping"
)

// LogEvent 是服务日志流中的一条事件。
type LogEvent struct {
	Type   LogEventType    `json:"type"`
	Shard  string          `json:"shard,omitempty"`
	Line   string          `json:"line,omitempty"`
	Shards map[string]bool `json:"shards,omitempty"`
}

// LogFileItem 是日志文件读取的一项：一行内容，或一次「文件已重置」通知。
type LogFileItem struct {
	Line  string
	Reset bool
}

// StreamLog 把各分片的服务日志合并成一条按行输出的流，阻塞到 ctx 结束或 emit 返回错误。
func (p *Plugin) StreamLog(ctx context.Context, emit func(LogEvent) error) error {
	shards := []string{ShardMaster, ShardCaves}
	meta := make(map[string]bool, len(shards))
	for _, shard := range shards {
		meta[shard] = fileutil.Exist(shardLogPath(shard))
	}
	if err := emit(LogEvent{Type: LogEventTypeMeta, Shards: meta}); err != nil {
		return err
	}

	items := make(chan LogEvent, 1024)
	var wg sync.WaitGroup
	for _, shard := range shards {
		wg.Add(1)
		go func(shard string) {
			defer wg.Done()
			StreamLogFile(ctx, shardLogPath(shard), LogBackfillLines, p.logGeneration.Load, func(item LogFileItem) bool {
				ev := LogEvent{Type: LogEventTypeLine, Shard: shard, Line: item.Line}
				if item.Reset {
					ev = LogEvent{Type: LogEventTypeReset, Shard: shard}
				}
				select {
				case items <- ev:
					return true
				case <-ctx.Done():
					return false
				}
			})
		}(shard)
	}
	go func() {
		wg.Wait()
		close(items)
	}()

	ping := time.NewTicker(logPingInterval)
	defer ping.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-items:
			if !ok {
				return nil
			}
			if err := emit(ev); err != nil {
				return err
			}
		case <-ping.C:
			if err := emit(LogEvent{Type: LogEventTypePing}); err != nil {
				return err
			}
		}
	}
}

// StreamLogFile 从 path 持续读取日志并逐项回调：首次打开时回灌最近 backfill 行，
// 之后轮询追加内容。文件被截断（每次启动都会清空）或重建时先回调一次 Reset，
// 再从文件头继续读。文件不存在时轮询等待创建。
//
// generation 不为 nil 时，其返回值变化也表示开启了一轮新的运行（即便旧偏移
// 已被新一轮的输出重新覆盖），此时同样复位到文件头并回调 Reset。
//
// emit 返回 false 时停止读取。函数在 ctx 结束后返回。
func StreamLogFile(ctx context.Context, path string, backfill int,
	generation func() int64, emit func(LogFileItem) bool) {
	var (
		f          *os.File
		offset     int64
		pending    []byte
		backfilled bool
		opened     bool
		buf        = make([]byte, 32<<10)
	)
	var lastGen int64
	if generation != nil {
		lastGen = generation()
	}

	closeFile := func() {
		if f != nil {
			_ = f.Close()
			f = nil
		}
		pending = nil
		offset = 0
	}
	defer closeFile()

	for ctx.Err() == nil {
		if f == nil {
			file, err := os.Open(path)
			if err != nil {
				if !sleepCtx(ctx, logPollInterval) {
					return
				}
				continue
			}
			f = file
			pending = nil
			offset = 0
			if opened {
				// 文件被删除后重新出现：视为一轮新的运行
				if !emit(LogFileItem{Reset: true}) {
					return
				}
			}
			opened = true
			if !backfilled {
				backfilled = true
				lines, end := tailLines(f, backfill)
				for _, line := range lines {
					if !emit(LogFileItem{Line: line}) {
						return
					}
				}
				// tailLines 不保证文件位置已在末尾（backfill=0 时甚至没动过位置）
				if _, err := f.Seek(end, io.SeekStart); err != nil {
					closeFile()
					continue
				}
				offset = end
			}
		}

		// 新一轮启动：丢弃上一轮的内容
		if generation != nil && generation() != lastGen {
			lastGen = generation()
			if f != nil {
				if _, err := f.Seek(0, io.SeekStart); err != nil {
					closeFile()
					continue
				}
				hadContent := offset > 0 || len(pending) > 0
				offset = 0
				pending = nil
				// 没读过任何内容时无事可清，不发 reset（避免与截断检测重复）
				if hadContent && !emit(LogFileItem{Reset: true}) {
					return
				}
			}
		}

		// 日志文件被截断后，旧偏移已经没有意义，复位到文件头继续读
		if stat, err := f.Stat(); err == nil && stat.Size() < offset {
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				closeFile()
				continue
			}
			offset = 0
			pending = nil
			if !emit(LogFileItem{Reset: true}) {
				return
			}
		}

		n, err := f.Read(buf)
		if n > 0 {
			offset += int64(n)
			pending = append(pending, buf[:n]...)
			for {
				idx := bytes.IndexByte(pending, '\n')
				if idx < 0 {
					break
				}
				line := string(bytes.TrimRight(pending[:idx], "\r"))
				pending = pending[idx+1:]
				if !emit(LogFileItem{Line: line}) {
					return
				}
			}
			if len(pending) > logMaxPendingBytes {
				pending = pending[len(pending)-logMaxPendingBytes:]
			}
			continue
		}
		if err != nil && !errors.Is(err, io.EOF) {
			// 文件被删除或替换：重新打开
			closeFile()
		}

		if !sleepCtx(ctx, logPollInterval) {
			return
		}
	}
}

// tailLines 从文件尾部读取最多 backfill 行，返回这些行和文件末尾偏移。
// backfill 为 0 时不读历史，只返回当前偏移。
func tailLines(f *os.File, backfill int) ([]string, int64) {
	stat, err := f.Stat()
	if err != nil || stat.Size() == 0 {
		return nil, 0
	}
	size := stat.Size()
	if backfill <= 0 {
		return nil, size
	}

	start := size - logBackfillBytes
	if start < 0 {
		start = 0
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return nil, size
	}
	data, err := io.ReadAll(io.LimitReader(f, size-start))
	if err != nil && len(data) == 0 {
		return nil, size
	}
	if start > 0 {
		// 从中间开始读时首行可能是半行，丢弃
		if idx := bytes.IndexByte(data, '\n'); idx >= 0 {
			data = data[idx+1:]
		} else {
			data = nil
		}
	}

	lines := strings.Split(string(data), "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	if len(lines) > backfill {
		lines = lines[len(lines)-backfill:]
	}
	return lines, size
}

// sleepCtx 等待 d，ctx 结束后立即返回 false。
func sleepCtx(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

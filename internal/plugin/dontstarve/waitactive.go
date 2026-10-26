package dontstarve

import (
	"aurora-admin/pkg/service"
	"bufio"
	"bytes"
	"context"
	"os"
	"sync"
	"time"
)

// WaitActive 等待服务真正启动完成：看到分片就绪标记返回 true，
// 进程提前退出或出现明确的启动失败标记返回 false。
func (p *Plugin) WaitActive(waitActiveCtx context.Context, process map[string]*service.SubProcess) bool {
	// 以 Master 分片的就绪标记作为整个服务的启动完成信号；
	// 没有 Master（理论上不会出现）时退化为任意一个分片。
	shard := process[ShardMaster]
	if shard == nil {
		for _, proc := range process {
			shard = proc
			break
		}
	}
	if shard == nil {
		return false
	}

	ready := make(chan struct{})
	failed := make(chan struct{})
	var readyOnce, failedOnce sync.Once
	markReady := func() { readyOnce.Do(func() { close(ready) }) }
	markFailed := func() { failedOnce.Do(func() { close(failed) }) }

	const listener = "wait-active"
	shard.RegisterOutputFunc(listener, func(line []byte) {
		switch {
		case isShardFailedLine(line):
			markFailed()
		case isShardActiveLine(line):
			markReady()
		}
	})
	defer shard.UnregisterOutputFunc(listener)

	// relay 模式下输出先落到分片日志文件，注册监听前已输出的内容要补扫一遍，
	// 否则服务启动很快时可能错过就绪/失败标记。
	if path := shard.LogPath(); path != "" {
		switch scanShardLog(path) {
		case shardLogActive:
			markReady()
		case shardLogFailed:
			markFailed()
		}
	}

	timeout := time.NewTimer(waitActiveTimeout)
	defer timeout.Stop()
	ticker := time.NewTicker(waitActivePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-waitActiveCtx.Done():
			return false
		case <-ready:
			p.logger.Info("shard active", "shard", shard.Name())
			return true
		case <-failed:
			p.logger.Warn("shard reported startup failure", "shard", shard.Name())
			return false
		case <-ticker.C:
			if !shard.Alive() {
				p.logger.Warn("shard exited before active", "shard", shard.Name())
				return false
			}
		case <-timeout.C:
			// 就绪标记缺失但进程仍存活（例如日志格式变化）时，按已启动处理，
			// 避免服务永远停留在 WaitingActive。
			if shard.Alive() {
				p.logger.Warn("wait shard active timeout, process still alive, treat as active",
					"shard", shard.Name())
				return true
			}
			return false
		}
	}
}

// isShardActiveLine 判断一行分片输出是否表示世界已加载、服务可以接受连接。
// DST 在模拟开始运行时输出 "Sim unpaused"；若启动时无人连接直接进入暂停，
// 则会输出 "Sim paused"，同样说明启动已完成。
func isShardActiveLine(line []byte) bool {
	return bytes.Contains(line, []byte(shardActiveMarker)) ||
		bytes.Contains(line, []byte(shardPausedMarker))
}

// isShardFailedLine 判断一行分片输出是否表示服务已经无法完成启动，
// 这类错误下进程可能仍存活，不能靠 Alive 判断。
func isShardFailedLine(line []byte) bool {
	for _, marker := range shardFailureMarkers {
		if bytes.Contains(line, []byte(marker)) {
			return true
		}
	}
	return false
}

type shardLogStatus int

const (
	shardLogUnknown shardLogStatus = iota
	shardLogActive
	shardLogFailed
)

// scanShardLog 从分片日志文件中查找就绪/失败标记。
func scanShardLog(path string) shardLogStatus {
	f, err := os.Open(path)
	if err != nil {
		return shardLogUnknown
	}
	defer f.Close()

	status := shardLogUnknown
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if isShardFailedLine(line) {
			return shardLogFailed
		}
		if isShardActiveLine(line) {
			status = shardLogActive
		}
	}
	return status
}

package dontstarve

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"aurora-admin/pkg/cache"
	"aurora-admin/pkg/service"
)

// serverBinaryName 是 DST 服务端可执行文件名（不带平台后缀，用于 cmdline 匹配）。
const serverBinaryName = "dontstarve_dedicated_server_nullrenderer"

// detachedStopTimeout 孤儿进程优雅停止的最长等待时间，超时后强杀。
const detachedStopTimeout = 12 * time.Second

func readCmdline(pid int) []string {
	raw, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "cmdline"))
	if err != nil || len(raw) == 0 {
		return nil
	}
	return strings.Split(strings.TrimRight(string(raw), "\x00"), "\x00")
}

// findRunningServerPids 扫描系统中属于指定集群的 DST 服务端进程。
//
// 背景：管理面板重启后，内存中的服务状态会丢失。新版本会通过 FIFO +
// 运行态持久化自动接管；接管失败时（如旧版本遗留进程），仍需要能停止它们。
//
// 仅在 Linux 上生效（读取 /proc）；其他平台返回 nil，不影响本地开发。
func findRunningServerPids(clusterID string) []int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}

	var pids []int
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		if matchServerCmdline(readCmdline(pid), clusterID) {
			pids = append(pids, pid)
		}
	}
	return pids
}

// matchServerCmdline 判断进程命令行是否为指定集群的 DST 服务端。
// clusterID 为空时只匹配可执行文件。
func matchServerCmdline(args []string, clusterID string) bool {
	hasBinary := false
	hasCluster := false
	for _, arg := range args {
		// 第一个参数是可执行文件路径，可能带前缀路径
		if strings.Contains(filepath.Base(arg), serverBinaryName) {
			hasBinary = true
		}
		if clusterID != "" && arg == clusterID {
			hasCluster = true
		}
	}
	if !hasBinary {
		return false
	}
	return clusterID == "" || hasCluster
}

// matchShardCmdline 判断进程命令行是否为指定集群、指定分片的 DST 服务端。
func matchShardCmdline(args []string, clusterID, shard string) bool {
	if !matchServerCmdline(args, clusterID) {
		return false
	}
	for i, arg := range args {
		if arg == "-shard" && i+1 < len(args) {
			return args[i+1] == shard
		}
	}
	return shard == ""
}

// ensureNoRunningServer 在启动前确认没有同集群的服务端进程在跑。
func (p *Plugin) ensureNoRunningServer(clusterID string) error {
	pids := findRunningServerPids(clusterID)
	if len(pids) == 0 {
		return nil
	}
	return fmt.Errorf(
		"检测到 %d 个该集群的服务端进程已在运行（PID: %v），请先停止它们再启动，"+
			"否则会因端口占用产生重复进程",
		len(pids), pids,
	)
}

// DetachedRunning 实现 service.Detached：判断是否存在不归当前面板管理的服务端进程。
func (p *Plugin) DetachedRunning() bool {
	return len(findRunningServerPids(p.state.DontStarve.EnabledArchiveID)) > 0
}

// StopDetached 实现 service.Detached：向孤儿进程发送 SIGINT 等待存档退出，
// 超时后强杀。用于面板重启后接管失败、或旧版本遗留进程的兜底停止。
func (p *Plugin) StopDetached(ctx context.Context) error {
	archID := p.state.DontStarve.EnabledArchiveID
	pids := findRunningServerPids(archID)
	if len(pids) == 0 {
		p.state.ClearRuntime()
		return nil
	}

	p.logger.InfoC(ctx, "stop detached server processes", "arch", archID, "pids", pids)
	for _, pid := range pids {
		if err := service.SignalInterrupt(pid); err != nil {
			p.logger.WarnC(ctx, "send interrupt failed", "pid", pid, "err", err)
		}
	}

	deadline := time.Now().Add(detachedStopTimeout)
	for time.Now().Before(deadline) {
		alive := false
		for _, pid := range pids {
			if service.PidAlive(pid) {
				alive = true
				break
			}
		}
		if !alive {
			p.state.ClearRuntime()
			p.logger.InfoC(ctx, "detached server processes stopped", "pids", pids)
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}

	p.logger.WarnC(ctx, "detached server graceful stop timeout, force kill", "pids", pids)
	for _, pid := range pids {
		if service.PidAlive(pid) {
			_ = service.SignalKill(pid)
		}
	}
	p.state.ClearRuntime()
	return nil
}

// OnServiceStarted 实现 service.Lifecycle：启动成功后持久化 PID/FIFO/日志路径，
// 供面板重启后重新接管。
func (p *Plugin) OnServiceStarted(ctx context.Context, process map[string]*service.SubProcess) error {
	rt := &cache.Runtime{
		ArchID:    p.state.DontStarve.EnabledArchiveID,
		StartedAt: time.Now().Unix(),
		Shards:    make(map[string]cache.RuntimeShard, len(process)),
	}
	for name, proc := range process {
		if proc.Pid() == 0 {
			continue
		}
		rt.Shards[name] = cache.RuntimeShard{
			PID:        proc.Pid(),
			StartToken: proc.StartToken(),
			FIFO:       proc.FIFOPath(),
			Log:        proc.LogPath(),
		}
		if proc.StartToken() == "" {
			// 非 relay 模式（本地开发/不支持 FIFO 的平台）没有重连能力
			p.logger.WarnC(ctx, "subprocess has no start token, runtime state not adoptable", "shard", name)
		}
	}
	p.state.SetRuntime(rt)
	p.logger.InfoC(ctx, "runtime state persisted", "arch", rt.ArchID, "shards", len(rt.Shards))
	return nil
}

// OnServiceStopped 实现 service.Lifecycle：正常停止后清理运行态。
func (p *Plugin) OnServiceStopped(ctx context.Context) {
	p.state.ClearRuntime()
	p.logger.DebugC(ctx, "runtime state cleared")
}

// AdoptRuntime 在面板启动时接管仍在运行的服务端进程。
// 只有 PID、启动标识、命令行（集群 + 分片）全部匹配时才接管；
// 否则清理运行态，交由 DetachedRunning/StopDetached 兜底。
func (p *Plugin) AdoptRuntime(ctx context.Context) error {
	rt := p.state.GetRuntime()
	if rt == nil {
		return nil
	}
	if len(rt.Shards) == 0 {
		p.state.ClearRuntime()
		return nil
	}

	process := make(map[string]*service.SubProcess, len(rt.Shards))
	for name, shard := range rt.Shards {
		if !service.ProcessAliveWithToken(shard.PID, shard.StartToken) {
			p.logger.WarnC(ctx, "runtime shard process gone, skip adopt", "shard", name, "pid", shard.PID)
			p.state.ClearRuntime()
			return nil
		}
		if !matchShardCmdline(readCmdline(shard.PID), rt.ArchID, name) {
			p.logger.WarnC(ctx, "runtime shard cmdline mismatch, skip adopt", "shard", name, "pid", shard.PID)
			p.state.ClearRuntime()
			return nil
		}

		proc := service.NewAttachedSubprocess(name, shard.PID, shard.StartToken, shard.FIFO, shard.Log).
			SetLogger(p.logger)
		if err := proc.Attach(ctx); err != nil {
			p.logger.WarnC(ctx, "attach shard process failed", "shard", name, "pid", shard.PID, "err", err)
			p.state.ClearRuntime()
			return err
		}
		process[name] = proc
	}

	p.svc.Adopt(process)
	p.logger.InfoC(ctx, "adopted running server processes", "arch", rt.ArchID, "shards", len(process))
	return nil
}

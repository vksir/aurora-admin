package service

import "errors"

// ErrRelayUnsupported 表示当前平台不支持 FIFO 中继模式（仅 Linux 生产环境使用）。
var ErrRelayUnsupported = errors.New("relay mode (FIFO) is only supported on linux")

// SignalInterrupt 向进程发送 SIGINT（DST 收到后会存档退出）。
func SignalInterrupt(pid int) error { return signalInterrupt(pid) }

// SignalKill 强杀进程，仅在优雅停止超时后使用。
func SignalKill(pid int) error { return signalKill(pid) }

// PidAlive 判断进程是否仍存在（不校验 PID 复用）。
func PidAlive(pid int) bool { return pidAlive(pid) }

// ProcessAliveWithToken 结合 starttime 判断进程是否仍是最初那个，避免 PID 复用误判。
func ProcessAliveWithToken(pid int, token string) bool { return processAliveWithToken(pid, token) }

// ProcessStartToken 返回进程启动标识（Linux 为 /proc/<pid>/stat 的 starttime）。
func ProcessStartToken(pid int) (string, error) { return processStartToken(pid) }

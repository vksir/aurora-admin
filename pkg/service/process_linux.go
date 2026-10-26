//go:build linux

package service

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// makeFIFO 创建命名管道；已存在时先删除再创建。
func makeFIFO(path string) error {
	_ = os.Remove(path)
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		return fmt.Errorf("mkfifo %s: %w", path, err)
	}
	return nil
}

// openFIFOReadWrite 以 O_RDWR 打开 FIFO 并交给子进程当 stdin。
// 子进程自己持有写端，面板退出时不会让子进程的 stdin 读到 EOF，
// 控制台线程因此不会退出；面板重启后重新 openFIFOWrite 即可继续下发命令。
func openFIFOReadWrite(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open fifo rw %s: %w", path, err)
	}
	return f, nil
}

// openFIFOWrite 以只写方式打开 FIFO；进程已退出（无读者）时返回 ENXIO。
func openFIFOWrite(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("open fifo w %s: %w", path, err)
	}
	// 打开成功后恢复阻塞模式，避免控制台命令写入时遇到 EAGAIN
	if err := syscall.SetNonblock(int(f.Fd()), false); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("set fifo blocking %s: %w", path, err)
	}
	return f, nil
}

func signalInterrupt(pid int) error {
	return syscall.Kill(pid, syscall.SIGINT)
}

func signalKill(pid int) error {
	return syscall.Kill(pid, syscall.SIGKILL)
}

func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

// processStartToken 读取 /proc/<pid>/stat 的 starttime（字段 22），
// 用于在重连时区分 PID 复用。
func processStartToken(pid int) (string, error) {
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return "", err
	}
	// 格式：pid (comm) state ppid ...，comm 内可能包含空格和括号
	idx := strings.LastIndex(string(raw), ")")
	if idx < 0 {
		return "", fmt.Errorf("unexpected stat format for pid %d", pid)
	}
	fields := strings.Fields(string(raw)[idx+1:])
	// fields[0] 对应第 3 列 state，starttime 是第 22 列
	const startTimeOffset = 22 - 3
	if len(fields) <= startTimeOffset {
		return "", fmt.Errorf("unexpected stat fields for pid %d", pid)
	}
	return fields[startTimeOffset], nil
}

func processAliveWithToken(pid int, token string) bool {
	if !pidAlive(pid) {
		return false
	}
	if token == "" {
		return true
	}
	current, err := processStartToken(pid)
	return err == nil && current == token
}

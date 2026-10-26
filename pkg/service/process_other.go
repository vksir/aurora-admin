//go:build !linux

package service

import "os"

func makeFIFO(string) error { return ErrRelayUnsupported }

func openFIFOReadWrite(string) (*os.File, error) { return nil, ErrRelayUnsupported }

func openFIFOWrite(string) (*os.File, error) { return nil, ErrRelayUnsupported }

func signalInterrupt(int) error { return ErrRelayUnsupported }

func signalKill(int) error { return ErrRelayUnsupported }

func pidAlive(pid int) bool { return pid > 0 }

func processStartToken(int) (string, error) { return "", ErrRelayUnsupported }

func processAliveWithToken(pid int, _ string) bool { return pidAlive(pid) }

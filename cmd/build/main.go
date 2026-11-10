package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal, 1)
	done := make(chan struct{})
	var signalStatus atomic.Int32
	if runtime.GOOS == "linux" {
		signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(signals)
		go func() {
			select {
			case received := <-signals:
				signalStatus.Store(int32(128 + received.(syscall.Signal)))
				cancel()
			case <-done:
			}
		}()
	}
	var err error
	switch runtime.GOOS {
	case "linux":
		err = runDaemonless(ctx, os.Args[1:])
	case "darwin":
		err = runBuildx(os.Args[1:])
	default:
		err = fmt.Errorf("Unsupported build host: %s", runtime.GOOS)
	}
	close(done)
	status := 0
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		status = 1
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			status = exit.ExitCode()
		}
	}
	if errors.Is(err, context.Canceled) && signalStatus.Load() != 0 {
		status = int(signalStatus.Load())
	}
	cancel()
	os.Exit(status)
}

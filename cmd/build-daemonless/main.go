package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/ririnto/static-builds/internal/buildconfig"
	"github.com/ririnto/static-builds/internal/metadata"
)

type session struct {
	command *exec.Cmd
	done    chan struct{}
	err     error
}

func start(command *exec.Cmd) (*session, error) {
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := command.Start(); err != nil {
		return nil, err
	}
	s := &session{command: command, done: make(chan struct{})}
	go func() {
		s.err = command.Wait()
		close(s.done)
	}()
	return s, nil
}

func alive(pid int) bool {
	err := syscall.Kill(pid, 0)
	if err != nil && !errors.Is(err, syscall.ESRCH) && !errors.Is(err, syscall.EPERM) {
		fmt.Fprintln(os.Stderr, "Error: probe owned process:", err)
	}
	return err == nil || errors.Is(err, syscall.EPERM)
}

func send(pid int, signal syscall.Signal) {
	if err := syscall.Kill(pid, signal); err != nil && !errors.Is(err, syscall.ESRCH) {
		fmt.Fprintln(os.Stderr, "Error: signal owned process:", err)
	}
}

func (s *session) stop() error {
	if s == nil {
		return nil
	}
	pid := s.command.Process.Pid
	send(-pid, syscall.SIGTERM)
	send(pid, syscall.SIGTERM)
	deadline := time.Now().Add(5 * time.Second)
	for (alive(-pid) || alive(pid)) && time.Now().Before(deadline) {
		time.Sleep(time.Second)
	}
	send(-pid, syscall.SIGKILL)
	send(pid, syscall.SIGKILL)
	<-s.done
	var exit *exec.ExitError
	if s.err != nil && !errors.As(s.err, &exit) {
		fmt.Fprintln(os.Stderr, "Error: reap owned process:", s.err)
	}
	if alive(-pid) {
		return fmt.Errorf("Owned process group remains; retaining private state.")
	}
	return nil
}

func wait(ctx context.Context, s *session) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.done:
		return s.err
	}
}

func run(ctx context.Context, args []string) (err error) {
	if len(args) == 0 {
		return fmt.Errorf("usage: build-daemonless <target> [buildctl build arguments...]")
	}
	target := args[0]
	if target == "" || !buildconfig.ValidTarget(target) {
		return fmt.Errorf("Invalid target: %s", target)
	}
	config, err := buildconfig.Load(target)
	if err != nil {
		return err
	}
	if info, err := os.Stat(filepath.Join(config.Root, target, "Dockerfile")); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("Target Dockerfile not found: %s", target)
	}
	data, err := metadata.LoadDefault()
	if err != nil {
		return err
	}
	entry, err := data.Target(target)
	if err != nil {
		return err
	}
	environment, err := entry.Environment()
	if err != nil {
		return err
	}
	if os.Geteuid() == 0 {
		return fmt.Errorf("Native Linux builds require a non-root user.")
	}
	for _, prerequisite := range []string{"rootlesskit", "setsid", "timeout"} {
		path, err := exec.LookPath(prerequisite)
		if err != nil {
			return err
		}
		fmt.Println(path)
	}
	oci, err := exec.LookPath("runc")
	if errors.Is(err, exec.ErrNotFound) {
		oci, err = exec.LookPath("crun")
	}
	if err != nil {
		return fmt.Errorf("Install runc or crun for the rootless OCI worker: %w", err)
	}
	fmt.Println(oci)
	oci = filepath.Base(oci)
	resolve := exec.Command("go", "tool", "-n", "buildkitd")
	resolve.Stderr = os.Stderr
	output, err := resolve.Output()
	if err != nil {
		return err
	}
	buildkitd := strings.TrimRight(string(output), "\n")
	if info, err := os.Stat(buildkitd); err != nil || !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
		return fmt.Errorf("go tool -n buildkitd must resolve one executable file.")
	}
	version := exec.Command("go", "tool", "buildctl", "--version")
	version.Stdout, version.Stderr = os.Stdout, os.Stderr
	if err := version.Run(); err != nil {
		return err
	}
	cacheFrom, cacheTo, err := config.Caches()
	if err != nil {
		return err
	}
	network, err := config.FrontendNetwork()
	if err != nil {
		return err
	}
	arguments := []string{"build", "--frontend", "dockerfile.v0", "--local", "context=" + config.Root, "--local", "dockerfile=" + config.Root, "--opt", "filename=" + target + "/Dockerfile", "--opt", "platform=" + config.Platform, "--opt", "force-network-mode=" + network, "--output", "type=local,dest=" + config.Output, "--import-cache", cacheFrom, "--export-cache", cacheTo}
	buildArgs := buildconfig.Arguments(environment)
	for index := len(buildArgs) - 1; index >= 0; index-- {
		arguments = append(arguments, "--opt", "build-arg:"+buildArgs[index])
	}
	arguments = append(arguments, args[1:]...)
	if config.Network == "host" {
		arguments = append(arguments, "--allow", "network.host")
	}
	temporary, err := os.MkdirTemp("", "static-buildkit.")
	if err != nil {
		return err
	}
	var client, daemon *session
	var reader *os.File
	var loggerDone chan error
	defer func() {
		namespace := 0
		if contents, readErr := os.ReadFile(filepath.Join(temporary, "rootlesskit", "child_pid")); readErr == nil {
			namespace, _ = strconv.Atoi(strings.TrimSpace(string(contents)))
		}
		cleanupErr := client.stop()
		if stopErr := daemon.stop(); stopErr != nil {
			cleanupErr = stopErr
		}
		if loggerDone != nil {
			if cleanupErr != nil && reader != nil {
				if closeErr := reader.Close(); closeErr != nil && !errors.Is(closeErr, os.ErrClosed) {
					fmt.Fprintln(os.Stderr, closeErr)
				}
			}
			if logErr := <-loggerDone; logErr != nil {
				fmt.Fprintln(os.Stderr, "Error: drain daemon diagnostics:", logErr)
			}
		}
		if reader != nil {
			if closeErr := reader.Close(); closeErr != nil && !errors.Is(closeErr, os.ErrClosed) {
				fmt.Fprintln(os.Stderr, closeErr)
			}
		}
		if namespace > 0 {
			deadline := time.Now().Add(5 * time.Second)
			for alive(namespace) && time.Now().Before(deadline) {
				time.Sleep(time.Second)
			}
			if alive(namespace) {
				cleanupErr = fmt.Errorf("Owned PID namespace remains; retaining private state.")
			}
		}
		if cleanupErr != nil {
			err = cleanupErr
		} else {
			if err != nil {
				fmt.Fprintln(os.Stderr, "Rootless BuildKit failed; final daemon diagnostics:")
				if diagnostics, readErr := os.ReadFile(filepath.Join(temporary, "daemon.log")); readErr == nil {
					if len(diagnostics) > 8192 {
						diagnostics = diagnostics[len(diagnostics)-8192:]
					}
					os.Stderr.Write(diagnostics)
				}
			}
			if removeErr := os.RemoveAll(temporary); removeErr != nil {
				err = removeErr
			}
		}
	}()
	if config.CacheBackend == "local" {
		if err := os.MkdirAll(config.Cache, 0755); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(config.Output, 0755); err != nil {
		return err
	}
	address := "--addr=unix://" + filepath.Join(temporary, "buildkit.sock")
	reader, writer, err := os.Pipe()
	if err != nil {
		return err
	}
	log, err := os.OpenFile(filepath.Join(temporary, "daemon.log"), os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		writer.Close()
		return err
	}
	loggerDone = make(chan error, 1)
	go func() {
		buffer := make([]byte, 4096)
		remaining := 65536
		var logErr error
		for {
			count, readErr := reader.Read(buffer)
			if count > 0 && remaining > 0 && logErr == nil {
				written, writeErr := log.Write(buffer[:min(count, remaining)])
				remaining -= written
				logErr = writeErr
			}
			if readErr != nil {
				if readErr != io.EOF && !errors.Is(readErr, os.ErrClosed) && logErr == nil {
					logErr = readErr
				}
				break
			}
		}
		if closeErr := log.Close(); logErr == nil {
			logErr = closeErr
		}
		loggerDone <- logErr
	}()
	daemonArguments := []string{"--pidns", "--reaper=true", "--state-dir=" + filepath.Join(temporary, "rootlesskit"), buildkitd, address, "--root=" + filepath.Join(temporary, "state"), "--containerd-worker=false", "--oci-worker=true", "--oci-worker-binary=" + oci}
	if config.Network == "host" {
		daemonArguments = append(daemonArguments, "--allow-insecure-entitlement", "network.host")
	}
	command := exec.Command("rootlesskit", daemonArguments...)
	command.Stdout, command.Stderr = writer, writer
	daemon, err = start(command)
	writer.Close()
	if err != nil {
		return err
	}
	deadline := time.Now().Add(60 * time.Second)
	ready := false
	for attempt := 0; !ready && attempt < 30 && time.Now().Before(deadline) && alive(daemon.command.Process.Pid) && ctx.Err() == nil; attempt++ {
		var workers bytes.Buffer
		probe := exec.Command("timeout", "--kill-after=1", "1", "go", "tool", "buildctl", address, "debug", "workers", "--format", "{{range .}}{{.ID}}{{end}}")
		probe.Stdout, probe.Stderr = &workers, &workers
		client, err = start(probe)
		if err != nil {
			return err
		}
		if wait(ctx, client) == nil && len(bytes.TrimSpace(workers.Bytes())) > 0 {
			ready = true
		} else if ctx.Err() == nil {
			timer := time.NewTimer(time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
			case <-timer.C:
			}
		}
		if ctx.Err() == nil {
			client = nil
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if !ready {
		return fmt.Errorf("Rootless BuildKit worker readiness deadline expired.")
	}
	command = exec.Command("go", append([]string{"tool", "buildctl", address}, arguments...)...)
	command.Dir = config.Root
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	client, err = start(command)
	if err != nil {
		return err
	}
	return wait(ctx, client)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	done := make(chan struct{})
	var signalStatus atomic.Int32
	go func() {
		select {
		case received := <-signals:
			signalStatus.Store(int32(128 + received.(syscall.Signal)))
			cancel()
		case <-done:
		}
	}()
	err := run(ctx, os.Args[1:])
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

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func execute(ctx context.Context, name string, arguments ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	command := exec.Command(name, arguments...)
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		_ = syscall.Kill(-command.Process.Pid, syscall.SIGTERM)
		timer := time.NewTimer(time.Second)
		defer timer.Stop()
		select {
		case <-done:
			// The leader may have exited while descendants still share its group.
			_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		case <-timer.C:
			_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
			<-done
		}
		return ctx.Err()
	}
}

type interrupted struct{ signal os.Signal }

func (err interrupted) Error() string { return "interrupted by " + err.signal.String() }

func run(ctx context.Context) error {
	response, err := os.CreateTemp("/tmp", "healthcheck-startup.")
	if err != nil {
		return err
	}
	defer os.Remove(response.Name())
	if err := response.Close(); err != nil {
		return err
	}
	defer func() {
		for _, path := range []string{"/tmp/healthcheck-build-ca.key", "/runtime/conf/backend-ca.pem", "/runtime/etc/hosts"} {
			_ = os.Remove(path)
		}
	}()
	if err := execute(ctx, "chroot", "/runtime", "/sbin/nginx", "-p", "/", "-t"); err != nil {
		return err
	}
	if err := execute(ctx, "openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes", "-days", "1", "-subj", "/CN=healthcheck-build-ca", "-keyout", "/tmp/healthcheck-build-ca.key", "-out", "/runtime/conf/backend-ca.pem"); err != nil {
		return err
	}
	if err := os.WriteFile("/runtime/etc/hosts", []byte("127.0.0.1 maintenance.internal\n"), 0644); err != nil {
		return err
	}
	if err := execute(ctx, "chroot", "/runtime", "/sbin/nginx", "-p", "/", "-t", "-c", "conf/upstream-healthcheck.conf"); err != nil {
		return err
	}
	for _, path := range []string{"/tmp/healthcheck-build-ca.key", "/runtime/conf/backend-ca.pem", "/runtime/etc/hosts"} {
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	server := exec.Command("timeout", "--signal=QUIT", "--kill-after=2s", "20s", "chroot", "/runtime", "/sbin/nginx", "-p", "/", "-g", "daemon off;")
	server.Stdout, server.Stderr = os.Stdout, os.Stderr
	if err := server.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- server.Wait() }()
	defer func() {
		if server != nil {
			_ = server.Process.Signal(syscall.SIGQUIT)
			<-done
		}
	}()
	probe := exec.Command("timeout", "--signal=TERM", "--kill-after=1s", "15s", "curl", "--retry", "10", "--retry-connrefused", "--retry-delay", "1", "--retry-max-time", "10", "--max-time", "2", "-fsS", "--output", response.Name(), "http://127.0.0.1/")
	probe.Stdout, probe.Stderr = os.Stdout, os.Stderr
	if err := probe.Start(); err != nil {
		return err
	}
	probeDone := make(chan error, 1)
	go func() { probeDone <- probe.Wait() }()
	select {
	case <-ctx.Done():
		_ = probe.Process.Signal(syscall.SIGTERM)
		<-probeDone
		return ctx.Err()
	case err := <-probeDone:
		if err != nil {
			return err
		}
	}
	pid, err := os.ReadFile("/runtime/logs/nginx.pid")
	if err != nil {
		return err
	}
	number, err := strconv.Atoi(strings.TrimSpace(string(pid)))
	if err != nil {
		return err
	}
	if err := syscall.Kill(number, 0); err != nil {
		return err
	}
	if err := server.Process.Signal(syscall.SIGQUIT); err != nil {
		return err
	}
	err = <-done
	server = nil
	if err != nil {
		return err
	}
	for _, path := range []string{"/runtime/logs/access.log", "/runtime/logs/error.log"} {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func main() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	ctx, cancel := context.WithCancelCause(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case received := <-signals:
			cancel(interrupted{received})
		case <-ctx.Done():
		}
	}()
	err := run(ctx)
	cancel(nil)
	<-done
	if stopped, ok := context.Cause(ctx).(interrupted); ok {
		fmt.Fprintln(os.Stderr, "Error:", stopped)
		os.Exit(128 + int(stopped.signal.(syscall.Signal)))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		if status, ok := err.(*exec.ExitError); ok {
			if status.ExitCode() < 0 {
				os.Exit(128 + int(status.Sys().(syscall.WaitStatus).Signal()))
			}
			os.Exit(status.ExitCode())
		}
		os.Exit(1)
	}
}

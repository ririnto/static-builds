package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type interrupted struct{ signal os.Signal }

func (err interrupted) Error() string { return "interrupted by " + err.signal.String() }

func verifyRuntime(ctx context.Context, binary string) error {
	command := exec.Command(binary, "-t", "-c", "conf/scripting.conf")
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := execute(ctx, command); err != nil {
		return err
	}
	response, err := os.CreateTemp("/tmp", "njs-result.")
	if err != nil {
		return err
	}
	defer os.Remove(response.Name())
	if err := response.Close(); err != nil {
		return err
	}
	server := exec.Command("timeout", "--signal=QUIT", "--kill-after=2s", "40s", binary, "-c", "conf/scripting.conf", "-g", "daemon off;")
	server.Stdout, server.Stderr = os.Stdout, os.Stderr
	if err := server.Start(); err != nil {
		return err
	}
	serverDone := make(chan error, 1)
	go func() { serverDone <- server.Wait() }()
	defer func() {
		_ = server.Process.Signal(syscall.SIGQUIT)
		<-serverDone
	}()
	for _, arguments := range [][]string{
		{"--retry", "10", "--retry-connrefused", "--retry-delay", "1", "--retry-max-time", "10", "--max-time", "2", "-fsS", "-o", response.Name(), "http://127.0.0.1:18081/njs"},
		{"--max-time", "5", "-fsS", "-o", response.Name(), "http://127.0.0.1:18081/njs"},
	} {
		probe := exec.Command("timeout", append([]string{"--signal=TERM", "--kill-after=1s", "15s", "curl"}, arguments...)...)
		probe.Stdout, probe.Stderr = os.Stdout, os.Stderr
		if err := probe.Start(); err != nil {
			return err
		}
		done := make(chan error, 1)
		go func() { done <- probe.Wait() }()
		select {
		case <-ctx.Done():
			_ = probe.Process.Signal(syscall.SIGTERM)
			<-done
			return ctx.Err()
		case err := <-done:
			if err != nil {
				return err
			}
		}
		contents, err := os.ReadFile(response.Name())
		if err != nil {
			return err
		}
		matched := false
		for _, line := range strings.Split(string(contents), "\n") {
			if line == "njs "+os.Getenv("NJS_VERSION")+" engine QuickJS" {
				fmt.Println(line)
				matched = true
			}
		}
		if !matched {
			return fmt.Errorf("unexpected njs HTTP response: %s", contents)
		}
	}
	// The former stream reader had one three-second deadline for connection and EOF.
	deadline := time.Now().Add(3 * time.Second)
	connection, err := (&net.Dialer{Deadline: deadline}).Dial("tcp", "127.0.0.1:18082")
	if err != nil {
		return err
	}
	defer connection.Close()
	if err := connection.SetDeadline(deadline); err != nil {
		return err
	}
	streamDone := make(chan error, 1)
	go func() {
		matched := false
		scanner := bufio.NewScanner(connection)
		for scanner.Scan() {
			if scanner.Text() == "njs "+os.Getenv("NJS_VERSION")+" engine QuickJS" {
				fmt.Println(scanner.Text())
				matched = true
			}
		}
		err := scanner.Err()
		if err == nil && !matched {
			err = fmt.Errorf("unexpected njs stream response")
		}
		streamDone <- err
	}()
	select {
	case <-ctx.Done():
		_ = connection.Close()
		<-streamDone
		return ctx.Err()
	case err := <-streamDone:
		return err
	}
}

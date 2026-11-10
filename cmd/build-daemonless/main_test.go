package main

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"
)

func TestSessionFailureAndReap(t *testing.T) {
	command := exec.Command("sh", "-c", "exit 37")
	owned, err := start(command)
	if err != nil {
		t.Fatal(err)
	}
	err = wait(context.Background(), owned)
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 37 {
		t.Fatalf("unexpected build status: %v", err)
	}
	if err := owned.stop(); err != nil {
		t.Fatal(err)
	}
	if alive(command.Process.Pid) || alive(-command.Process.Pid) {
		t.Fatal("owned process survived reap")
	}
}

func TestSessionCancellation(t *testing.T) {
	owned, err := start(exec.Command("sleep", "60"))
	if err != nil {
		t.Fatal(err)
	}
	defer owned.stop()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := wait(ctx, owned); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	if err := owned.stop(); err != nil {
		t.Fatal(err)
	}
	if alive(owned.command.Process.Pid) || alive(-owned.command.Process.Pid) {
		t.Fatal("cancelled process survived cleanup")
	}
}

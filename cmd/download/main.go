package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ririnto/static-builds/internal/metadata"
)

func setting(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func pause(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func lookPath(name string) (string, error) {
	path, err := exec.LookPath(name)
	if errors.Is(err, exec.ErrNotFound) {
		return "", nil
	}
	return path, err
}

func download(ctx context.Context, url, destination string) (err error) {
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Downloading %s -> %s\n", url, destination)
	if strings.Contains(url, "/archive/refs/heads/") {
		lock := destination + ".lock"
		for attempts := 0; ; attempts++ {
			err := os.Mkdir(lock, 0755)
			if err == nil {
				break
			}
			if !os.IsExist(err) {
				return err
			}
			if attempts >= 60 {
				return fmt.Errorf("Timed out waiting for branch download lock: %s", lock)
			}
			if err := pause(ctx, time.Second); err != nil {
				return err
			}
		}
		defer func() {
			if cleanupErr := os.Remove(lock); cleanupErr != nil {
				fmt.Fprintln(os.Stderr, cleanupErr)
				if err == nil {
					err = cleanupErr
				}
			}
		}()
	} else if info, err := os.Stat(destination); err == nil && info.Mode().IsRegular() && info.Size() > 0 {
		fmt.Fprintln(os.Stderr, "File exists:", destination)
		return nil
	}
	if !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("Unsupported URL scheme (https only): %s", url)
	}
	temporary, err := os.CreateTemp(filepath.Dir(destination), filepath.Base(destination)+".tmp.")
	if err != nil {
		return fmt.Errorf("Failed to create temporary file in %s: %w", filepath.Dir(destination), err)
	}
	path := temporary.Name()
	if err := temporary.Close(); err != nil {
		return err
	}
	defer func() {
		if cleanupErr := os.Remove(path); cleanupErr != nil && !os.IsNotExist(cleanupErr) {
			fmt.Fprintln(os.Stderr, cleanupErr)
		}
	}()
	retries, err := strconv.Atoi(setting("DOWNLOAD_RETRIES", "5"))
	if err != nil {
		return err
	}
	baseDelay, err := strconv.Atoi(setting("DOWNLOAD_BASE_DELAY", "1"))
	if err != nil {
		return err
	}
	connectTimeout := setting("DOWNLOAD_CONNECT_TIMEOUT", "15")
	maxTime := setting("DOWNLOAD_MAX_TIME", "300")
	for attempt := 1; attempt <= retries; attempt++ {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		curl, err := lookPath("curl")
		if err != nil {
			return err
		}
		var command *exec.Cmd
		if curl != "" {
			command = exec.CommandContext(ctx, curl, "--fail", "--location", "--silent", "--show-error", "--connect-timeout", connectTimeout, "--max-time", maxTime, "--output", path, url)
		} else {
			limiter, err := lookPath("timeout")
			if err != nil {
				return err
			}
			if limiter == "" {
				limiter, err = lookPath("gtimeout")
				if err != nil {
					return err
				}
			}
			if limiter == "" {
				return fmt.Errorf("wget downloads require timeout (GNU Coreutils or BusyBox) or gtimeout")
			}
			command = exec.CommandContext(ctx, limiter, "-k", "1", maxTime, "wget", "-q", "-T", connectTimeout, "-O", path, url)
		}
		command.Stdout, command.Stderr = os.Stdout, os.Stderr
		command.Cancel = func() error {
			err := command.Process.Signal(syscall.SIGTERM)
			if err != nil && !errors.Is(err, os.ErrProcessDone) && !errors.Is(err, syscall.ESRCH) {
				fmt.Fprintln(os.Stderr, "Error: cancel download:", err)
			}
			return err
		}
		command.WaitDelay = 2 * time.Second
		err = command.Run()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err == nil {
			info, err := os.Stat(path)
			if err != nil || info.Size() == 0 {
				return fmt.Errorf("Downloaded file is empty: %s (from %s)", destination, url)
			}
			return os.Rename(path, destination)
		}
		status := 1
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			status = exit.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "Error: Download failed (attempt %d/%d, exit=%d): %s -> %s\n", attempt, retries, status, url, destination)
		if attempt < retries {
			delay := baseDelay
			for exponent := 1; exponent < attempt && delay < 60; exponent++ {
				delay *= 2
			}
			if delay > 60 {
				delay = 60
			}
			fmt.Fprintf(os.Stderr, "Retrying in %d seconds...\n", delay)
			if err := pause(ctx, time.Duration(delay)*time.Second); err != nil {
				return err
			}
		}
	}
	return fmt.Errorf("download retries exhausted")
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: download <target>")
	}
	if strings.ContainsAny(args[0], "/\\") || strings.IndexFunc(args[0], func(c rune) bool {
		return !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.ContainsRune("._-", c))
	}) >= 0 {
		return fmt.Errorf("Invalid target '%s'", args[0])
	}
	data, err := metadata.LoadDefault()
	if err != nil {
		return err
	}
	target, err := data.Target(args[0])
	if err != nil {
		return err
	}
	downloads, err := target.RenderDownloads()
	if err != nil {
		return err
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	for _, record := range downloads {
		if record.URL != "" && record.Name != "" {
			if err := download(ctx, record.URL, filepath.Join(root, ".tmp", record.Name)); err != nil {
				return err
			}
		}
	}
	return nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

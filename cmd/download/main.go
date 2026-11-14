// Command download retrieves the sources required by one build target.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/alecthomas/kong"

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
	lock, err := os.OpenFile(destination+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	// Keep the lock file: removing it lets waiters lock a different inode.
	defer func() {
		if closeErr := lock.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()
	for attempts := 0; ; attempts++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) {
			return err
		}
		if attempts >= 60 {
			return fmt.Errorf("timed out waiting for download lock: %s", lock.Name())
		}
		if err := pause(ctx, time.Second); err != nil {
			return err
		}
	}
	// Only this archive's lock owner may remove interrupted download files.
	entries, err := os.ReadDir(filepath.Dir(destination))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), filepath.Base(destination)+".tmp.") {
			if err := os.Remove(filepath.Join(filepath.Dir(destination), entry.Name())); err != nil {
				return err
			}
		}
	}
	if !strings.Contains(url, "/archive/refs/heads/") {
		if info, err := os.Stat(destination); err == nil && info.Mode().IsRegular() && info.Size() > 0 {
			fmt.Fprintln(os.Stderr, "File exists:", destination)
			return nil
		}
	}
	if !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("unsupported URL scheme (https only): %s", url)
	}
	temporary, err := os.CreateTemp(filepath.Dir(destination), filepath.Base(destination)+".tmp.")
	if err != nil {
		return fmt.Errorf("failed to create temporary file in %s: %w", filepath.Dir(destination), err)
	}
	path := temporary.Name()
	defer func() {
		if cleanupErr := os.Remove(path); cleanupErr != nil && !os.IsNotExist(cleanupErr) {
			err = errors.Join(err, cleanupErr)
		}
	}()
	if err := temporary.Close(); err != nil {
		return err
	}
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
		// Keep ownership while an orphaned transport may still write after SIGKILL.
		command.ExtraFiles = []*os.File{lock}
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
				return fmt.Errorf("downloaded file is empty: %s (from %s)", destination, url)
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
	var cli struct {
		Target string `arg:"" help:"Target whose sources should be downloaded."`
	}
	parser, err := kong.New(&cli, kong.Name("download"))
	if err != nil {
		return err
	}
	if _, err := parser.Parse(args); err != nil {
		return err
	}
	if strings.ContainsAny(cli.Target, "/\\") || strings.IndexFunc(cli.Target, func(c rune) bool {
		return (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9') && !strings.ContainsRune("._-", c)
	}) >= 0 {
		return fmt.Errorf("invalid target '%s'", cli.Target)
	}
	data, err := metadata.LoadDefault()
	if err != nil {
		return err
	}
	target, err := data.Target(cli.Target)
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
	for record := range slices.Values(downloads) {
		if record.URL != "" && record.Name != "" {
			if err := download(ctx, record.URL, filepath.Join(root, ".cache", "downloads", cli.Target, record.Name)); err != nil {
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

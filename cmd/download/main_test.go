package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestDownloadProcess(t *testing.T) {
	if os.Getenv("DOWNLOAD_TEST_PROCESS") == "1" {
		os.Args = []string{"download", "fixture"}
		main()
		os.Exit(0)
	}
}

func downloadFixture(t *testing.T, branch bool) string {
	t.Helper()
	root := t.TempDir()
	url := "https://example.invalid/source.tar.gz"
	if branch {
		url = "https://example.invalid/archive/refs/heads/main.tar.gz"
	}
	if err := os.WriteFile(filepath.Join(root, "metadata.json"), []byte(fmt.Sprintf(`{"fixture":{"env":{"VERSION":"1"},"downloads":[{"url":%q,"name":"source.tar.gz"}]}}`, url)), 0600); err != nil {
		t.Fatal(err)
	}
	// Substitute only the external transport; exercise main, metadata, locking and publication.
	if err := os.WriteFile(filepath.Join(root, "curl"), []byte(`#!/bin/sh
set -eu
while [ "$1" != --output ]; do shift; done
shift
output=$1
printf 'request\n' >> requests
printf 'partial' > "$output"
if [ "${DOWNLOAD_TEST_HOLD:-}" = 1 ]; then
    : > started
    while [ ! -f proceed ]; do /bin/sleep 0.05; done
fi
case "${DOWNLOAD_TEST_RESULT:-success}" in
    failure) exit 22 ;;
    empty) : > "$output" ;;
    *) printf '%s' "${DOWNLOAD_TEST_CONTENT:-complete source}" > "$output" ;;
esac
`), 0700); err != nil {
		t.Fatal(err)
	}
	return root
}

func downloadCommand(t *testing.T, root string, extra ...string) (*exec.Cmd, *bytes.Buffer) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(executable, "-test.run=^TestDownloadProcess$")
	command.Dir = root
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Env = append(os.Environ(), "DOWNLOAD_TEST_PROCESS=1", "METADATA_PATH_OVERRIDE="+filepath.Join(root, "metadata.json"), "PATH="+root, "DOWNLOAD_RETRIES=1")
	command.Env = append(command.Env, extra...)
	output := new(bytes.Buffer)
	command.Stdout, command.Stderr = output, output
	return command, output
}

func assertDownload(t *testing.T, root, want string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".cache", "downloads", "source.tar.gz"))
	if err != nil || string(data) != want {
		t.Fatalf("archive = %q, %v; want %q", data, err, want)
	}
	partials, err := filepath.Glob(filepath.Join(root, ".cache", "downloads", "*.tmp.*"))
	if err != nil || len(partials) != 0 {
		t.Fatalf("incomplete downloads remain: %v, %v", partials, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".tmp")); !os.IsNotExist(err) {
		t.Fatalf("legacy temporary directory exists: %v", err)
	}
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}

func startDownload(t *testing.T, command *exec.Cmd) {
	t.Helper()
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			t.Error(err)
		}
		if command.ProcessState == nil {
			_ = command.Wait()
		}
	})
}

func downloadLog(t *testing.T, command *exec.Cmd) string {
	t.Helper()
	log, err := os.CreateTemp(command.Dir, "download-log-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := log.Close(); err != nil {
			t.Error(err)
		}
	})
	command.Stdout, command.Stderr = log, log
	return log.Name()
}

func assertContendingDownload(t *testing.T, root string, command *exec.Cmd) {
	t.Helper()
	partials, err := filepath.Glob(filepath.Join(root, ".cache", "downloads", "*.tmp.*"))
	if err != nil || len(partials) != 1 {
		t.Fatalf("expected one active partial: %v, %v", partials, err)
	}
	original, err := os.Stat(partials[0])
	if err != nil {
		t.Fatal(err)
	}
	lock, err := os.OpenFile(filepath.Join(root, ".cache", "downloads", "source.tar.gz.lock"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := lock.Close(); err != nil {
			t.Error(err)
		}
	}()
	log := downloadLog(t, command)
	startDownload(t, command)
	deadline := time.Now().Add(10 * time.Second)
	for {
		data, err := os.ReadFile(log)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "Downloading ") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("replacement did not reach download: %s", data)
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Observe the blocked replacement across one full production lock-poll interval.
	deadline = time.Now().Add(1100 * time.Millisecond)
	for time.Now().Before(deadline) {
		if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); !errors.Is(err, syscall.EWOULDBLOCK) {
			t.Fatalf("active transport lost exclusive ownership: %v", err)
		}
		data, err := os.ReadFile(filepath.Join(root, "requests"))
		if err != nil || string(data) != "request\n" {
			t.Fatalf("replacement started a transport before release: %q, %v", data, err)
		}
		info, err := os.Stat(partials[0])
		if err != nil || !os.SameFile(original, info) {
			t.Fatalf("replacement removed or replaced the active partial: %v", err)
		}
		data, err = os.ReadFile(partials[0])
		if err != nil || string(data) != "partial" {
			t.Fatalf("replacement modified the active partial: %q, %v", data, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestDownloadRerun(t *testing.T) {
	for _, branch := range []bool{false, true} {
		t.Run(fmt.Sprintf("branch=%t", branch), func(t *testing.T) {
			root := downloadFixture(t, branch)
			command, output := downloadCommand(t, root)
			if err := command.Run(); err != nil {
				t.Fatalf("first download: %v\n%s", err, output)
			}
			command, output = downloadCommand(t, root, "DOWNLOAD_TEST_CONTENT=refreshed source")
			if err := command.Run(); err != nil {
				t.Fatalf("repeat download: %v\n%s", err, output)
			}
			want, requests := "complete source", "request\n"
			if branch {
				want, requests = "refreshed source", "request\nrequest\n"
			}
			assertDownload(t, root, want)
			if data, err := os.ReadFile(filepath.Join(root, "requests")); err != nil || string(data) != requests {
				t.Fatalf("requests = %q, %v; want %q", data, err, requests)
			}
		})
	}
}

func TestFailedRefreshPreservesSource(t *testing.T) {
	for _, result := range []string{"failure", "empty"} {
		t.Run(result, func(t *testing.T) {
			root := downloadFixture(t, true)
			command, output := downloadCommand(t, root)
			if err := command.Run(); err != nil {
				t.Fatalf("initial download: %v\n%s", err, output)
			}
			command, output = downloadCommand(t, root, "DOWNLOAD_TEST_RESULT="+result)
			if err := command.Run(); err == nil {
				t.Fatalf("accepted %s download: %s", result, output)
			}
			assertDownload(t, root, "complete source")
			command, output = downloadCommand(t, root)
			if err := command.Run(); err != nil {
				t.Fatalf("rerun after failure: %v\n%s", err, output)
			}
		})
	}
}

func TestConcurrentDownloads(t *testing.T) {
	for _, branch := range []bool{false, true} {
		t.Run(fmt.Sprintf("branch=%t", branch), func(t *testing.T) {
			root := downloadFixture(t, branch)
			first, firstOutput := downloadCommand(t, root, "DOWNLOAD_TEST_HOLD=1")
			startDownload(t, first)
			waitForFile(t, filepath.Join(root, "started"))
			second, secondOutput := downloadCommand(t, root, "DOWNLOAD_TEST_CONTENT=second source")
			assertContendingDownload(t, root, second)
			if err := os.WriteFile(filepath.Join(root, "proceed"), nil, 0600); err != nil {
				t.Fatal(err)
			}
			if err := first.Wait(); err != nil {
				t.Fatalf("first download: %v\n%s", err, firstOutput)
			}
			if err := second.Wait(); err != nil {
				t.Fatalf("second download: %v\n%s", err, secondOutput)
			}
			want, count := "complete source", 1
			if branch {
				want, count = "second source", 2
			}
			assertDownload(t, root, want)
			data, err := os.ReadFile(filepath.Join(root, "requests"))
			if err != nil || strings.Count(string(data), "request\n") != count {
				t.Fatalf("requests = %q, %v; want %d", data, err, count)
			}
		})
	}
}

func TestInterruptedDownloadCanRerun(t *testing.T) {
	for _, signal := range []syscall.Signal{syscall.SIGTERM, syscall.SIGKILL} {
		t.Run(signal.String(), func(t *testing.T) {
			root := downloadFixture(t, true)
			first, output := downloadCommand(t, root, "DOWNLOAD_TEST_HOLD=1")
			// A regular file lets Wait reap the parent independently of its orphan.
			downloadLog(t, first)
			startDownload(t, first)
			waitForFile(t, filepath.Join(root, "started"))
			if err := first.Process.Signal(signal); err != nil {
				t.Fatal(err)
			}
			if err := first.Wait(); err == nil {
				t.Fatalf("interrupted download succeeded: %s", output)
			}
			second, output := downloadCommand(t, root)
			if signal == syscall.SIGKILL {
				assertContendingDownload(t, root, second)
				if err := os.WriteFile(filepath.Join(root, "proceed"), nil, 0600); err != nil {
					t.Fatal(err)
				}
				if err := second.Wait(); err != nil {
					t.Fatalf("rerun after orphan release: %v\n%s", err, output)
				}
			} else if err := second.Run(); err != nil {
				t.Fatalf("rerun after interruption: %v\n%s", err, output)
			}
			assertDownload(t, root, "complete source")
		})
	}
}

func TestTargetRequired(t *testing.T) {
	if err := run(context.Background(), nil); err == nil {
		t.Fatal("accepted a missing target")
	}
}

func TestAbandonedPartialCleanup(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		t.Run(fmt.Sprintf("blocked=%t", blocked), func(t *testing.T) {
			root := downloadFixture(t, false)
			command, output := downloadCommand(t, root)
			if err := command.Run(); err != nil {
				t.Fatalf("initial download: %v\n%s", err, output)
			}
			partial := filepath.Join(root, ".cache", "downloads", "source.tar.gz.tmp.abandoned")
			if blocked {
				if err := os.Mkdir(partial, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(partial, "keep"), nil, 0600); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(partial, []byte("incomplete"), 0600); err != nil {
				t.Fatal(err)
			}
			command, output = downloadCommand(t, root)
			err := command.Run()
			if blocked {
				if err == nil {
					t.Fatalf("cleanup failure was ignored: %s", output)
				}
				if err := os.Remove(filepath.Join(partial, "keep")); err != nil {
					t.Fatal(err)
				}
				command, output = downloadCommand(t, root)
				err = command.Run()
			}
			if err != nil {
				t.Fatalf("recovery download: %v\n%s", err, output)
			}
			assertDownload(t, root, "complete source")
		})
	}
}

func TestInsecureRefreshPreservesSource(t *testing.T) {
	root := downloadFixture(t, true)
	command, output := downloadCommand(t, root)
	if err := command.Run(); err != nil {
		t.Fatalf("initial download: %v\n%s", err, output)
	}
	path := filepath.Join(root, "metadata.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, bytes.ReplaceAll(data, []byte("https://"), []byte("http://")), 0600); err != nil {
		t.Fatal(err)
	}
	command, output = downloadCommand(t, root)
	if err := command.Run(); err == nil || !strings.Contains(output.String(), "https only") {
		t.Fatalf("insecure refresh: %v\n%s", err, output)
	}
	assertDownload(t, root, "complete source")
}

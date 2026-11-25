package main

import (
	"bytes"
	"context"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestDownloadProcess(t *testing.T) {
	if os.Getenv("DOWNLOAD_TEST_PROCESS") == "1" {
		os.Args = []string{"download", setting("DOWNLOAD_TEST_TARGET", "fixture")}
		main()
		os.Exit(0)
	}
}

func TestTargetDownloadsAreIndependent(t *testing.T) {
	root := downloadFixture(t, false)
	if err := os.WriteFile(filepath.Join(root, "metadata.json"), []byte(`{
		"fixture":{"env":{"VERSION":"1"},"downloads":[{"url":"https://example.invalid/first.tar.gz","name":"source.tar.gz"}]},
		"other":{"env":{"VERSION":"1"},"downloads":[{"url":"https://example.invalid/second.tar.gz","name":"source.tar.gz"}]}
	}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".cache", "downloads"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".cache", "downloads", "source.tar.gz"), []byte("legacy archive"), 0600); err != nil {
		t.Fatal(err)
	}
	first, firstOutput := downloadCommand(t, root, "DOWNLOAD_TEST_HOLD=1")
	startDownload(t, first)
	waitForFile(t, filepath.Join(root, "started"))
	second, secondOutput := downloadCommand(t, root, "DOWNLOAD_TEST_TARGET=other", "DOWNLOAD_TEST_CONTENT=other source")
	if err := second.Run(); err != nil {
		t.Fatalf("independent target blocked or failed: %v\n%s", err, secondOutput)
	}
	if err := os.WriteFile(filepath.Join(root, "proceed"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := first.Wait(); err != nil {
		t.Fatalf("first target: %v\n%s", err, firstOutput)
	}
	assertDownload(t, root, "complete source")
	for _, target := range []string{"fixture", "other"} {
		command, output := downloadCommand(t, root, "DOWNLOAD_TEST_TARGET="+target, "DOWNLOAD_TEST_CONTENT=unexpected replacement")
		if err := command.Run(); err != nil {
			t.Fatalf("cached target %s: %v\n%s", target, err, output)
		}
	}
	assertDownload(t, root, "complete source")
	if data, err := os.ReadFile(filepath.Join(root, ".cache", "downloads", "other", "source.tar.gz")); err != nil || string(data) != "other source" {
		t.Fatalf("other archive = %q, %v", data, err)
	}
	if data, err := os.ReadFile(filepath.Join(root, ".cache", "downloads", "source.tar.gz")); err != nil || string(data) != "legacy archive" {
		t.Fatalf("legacy archive changed: %q, %v", data, err)
	}
	if data, err := os.ReadFile(filepath.Join(root, "requests")); err != nil || string(data) != "request\nrequest\n" {
		t.Fatalf("expected one download per target: %q, %v", data, err)
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
echo request >> requests
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
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	t.Cleanup(cancel)
	command := exec.CommandContext(ctx, executable, "-test.run=^TestDownloadProcess$")
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
	data, err := os.ReadFile(filepath.Join(root, ".cache", "downloads", "fixture", "source.tar.gz"))
	if err != nil || string(data) != want {
		t.Fatalf("archive = %q, %v; want %q", data, err, want)
	}
	partials, err := filepath.Glob(filepath.Join(root, ".cache", "downloads", "fixture", "*.tmp.*"))
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
	partials, err := filepath.Glob(filepath.Join(root, ".cache", "downloads", "fixture", "*.tmp.*"))
	if err != nil || len(partials) != 1 {
		t.Fatalf("expected one active partial: %v, %v", partials, err)
	}
	original, err := os.Stat(partials[0])
	if err != nil {
		t.Fatal(err)
	}
	lock, err := os.OpenFile(filepath.Join(root, ".cache", "downloads", "fixture", "source.tar.gz.lock"), os.O_RDWR, 0)
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
			partial := filepath.Join(root, ".cache", "downloads", "fixture", "source.tar.gz.tmp.abandoned")
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

func TestDownloadHelpWithoutMetadata(t *testing.T) {
	command := exec.Command("go", "run", ".", "--help")
	command.Env = append(os.Environ(), "METADATA_PATH_OVERRIDE="+filepath.Join(t.TempDir(), "absent.json"))
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("help failed: %v\n%s", err, output)
	}
}

type downloadRedirectCase struct {
	name, path string
	requests   int32
	fails      bool
}

func TestDownloadRedirectPolicy(t *testing.T) {
	for _, transport := range []string{"curl", "wget"} {
		t.Run(transport, func(t *testing.T) {
			executable, err := exec.LookPath(transport)
			if err != nil {
				t.Skipf("requires installed %s: %v", transport, err)
			}
			tools := map[string]string{transport: executable}
			cases := []downloadRedirectCase{
				{name: "HTTPS chain", path: "https", requests: 3},
				{name: "HTTP downgrade", path: "downgrade", requests: 2, fails: true},
			}
			if transport == "wget" {
				for _, helper := range []string{"timeout", "gtimeout", "ssl_client", "openssl"} {
					if path, err := exec.LookPath(helper); err == nil {
						tools[helper] = path
					}
				}
				if tools["timeout"] == "" && tools["gtimeout"] == "" {
					t.Skip("wget fallback requires timeout or gtimeout")
				}
				help, err := exec.CommandContext(t.Context(), executable, "--help").CombinedOutput()
				if len(bytes.TrimSpace(help)) == 0 {
					t.Skipf("cannot read wget's supported options: %v", err)
				}
				cases = []downloadRedirectCase{{name: "unsupported policy", path: "direct", fails: true}}
				if bytes.Contains(help, []byte("--max-redirect")) {
					cases = []downloadRedirectCase{
						{name: "direct HTTPS", path: "direct", requests: 1},
						{name: "redirect refused", path: "https", requests: 1, fails: true},
						{name: "downgrade refused", path: "downgrade", requests: 1, fails: true},
					}
				}
			}
			path := t.TempDir()
			for name, executable := range tools {
				if err := os.Symlink(executable, filepath.Join(path, name)); err != nil {
					t.Fatal(err)
				}
			}
			for _, test := range cases {
				t.Run(test.name, func(t *testing.T) {
					checkDownloadRedirect(t, transport, path, test)
				})
			}
		})
	}
}

func checkDownloadRedirect(t *testing.T, transport, path string, test downloadRedirectCase) {
	t.Helper()
	const (
		branchURL  = "/archive/refs/heads/"
		previous   = "existing archive"
		downloaded = "secure source"
	)
	var httpsRequests, httpRequests, tlsConnections atomic.Int32
	clear := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		httpRequests.Add(1)
		_, _ = w.Write([]byte("insecure source"))
	}))
	t.Cleanup(clear.Close)
	secure := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpsRequests.Add(1)
		switch r.URL.Path {
		case branchURL + "https", branchURL + "downgrade":
			http.Redirect(w, r, "/hop/"+strings.TrimPrefix(r.URL.Path, branchURL), http.StatusFound)
		case "/hop/https":
			http.Redirect(w, r, "/archive", http.StatusFound)
		case "/hop/downgrade":
			http.Redirect(w, r, clear.URL+"/archive", http.StatusFound)
		case branchURL + "direct", "/archive":
			_, _ = w.Write([]byte(downloaded))
		default:
			http.NotFound(w, r)
		}
	}))
	secure.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			tlsConnections.Add(1)
		}
	}
	secure.StartTLS()
	t.Cleanup(secure.Close)
	root := t.TempDir()
	archive := filepath.Join(".cache", "downloads", "fixture", "source.tar.gz")
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(archive)), 0755); err != nil {
		t.Fatal(err)
	}
	caPath := filepath.Join(root, "ca.pem")
	metadata := fmt.Sprintf(`{"fixture":{"env":{"VERSION":"1"},"downloads":[{"url":%q,"name":"source.tar.gz"}]}}`, secure.URL+branchURL+test.path)
	for name, content := range map[string][]byte{
		"metadata.json": []byte(metadata),
		archive:         []byte(previous),
		".curlrc":       {},
		"ca.pem":        pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: secure.Certificate().Raw}),
		"wgetrc":        []byte("ca_certificate = " + caPath + "\n"),
	} {
		if err := os.WriteFile(filepath.Join(root, name), content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	command, output := downloadCommand(t, root, "PATH="+path, "CURL_HOME="+root, "CURL_CA_BUNDLE="+caPath,
		"WGETRC="+filepath.Join(root, "wgetrc"), "NO_PROXY=*", "no_proxy=*", "http_proxy=", "https_proxy=",
		"DOWNLOAD_CONNECT_TIMEOUT=3", "DOWNLOAD_MAX_TIME=5")
	if err := command.Run(); (err != nil) != test.fails {
		t.Fatalf("download error = %v, want failure %t\n%s", err, test.fails, output)
	}
	wantArchive := downloaded
	if test.fails {
		wantArchive = previous
		if transport == "wget" && (!strings.Contains(output.String(), "--max-redirect=0") || !strings.Contains(output.String(), "Use curl")) {
			t.Fatalf("missing wget policy diagnostic: %s", output)
		}
	}
	assertDownload(t, root, wantArchive)
	if got := httpsRequests.Load(); got != test.requests {
		t.Fatalf("HTTPS requests = %d, want %d", got, test.requests)
	}
	if got := httpRequests.Load(); got != 0 {
		t.Fatalf("HTTP requests = %d, want 0", got)
	}
	if got := tlsConnections.Load(); test.requests == 0 && got != 0 {
		t.Fatalf("HTTPS connections = %d, want 0", got)
	}
}

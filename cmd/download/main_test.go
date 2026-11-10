package main

import (
	"bytes"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if os.Getenv("DOWNLOAD_FIXTURE") == "1" {
		name := filepath.Base(os.Args[0])
		program := os.Getenv("DOWNLOAD_FIXTURE_LIMITER")
		pidFile := os.Getenv("DOWNLOAD_FIXTURE_LIMITER_PID")
		if name == "wget" {
			program = os.Getenv("DOWNLOAD_FIXTURE_WGET")
			pidFile = os.Getenv("DOWNLOAD_FIXTURE_WGET_PID")
			if os.Getenv("DOWNLOAD_FIXTURE_SCENARIO") == "resistant" {
				signal.Ignore(syscall.SIGTERM)
			}
		}
		if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if name == "gtimeout" {
			name = "timeout"
		}
		if err := syscall.Exec(program, append([]string{name}, os.Args[1:]...), os.Environ()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	os.Exit(m.Run())
}

func fixturePID(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	pid, _ := strconv.Atoi(string(data))
	return pid
}

func fixtureRunning(pid int) bool {
	if pid == 0 {
		return false
	}
	output, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "stat=").Output()
	return err == nil && !strings.HasPrefix(strings.TrimSpace(string(output)), "Z")
}

func TestDownloadEndToEnd(t *testing.T) {
	wget, err := exec.LookPath("wget")
	if err != nil {
		t.Fatal("wget is required:", err)
	}
	limiter, err := exec.LookPath("timeout")
	if err != nil {
		limiter, err = exec.LookPath("gtimeout")
		if err != nil {
			t.Fatal("timeout or gtimeout is required:", err)
		}
	}
	dir := t.TempDir()
	binary := filepath.Join(dir, "download")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build download: %v\n%s", err, output)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scenario := strings.Split(r.URL.Path, "/")[1]
		if scenario == "failure" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if scenario == "success" {
			_, _ = w.Write([]byte("archive"))
			return
		}
		w.Header().Set("Content-Length", "100")
		for range 100 {
			if _, err := w.Write([]byte("x")); err != nil {
				return
			}
			w.(http.Flusher).Flush()
			select {
			case <-r.Context().Done():
				return
			case <-time.After(100 * time.Millisecond):
			}
		}
	}))
	defer server.Close()
	cert := filepath.Join(dir, "certificate.pem")
	if err := os.WriteFile(cert, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0600); err != nil {
		t.Fatal(err)
	}

	for _, scenario := range []string{"success", "slow", "resistant", "gtimeout", "signal", "failure", "missing_timeout", "fixture_timeout"} {
		t.Run(scenario, func(t *testing.T) {
			work := t.TempDir()
			bin := filepath.Join(work, "bin")
			if err := os.Mkdir(bin, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(executable, filepath.Join(bin, "wget")); err != nil {
				t.Fatal(err)
			}
			if scenario != "missing_timeout" {
				name := "timeout"
				if scenario == "gtimeout" {
					name = "gtimeout"
				}
				if err := os.Symlink(executable, filepath.Join(bin, name)); err != nil {
					t.Fatal(err)
				}
			}
			metadata, err := json.Marshal(map[string]any{"fixture": map[string]any{
				"env":       map[string]string{"VERSION": "1"},
				"downloads": []map[string]string{{"url": server.URL + "/" + scenario + "/archive/refs/heads/main.tar.gz", "name": "fixture.tar.gz"}},
			}})
			if err != nil {
				t.Fatal(err)
			}
			metadataPath := filepath.Join(work, "metadata.json")
			if err := os.WriteFile(metadataPath, metadata, 0600); err != nil {
				t.Fatal(err)
			}
			wgetrc := filepath.Join(work, "wgetrc")
			if err := os.WriteFile(wgetrc, []byte("ca_certificate = "+cert+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			wgetPID := filepath.Join(work, "wget.pid")
			limiterPID := filepath.Join(work, "timeout.pid")
			maxTime := "10"
			if scenario == "slow" || scenario == "resistant" || scenario == "gtimeout" {
				maxTime = "1"
			}
			cmd := exec.Command(binary, "fixture")
			cmd.Dir = work
			cmd.Env = append(os.Environ(), "PATH="+bin, "WGETRC="+wgetrc, "METADATA_PATH_OVERRIDE="+metadataPath,
				"DOWNLOAD_FIXTURE=1", "DOWNLOAD_FIXTURE_SCENARIO="+scenario,
				"DOWNLOAD_FIXTURE_WGET="+wget, "DOWNLOAD_FIXTURE_LIMITER="+limiter,
				"DOWNLOAD_FIXTURE_WGET_PID="+wgetPID, "DOWNLOAD_FIXTURE_LIMITER_PID="+limiterPID,
				"DOWNLOAD_RETRIES=1", "DOWNLOAD_CONNECT_TIMEOUT=1", "DOWNLOAD_MAX_TIME="+maxTime)
			cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
			var output bytes.Buffer
			cmd.Stdout, cmd.Stderr = &output, &output
			started := time.Now()
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			finished := false
			defer func() {
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				if pid := fixturePID(limiterPID); pid != 0 {
					_ = syscall.Kill(-pid, syscall.SIGKILL)
				}
				for _, path := range []string{wgetPID, limiterPID} {
					if pid := fixturePID(path); pid != 0 {
						_ = syscall.Kill(pid, syscall.SIGKILL)
					}
				}
				if !finished {
					select {
					case <-done:
					case <-time.After(3 * time.Second):
						t.Error("fixture driver failed to reap download")
					}
				}
				deadline := time.Now().Add(3 * time.Second)
				for (fixtureRunning(fixturePID(wgetPID)) || fixtureRunning(fixturePID(limiterPID))) && time.Now().Before(deadline) {
					time.Sleep(20 * time.Millisecond)
				}
				if fixtureRunning(fixturePID(wgetPID)) || fixtureRunning(fixturePID(limiterPID)) {
					t.Error("fixture left a child running")
				}
			}()
			if scenario == "signal" {
				deadline := time.Now().Add(5 * time.Second)
				for fixturePID(wgetPID) == 0 && time.Now().Before(deadline) {
					time.Sleep(20 * time.Millisecond)
				}
				if fixturePID(wgetPID) == 0 {
					t.Fatal("wget did not start")
				}
				if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
					t.Fatal(err)
				}
			}
			deadline := 6 * time.Second
			if scenario == "fixture_timeout" {
				deadline = time.Second
			}
			select {
			case err := <-done:
				finished = true
				if scenario == "fixture_timeout" {
					t.Fatal("fixture finished before driver deadline")
				}
				if scenario == "success" && err != nil || scenario != "success" && (err == nil || cmd.ProcessState.ExitCode() != 1) {
					t.Fatalf("unexpected exit: %v\n%s", err, &output)
				}
			case <-time.After(deadline):
				if scenario != "fixture_timeout" {
					t.Fatal("download exceeded driver deadline")
				}
				if fixturePID(wgetPID) == 0 || fixturePID(limiterPID) == 0 {
					t.Fatal("fixture deadline reached before both children started")
				}
				return
			}
			if (scenario == "slow" || scenario == "resistant" || scenario == "gtimeout") && time.Since(started) > 4*time.Second {
				t.Fatal("one-second download deadline took too long")
			}
			if scenario == "missing_timeout" && !strings.Contains(output.String(), "require timeout") {
				t.Fatalf("missing timeout diagnostic: %s", &output)
			}
			if scenario != "missing_timeout" && (fixturePID(wgetPID) == 0 || fixturePID(limiterPID) == 0) {
				t.Error("download did not start both fixture processes")
			}
			if fixtureRunning(fixturePID(wgetPID)) || fixtureRunning(fixturePID(limiterPID)) {
				t.Error("download left a child running")
			}
			entries, err := os.ReadDir(filepath.Join(work, ".tmp"))
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "success" {
				if len(entries) != 1 || entries[0].Name() != "fixture.tar.gz" {
					t.Fatalf("unexpected download artifacts: %v", entries)
				}
				data, err := os.ReadFile(filepath.Join(work, ".tmp", "fixture.tar.gz"))
				if err != nil || string(data) != "archive" {
					t.Fatalf("download content %q: %v", data, err)
				}
			} else if len(entries) != 0 {
				t.Fatalf("temporary files or locks remain: %v", entries)
			}
		})
	}
}

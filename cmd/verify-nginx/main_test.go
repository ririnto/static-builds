package main

import (
	"fmt"
	"net"
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
	if os.Getenv("NGINX_GO_FIXTURE") != "1" {
		os.Exit(m.Run())
	}
	arguments := os.Args[1:]
	directory := os.Getenv("NGINX_FIXTURE_DIR")
	scenario := os.Getenv("NGINX_SCENARIO")
	record := func(name string) {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
			panic(err)
		}
	}
	switch filepath.Base(os.Args[0]) {
	case "readelf":
		if strings.HasPrefix(scenario, "setup_") {
			signal.Ignore(syscall.SIGTERM)
			record("setup.pid")
			for {
				time.Sleep(time.Hour)
			}
		}
		fmt.Println("Type: DYN\nMachine: fixture")
	case "file":
		fmt.Println("ELF statically linked")
	case "ldd":
		fmt.Println("not a dynamic executable")
	case "strace":
		if err := syscall.Exec(arguments[0], arguments, os.Environ()); err != nil {
			panic(err)
		}
	case "timeout":
		records, err := os.OpenFile(filepath.Join(directory, "timeout.pids"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}
		fmt.Fprintln(records, os.Getpid())
		records.Close()
		if err := syscall.Exec(os.Getenv("REAL_TIMEOUT"), append([]string{"timeout"}, arguments...), os.Environ()); err != nil {
			panic(err)
		}
	case "curl":
		count := 1
		if value, err := os.ReadFile(filepath.Join(directory, "count")); err == nil {
			count, _ = strconv.Atoi(string(value))
			count++
		}
		os.WriteFile(filepath.Join(directory, "count"), []byte(strconv.Itoa(count)), 0600)
		record("curl.pid")
		for i, value := range arguments {
			if value == "-o" {
				os.WriteFile(filepath.Join(directory, "response.path"), []byte(arguments[i+1]), 0600)
			}
		}
		if scenario == "http_error_first" && count == 1 || scenario == "http_error_second" && count == 2 {
			os.Exit(22)
		}
		if scenario == "http_deadline" || scenario == "signal_int" || scenario == "signal_term" || scenario == "active_probe_cancellation" {
			signal.Ignore(syscall.SIGTERM)
			for {
				time.Sleep(time.Hour)
			}
		}
		for i, value := range arguments {
			if value == "-o" {
				os.WriteFile(arguments[i+1], []byte("njs 1.0.2 engine QuickJS\n"), 0600)
			}
		}
	case "nginx":
		for _, value := range arguments {
			if value == "-V" || value == "-t" {
				os.Exit(0)
			}
		}
		record("nginx.pid")
		if scenario == "resistant_nginx" {
			signal.Ignore(syscall.SIGQUIT, syscall.SIGTERM)
			for {
				time.Sleep(time.Hour)
			}
		}
		received := make(chan os.Signal, 1)
		signal.Notify(received, syscall.SIGQUIT, syscall.SIGTERM)
		<-received
	}
	os.Exit(0)
}

func TestVerifierScenarios(t *testing.T) {
	limiter, err := exec.LookPath("timeout")
	if err != nil {
		limiter, err = exec.LookPath("gtimeout")
	}
	if err != nil {
		t.Fatal("GNU timeout is required:", err)
	}
	directory := t.TempDir()
	binary := filepath.Join(directory, "verify-nginx")
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, output)
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []struct {
		name      string
		status    int
		interrupt syscall.Signal
	}{
		{"success", 0, 0}, {"setup_int", 130, syscall.SIGINT}, {"setup_term", 143, syscall.SIGTERM}, {"http_error_first", 22, 0}, {"http_error_second", 22, 0}, {"http_deadline", 137, 0}, {"signal_int", 130, syscall.SIGINT}, {"signal_term", 143, syscall.SIGTERM}, {"active_probe_cancellation", 143, syscall.SIGTERM}, {"resistant_nginx", 0, 0},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			fixture := t.TempDir()
			tools := filepath.Join(fixture, "bin")
			prefix := filepath.Join(fixture, "target")
			os.MkdirAll(tools, 0755)
			os.MkdirAll(filepath.Join(prefix, "sbin"), 0755)
			for _, name := range []string{"readelf", "file", "ldd", "strace", "timeout", "curl"} {
				if err := os.Symlink(self, filepath.Join(tools, name)); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Symlink(self, filepath.Join(prefix, "sbin/nginx")); err != nil {
				t.Fatal(err)
			}
			listener, err := net.Listen("tcp", "127.0.0.1:18082")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			served := make(chan struct{})
			go func() {
				defer close(served)
				connection, err := listener.Accept()
				if err == nil {
					fmt.Fprintln(connection, "njs 1.0.2 engine QuickJS")
					connection.Close()
				}
			}()
			defer func() { listener.Close(); <-served }()
			process := exec.Command(binary)
			process.Env = append(os.Environ(), "PATH="+tools, "TARGET_PREFIX="+prefix, "NJS_VERSION=1.0.2", "NGINX_GO_FIXTURE=1", "NGINX_FIXTURE_DIR="+fixture, "NGINX_SCENARIO="+scenario.name, "REAL_TIMEOUT="+limiter)
			process.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			var output strings.Builder
			process.Stdout, process.Stderr = &output, &output
			if err := process.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- process.Wait() }()
			if scenario.interrupt != 0 {
				deadline := time.Now().Add(8 * time.Second)
				marker := "curl.pid"
				if strings.HasPrefix(scenario.name, "setup_") {
					marker = "setup.pid"
				}
				for time.Now().Before(deadline) {
					if _, err := os.Stat(filepath.Join(fixture, marker)); err == nil {
						break
					}
					time.Sleep(20 * time.Millisecond)
				}
				if err := process.Process.Signal(scenario.interrupt); err != nil {
					t.Fatal(err)
				}
			}
			select {
			case <-done:
			case <-time.After(25 * time.Second):
				syscall.Kill(-process.Process.Pid, syscall.SIGKILL)
				<-done
				t.Fatal("verifier deadline exceeded")
			}
			if process.ProcessState.ExitCode() != scenario.status {
				t.Fatalf("status=%d expected=%d\n%s", process.ProcessState.ExitCode(), scenario.status, output.String())
			}
			if path, err := os.ReadFile(filepath.Join(fixture, "response.path")); err == nil {
				if _, err := os.Stat(string(path)); !os.IsNotExist(err) {
					t.Errorf("response file remains: %s", path)
				}
			}
			for _, name := range []string{"nginx.pid", "curl.pid", "timeout.pids", "setup.pid"} {
				contents, _ := os.ReadFile(filepath.Join(fixture, name))
				for _, value := range strings.Fields(string(contents)) {
					pid, _ := strconv.Atoi(value)
					deadline := time.Now().Add(3 * time.Second)
					for syscall.Kill(pid, 0) == nil && time.Now().Before(deadline) {
						time.Sleep(20 * time.Millisecond)
					}
					if syscall.Kill(pid, 0) == nil {
						syscall.Kill(pid, syscall.SIGKILL)
						t.Errorf("unreaped fixture %s pid=%d", name, pid)
					}
				}
			}
		})
	}
}

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func init() {
	if os.Getenv("HEALTHCHECK_LIFECYCLE_FIXTURE") == "" {
		return
	}
	name := filepath.Base(os.Args[0])
	if os.Getenv("HEALTHCHECK_CHILD") == "1" {
		name = "descendant"
	}
	if name == "nginx" && len(os.Args) > 1 && os.Args[1] == "-t" {
		config := os.Args[len(os.Args)-1]
		if strings.Contains(config, "missing-") {
			prefix := "TLS"
			if strings.Contains(filepath.Base(config), "http-") {
				prefix = "HTTPS"
			}
			fmt.Fprintln(os.Stderr, prefix+" healthcheck requires healthcheck_tls_name and healthcheck_trusted_certificate")
			os.Exit(1)
		}
		os.Exit(0)
	}
	if name == "dnsmasq" {
		for _, arg := range os.Args[1:] {
			if strings.HasPrefix(arg, "--pid-file=") {
				if err := os.WriteFile(filepath.Join(os.Getenv("HEALTHCHECK_LIFECYCLE_FIXTURE"), "directory"), []byte(filepath.Dir(strings.TrimPrefix(arg, "--pid-file="))), 0600); err != nil {
					os.Exit(2)
				}
			}
		}
	}
	if name == "socat" && len(os.Args) > 1 {
		if strings.Contains(os.Args[1], "19443") {
			name = "socat-tls"
		} else {
			name = "socat-http"
		}
	}
	if err := os.WriteFile(filepath.Join(os.Getenv("HEALTHCHECK_LIFECYCLE_FIXTURE"), name+".pid"), []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
		os.Exit(2)
	}
	if name == "dnsmasq" && os.Getenv("HEALTHCHECK_RESISTANT") == "1" {
		signal.Ignore(syscall.SIGTERM)
		for {
			time.Sleep(time.Hour)
		}
	}
	if name == "dnsmasq" {
		child := exec.Command(os.Args[0], "descendant")
		child.Env = append(os.Environ(), "HEALTHCHECK_CHILD=1")
		if os.Getenv("HEALTHCHECK_CHILD") != "1" {
			if err := child.Start(); err != nil {
				os.Exit(2)
			}
			if err := os.WriteFile(filepath.Join(os.Getenv("HEALTHCHECK_LIFECYCLE_FIXTURE"), "descendant.pid"), []byte(strconv.Itoa(child.Process.Pid)), 0600); err != nil {
				os.Exit(2)
			}
			received := make(chan os.Signal, 1)
			signal.Notify(received, syscall.SIGTERM)
			<-received
			_ = child.Wait()
			os.Exit(0)
		}
	}
	received := make(chan os.Signal, 1)
	signal.Notify(received, syscall.SIGTERM)
	<-received
	os.Exit(0)
}

func TestIntegrationLifecycle(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	runner := filepath.Join(directory, "run")
	if output, err := exec.Command("go", "build", "-o", runner, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	for _, args := range [][]string{nil, {filepath.Join(directory, "missing")}} {
		command := exec.Command(runner, args...)
		if output, err := command.CombinedOutput(); err == nil || command.ProcessState.ExitCode() != 2 {
			t.Fatalf("precondition %v: %v, %s", args, err, output)
		}
	}
	if runtime.GOOS != "linux" {
		t.Skip("fixture process lifecycle requires Linux")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(directory, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"nginx", "dnsmasq", "openssl", "socat", "sleep"} {
		if err := os.Symlink(executable, filepath.Join(bin, name)); err != nil {
			t.Fatal(err)
		}
	}
	for _, test := range []struct {
		name      string
		sig       syscall.Signal
		status    int
		resistant bool
	}{{"interrupt", syscall.SIGINT, 130, false}, {"terminate", syscall.SIGTERM, 143, false}, {"resistant", syscall.SIGTERM, 143, true}} {
		t.Run(test.name, func(t *testing.T) {
			markers := t.TempDir()
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, runner, filepath.Join(bin, "nginx"))
			command.Dir = root
			command.Env = append(os.Environ(), "PATH="+bin, "HEALTHCHECK_LIFECYCLE_FIXTURE="+markers)
			if test.resistant {
				command.Env = append(command.Env, "HEALTHCHECK_RESISTANT=1")
			}
			var output strings.Builder
			command.Stdout, command.Stderr = &output, &output
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			finished := make(chan error, 1)
			go func() { finished <- command.Wait() }()
			defer func() {
				if command.ProcessState == nil {
					_ = command.Process.Kill()
					<-finished
				}
			}()
			deadline := time.Now().Add(15 * time.Second)
			for time.Now().Before(deadline) {
				_, nginxErr := os.Stat(filepath.Join(markers, "nginx.pid"))
				_, tlsErr := os.Stat(filepath.Join(markers, "socat-tls.pid"))
				_, httpErr := os.Stat(filepath.Join(markers, "socat-http.pid"))
				_, dnsErr := os.Stat(filepath.Join(markers, "dnsmasq.pid"))
				_, childErr := os.Stat(filepath.Join(markers, "descendant.pid"))
				if nginxErr == nil && tlsErr == nil && httpErr == nil && dnsErr == nil && (test.resistant || childErr == nil) {
					break
				}
				time.Sleep(25 * time.Millisecond)
			}
			for _, name := range []string{"nginx", "dnsmasq", "socat-tls", "socat-http"} {
				if _, err := os.Stat(filepath.Join(markers, name+".pid")); err != nil {
					t.Fatalf("fixture %s did not start: %v output=%s", name, err, output.String())
				}
			}
			if err := command.Process.Signal(test.sig); err != nil {
				t.Fatal(err)
			}
			err := <-finished
			if ctx.Err() != nil || command.ProcessState.ExitCode() != test.status {
				t.Fatalf("status=%d err=%v context=%v output=%s", command.ProcessState.ExitCode(), err, ctx.Err(), output.String())
			}
			for _, name := range []string{"nginx", "dnsmasq", "socat-tls", "socat-http", "descendant"} {
				data, err := os.ReadFile(filepath.Join(markers, name+".pid"))
				if os.IsNotExist(err) && test.resistant && name == "descendant" {
					continue
				}
				if err != nil {
					t.Fatal(err)
				}
				pid, err := strconv.Atoi(string(data))
				if err != nil {
					t.Fatal(err)
				}
				if err := syscall.Kill(pid, 0); err != syscall.ESRCH {
					t.Errorf("%s pid %d still exists: %v", name, pid, err)
				}
			}
			data, err := os.ReadFile(filepath.Join(markers, "directory"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(string(data)); !os.IsNotExist(err) {
				t.Errorf("fixture directory remains: %s (%v)", data, err)
			}
		})
	}
}

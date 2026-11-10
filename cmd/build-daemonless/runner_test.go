package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if os.Getenv("DAEMONLESS_GO_FIXTURE") != "1" {
		os.Exit(m.Run())
	}
	dir := os.Getenv("DAEMONLESS_FIXTURE")
	scenario := os.Getenv("DAEMONLESS_SCENARIO")
	args := os.Args[1:]
	record := func(name string, value any) {
		data, err := json.Marshal(value)
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			panic(err)
		}
	}
	switch filepath.Base(os.Args[0]) {
	case "timeout":
		if err := syscall.Exec(os.Getenv("DAEMONLESS_TIMEOUT"), append([]string{"timeout"}, args...), os.Environ()); err != nil {
			panic(err)
		}
	case "go":
		if slices.Equal(args, []string{"tool", "buildctl", "--version"}) {
			os.Exit(0)
		}
		if slices.Equal(args, []string{"tool", "-n", "buildkitd"}) {
			fmt.Println(filepath.Join(dir, "bin", "buildkitd"))
			os.Exit(0)
		}
		if len(args) < 2 || !slices.Equal(args[:2], []string{"tool", "buildctl"}) {
			panic(args)
		}
		args = args[2:]
		if slices.Contains(args, "workers") {
			record("workers.json", args)
			if scenario == "startup_failure" {
				os.Exit(19)
			}
			if scenario == "readiness_deadline" {
				for {
					time.Sleep(time.Hour)
				}
			}
			if _, err := os.Stat(filepath.Join(dir, "daemon.pid")); err != nil {
				os.Exit(1)
			}
			fmt.Println("fixture-worker")
			os.Exit(0)
		}
		record("client.json", args)
		record("client.pid", os.Getpid())
		if strings.HasPrefix(scenario, "interrupt_") || scenario == "escalation" || scenario == "concurrent" {
			for {
				time.Sleep(time.Hour)
			}
		}
		if scenario == "client_failure" {
			os.Exit(37)
		}
	case "rootlesskit":
		record("rootlesskit.json", args)
		if err := syscall.Exec(args[3], args[3:], os.Environ()); err != nil {
			panic(err)
		}
	case "buildkitd":
		record("daemon.json", args)
		if scenario == "startup_failure" {
			record("daemon.pid", os.Getpid())
			os.Exit(23)
		}
		child := exec.Command(filepath.Join(dir, "bin", "descendant"))
		if err := child.Start(); err != nil {
			panic(err)
		}
		received := make(chan os.Signal, 2)
		signal.Notify(received, syscall.SIGTERM)
		record("descendant.pid", child.Process.Pid)
		record("daemon.pid", os.Getpid())
		<-received
		_ = child.Process.Kill()
		_ = child.Wait()
		if scenario == "escalation" {
			for {
				<-received
			}
		}
	case "descendant":
		for {
			time.Sleep(time.Hour)
		}
	case "setsid", "runc":
	default:
		panic(os.Args)
	}
	os.Exit(0)
}

func TestDaemonlessRunnerScenarios(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("daemonless runner requires a non-root user")
	}
	timeout, err := exec.LookPath("timeout")
	if err != nil {
		timeout, err = exec.LookPath("gtimeout")
	}
	if err != nil {
		t.Fatal("GNU timeout is required:", err)
	}
	repository, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "build-daemonless")
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, output)
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var metadata struct {
		Nginx struct {
			Env map[string]string `json:"env"`
		} `json:"nginx"`
	}
	data, err := os.ReadFile(filepath.Join(repository, "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []struct {
		name   string
		status int
		sig    syscall.Signal
	}{
		{"success", 0, 0}, {"client_failure", 37, 0}, {"startup_failure", 1, 0},
		{"interrupt_hup", 129, syscall.SIGHUP}, {"interrupt_int", 130, syscall.SIGINT}, {"interrupt_term", 143, syscall.SIGTERM},
		{"escalation", 143, syscall.SIGTERM}, {"readiness_deadline", 1, 0}, {"concurrent", 143, syscall.SIGTERM},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			type instance struct {
				dir    string
				cmd    *exec.Cmd
				output *bytes.Buffer
				done   chan struct{}
			}
			var instances []instance
			count := 1
			if scenario.name == "concurrent" {
				count = 2
			}
			for range count {
				dir := t.TempDir()
				for _, name := range []string{"bin", "tmp"} {
					if err := os.Mkdir(filepath.Join(dir, name), 0755); err != nil {
						t.Fatal(err)
					}
				}
				for _, name := range []string{"go", "rootlesskit", "buildkitd", "setsid", "timeout", "runc", "descendant"} {
					if err := os.Symlink(self, filepath.Join(dir, "bin", name)); err != nil {
						t.Fatal(err)
					}
				}
				network, backend := "none", "local"
				if scenario.name == "success" {
					network = "host"
				}
				if scenario.name == "client_failure" {
					backend = "gha"
				}
				cmd := exec.Command(binary, "nginx")
				cmd.Dir = repository
				cmd.Env = append(os.Environ(), "PATH="+filepath.Join(dir, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"), "DAEMONLESS_GO_FIXTURE=1", "GITLAB_CI=false", "DAEMONLESS_FIXTURE="+dir, "DAEMONLESS_SCENARIO="+scenario.name, "DAEMONLESS_TIMEOUT="+timeout, "TMPDIR="+filepath.Join(dir, "tmp"), "BUILD_OUTPUT_DEST="+filepath.Join(dir, "output with spaces"), "CACHE_DIR="+filepath.Join(dir, "cache with spaces"), "BUILDKIT_NETWORK="+network, "BUILDKIT_CACHE_BACKEND="+backend, "BUILDKIT_PLATFORM=linux/amd64", "BUILDKIT_HOST=unix:///foreign.sock", "BUILDKITD_FLAGS=--oci-worker-no-process-sandbox", "METADATA_PATH_OVERRIDE="+filepath.Join(repository, "metadata.json"))
				output := new(bytes.Buffer)
				cmd.Stdout, cmd.Stderr = output, output
				if err := cmd.Start(); err != nil {
					t.Fatal(err)
				}
				done := make(chan struct{})
				go func() { _ = cmd.Wait(); close(done) }()
				instances = append(instances, instance{dir, cmd, output, done})
				t.Cleanup(func() {
					select {
					case <-done:
					default:
						_ = cmd.Process.Signal(syscall.SIGTERM)
						select {
						case <-done:
						case <-time.After(20 * time.Second):
							_ = cmd.Process.Kill()
							<-done
						}
					}
				})
			}
			if scenario.sig != 0 {
				for _, instance := range instances {
					deadline := time.Now().Add(15 * time.Second)
					for time.Now().Before(deadline) {
						if _, err := os.Stat(filepath.Join(instance.dir, "client.pid")); err == nil {
							break
						}
						time.Sleep(20 * time.Millisecond)
					}
					if _, err := os.Stat(filepath.Join(instance.dir, "client.pid")); err != nil {
						t.Fatalf("client did not start: %v", err)
					}
				}
				for _, instance := range instances {
					if err := instance.cmd.Process.Signal(scenario.sig); err != nil {
						t.Fatal(err)
					}
				}
			}
			var addresses []string
			for _, instance := range instances {
				select {
				case <-instance.done:
				case <-time.After(85 * time.Second):
					_ = instance.cmd.Process.Kill()
					<-instance.done
					t.Fatal("runner deadline exceeded")
				}
				if instance.cmd.ProcessState.ExitCode() != scenario.status {
					t.Fatalf("status=%d expected=%d\n%s", instance.cmd.ProcessState.ExitCode(), scenario.status, instance.output)
				}
				for _, name := range []string{"daemon.pid", "client.pid", "descendant.pid"} {
					contents, err := os.ReadFile(filepath.Join(instance.dir, name))
					if os.IsNotExist(err) {
						continue
					}
					if err != nil {
						t.Fatal(err)
					}
					pid, err := strconv.Atoi(string(contents))
					if err != nil {
						t.Fatal(err)
					}
					deadline := time.Now().Add(5 * time.Second)
					for syscall.Kill(pid, 0) == nil && time.Now().Before(deadline) {
						time.Sleep(20 * time.Millisecond)
					}
					if err := syscall.Kill(pid, 0); err != syscall.ESRCH {
						_ = syscall.Kill(pid, syscall.SIGKILL)
						t.Errorf("process %s pid=%d survived: %v", name, pid, err)
					}
				}
				entries, err := os.ReadDir(filepath.Join(instance.dir, "tmp"))
				if err != nil || len(entries) != 0 {
					t.Errorf("private state remains: %v %v", entries, err)
				}
				readArgs := func(name string) []string {
					var args []string
					data, err := os.ReadFile(filepath.Join(instance.dir, name))
					if err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal(data, &args); err != nil {
						t.Fatal(err)
					}
					return args
				}
				daemon, rootless := readArgs("daemon.json"), readArgs("rootlesskit.json")
				if len(rootless) < 2 || !slices.Equal(rootless[:2], []string{"--pidns", "--reaper=true"}) {
					t.Errorf("rootlesskit arguments: %v", rootless)
				}
				address := ""
				for _, arg := range daemon {
					if strings.HasPrefix(arg, "--addr=") {
						address = arg
					}
				}
				if address == "" || strings.Contains(address, "foreign.sock") || slices.Contains(daemon, "--oci-worker-no-process-sandbox") {
					t.Errorf("daemon overrides: %v", daemon)
				}
				addresses = append(addresses, address)
				if _, err := os.Stat(filepath.Join(instance.dir, "client.json")); err == nil {
					client := readArgs("client.json")
					network := "none"
					if scenario.name == "success" {
						network = "host"
					}
					if len(client) == 0 || client[0] != address || !slices.Contains(client, "force-network-mode="+network) || slices.Contains(client, "network.host") != (network == "host") || slices.Contains(daemon, "network.host") != (network == "host") || !slices.Contains(client, "build-arg:NGINX_VERSION="+metadata.Nginx.Env["NGINX_VERSION"]) {
						t.Errorf("client or entitlement arguments: daemon=%v client=%v", daemon, client)
					}
				}
				if info, err := os.Stat(filepath.Join(instance.dir, "output with spaces")); err != nil || !info.IsDir() {
					t.Errorf("output not retained: %v", err)
				}
				_, err = os.Stat(filepath.Join(instance.dir, "cache with spaces"))
				if (err == nil) == (scenario.name == "client_failure") {
					t.Errorf("unexpected cache existence: %v", err)
				}
			}
			if len(addresses) == 2 && addresses[0] == addresses[1] {
				t.Error("concurrent runs shared an address")
			}
		})
	}
}

func TestGitLabDestinationAndCache(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("daemonless runner requires a non-root user")
	}
	timeout, err := exec.LookPath("timeout")
	if err != nil {
		timeout, err = exec.LookPath("gtimeout")
	}
	if err != nil {
		t.Fatal("GNU timeout is required:", err)
	}
	repository, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "build-daemonless")
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, output)
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := os.ReadFile(filepath.Join(repository, "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []struct {
		name, destination string
		status            int
	}{
		{"external_absolute", "external", 1}, {"parent_traversal", ".out/nginx/../../escaped-output", 1}, {"nested_traversal", ".out/nginx/nested/../outside_nested", 1},
		{"default_absolute", "", 0}, {"relative_target", ".out/nginx", 0}, {"relative_nested", ".out/nginx/nested", 0}, {"dot_prefix_nested", "./.out/nginx/nested", 0}, {"absolute_inside_nested", "inside", 0},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			dir, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			root, bin, tmp := filepath.Join(dir, "repo"), filepath.Join(dir, "bin"), filepath.Join(dir, "tmp")
			for _, path := range []string{filepath.Join(root, "nginx"), bin, tmp} {
				if err := os.MkdirAll(path, 0755); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(root, "metadata.json"), metadata, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "nginx", "Dockerfile"), []byte("FROM scratch\n"), 0600); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"go", "rootlesskit", "buildkitd", "setsid", "timeout", "runc", "descendant"} {
				if err := os.Symlink(self, filepath.Join(bin, name)); err != nil {
					t.Fatal(err)
				}
			}
			destination := scenario.destination
			if destination == "external" {
				destination = filepath.Join(dir, "outside")
			}
			if destination == "inside" {
				destination = filepath.Join(root, ".out", "nginx", "absolute")
			}
			expected := destination
			if destination == "" {
				expected = filepath.Join(root, ".out", "nginx")
			}
			path := expected
			if !filepath.IsAbs(path) {
				path = filepath.Join(root, path)
			}
			cmd := exec.Command(binary, "nginx")
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "DAEMONLESS_GO_FIXTURE=1", "DAEMONLESS_FIXTURE="+dir, "DAEMONLESS_SCENARIO=success", "DAEMONLESS_TIMEOUT="+timeout, "TMPDIR="+tmp, "GITLAB_CI=true", "BUILD_OUTPUT_DEST="+destination, "CACHE_DIR=", "METADATA_PATH_OVERRIDE="+filepath.Join(root, "metadata.json"), "BUILDKIT_CACHE_BACKEND=local", "BUILDKIT_NETWORK=default", "BUILDKIT_PLATFORM=linux/amd64")
			var output bytes.Buffer
			cmd.Stdout, cmd.Stderr = &output, &output
			err = cmd.Run()
			if cmd.ProcessState == nil || cmd.ProcessState.ExitCode() != scenario.status {
				t.Fatalf("status=%v expected=%d err=%v output=%s", cmd.ProcessState, scenario.status, err, &output)
			}
			_, outputErr := os.Stat(path)
			_, cacheErr := os.Stat(filepath.Join(root, ".cache", "nginx"))
			data, argsErr := os.ReadFile(filepath.Join(dir, "client.json"))
			if scenario.status != 0 {
				if !strings.Contains(output.String(), ".out/nginx") {
					t.Errorf("missing containment error: %s", &output)
				}
				_, daemonErr := os.Stat(filepath.Join(dir, "daemon.json"))
				entries, tmpErr := os.ReadDir(tmp)
				if !os.IsNotExist(outputErr) || !os.IsNotExist(cacheErr) || !os.IsNotExist(argsErr) || !os.IsNotExist(daemonErr) || tmpErr != nil || len(entries) != 0 {
					t.Errorf("invalid destination caused side effects: output=%v cache=%v exporter=%v daemon=%v private=%v %v", outputErr, cacheErr, argsErr, daemonErr, entries, tmpErr)
				}
			} else {
				if outputErr != nil || cacheErr != nil || argsErr != nil {
					t.Fatalf("missing output/cache/exporter: %v %v %v", outputErr, cacheErr, argsErr)
				}
				var args []string
				if err := json.Unmarshal(data, &args); err != nil {
					t.Fatal(err)
				}
				index := slices.Index(args, "--output")
				if index < 0 || index+1 >= len(args) || args[index+1] != "type=local,dest="+expected {
					t.Errorf("incorrect exporter arguments: %v", args)
				}
				if !slices.Contains(args, "type=local,src="+filepath.Join(root, ".cache", "nginx")) || !slices.Contains(args, "type=local,dest="+filepath.Join(root, ".cache", "nginx")+",mode=max") {
					t.Errorf("incorrect cache arguments: %v", args)
				}
			}
		})
	}
}

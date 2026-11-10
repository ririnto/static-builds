package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

type fixtureResource struct {
	Name  string `json:"name"`
	Owner string `json:"owner"`
}

type dockerState struct {
	Scenario  string           `json:"scenario"`
	Image     *fixtureResource `json:"image"`
	Container *fixtureResource `json:"container"`
	BaseImage string           `json:"base_image"`
	Calls     [][]string       `json:"calls"`
}

func TestDockerFixture(t *testing.T) {
	path := os.Getenv("HEALTHCHECK_DOCKER_FIXTURE")
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var state dockerState
	if err := json.Unmarshal(data, &state); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	args := os.Args[1:]
	state.Calls = append(state.Calls, args)
	option := func(name string) string {
		for i, value := range args {
			if value == name && i+1 < len(args) {
				return args[i+1]
			}
		}
		fmt.Fprintln(os.Stderr, "missing option", name)
		os.Exit(1)
		return ""
	}
	status, output := 0, ""
	switch {
	case len(args) >= 2 && args[0] == "buildx" && args[1] == "version":
		output = "mock buildx"
	case len(args) >= 2 && args[0] == "buildx" && args[1] == "build":
		state.BaseImage = strings.TrimPrefix(option("--build-arg"), "FUNCTIONAL_BASE_IMAGE=")
		owner := strings.TrimPrefix(option("--label"), "static-builds.healthcheck.run=")
		if state.Scenario == "foreign_image" {
			owner = "foreign"
		}
		state.Image = &fixtureResource{Name: option("--tag"), Owner: owner}
		if state.Scenario == "load_error" || state.Scenario == "foreign_image" {
			status = 41
		}
	case len(args) > 0 && args[0] == "create":
		owner := strings.TrimPrefix(option("--label"), "static-builds.healthcheck.run=")
		if state.Scenario == "foreign_container" {
			owner = "foreign"
		}
		state.Container = &fixtureResource{Name: option("--name"), Owner: owner}
		output = "mock-container-id"
		if state.Scenario == "create_error" || state.Scenario == "foreign_container" {
			status = 42
		}
	case len(args) >= 2 && args[0] == "container" && args[1] == "ls":
		if state.Container != nil {
			filters := strings.Join(args, "\x00")
			if strings.Contains(filters, "name=^/"+state.Container.Name+"$") && strings.Contains(filters, "label=static-builds.healthcheck.run="+state.Container.Owner) {
				output = "mock-container-id"
			}
		}
	case len(args) >= 2 && args[0] == "image" && args[1] == "ls":
		if state.Image != nil && args[len(args)-1] == state.Image.Name {
			output = "mock-image-id"
		}
	case len(args) >= 2 && args[0] == "image" && args[1] == "inspect":
		if state.Image == nil {
			status = 1
		} else {
			output = state.Image.Owner
		}
	case len(args) > 0 && args[0] == "rm":
		state.Container = nil
	case len(args) >= 2 && args[0] == "image" && args[1] == "rm":
		state.Image = nil
	case len(args) > 0 && args[0] == "wait":
		output = "0"
		if state.Scenario == "suite_error" {
			output = "17"
		}
	case len(args) > 0 && args[0] == "logs":
		output = "fixture suite output"
	case len(args) > 0 && args[0] == "start":
	default:
		fmt.Fprintln(os.Stderr, "unexpected docker arguments", args)
		os.Exit(1)
	}
	data, err = json.Marshal(state)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if state.Scenario == "load_signal" && len(args) >= 2 && args[0] == "buildx" && args[1] == "build" || state.Scenario == "create_signal" && len(args) > 0 && args[0] == "create" {
		if err := syscall.Kill(os.Getppid(), syscall.SIGTERM); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if output != "" {
		fmt.Println(output)
	}
	os.Exit(status)
}

func TestDockerRunner(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	runner := filepath.Join(directory, "healthcheck-docker")
	if output, err := exec.Command("go", "build", "-o", runner, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	docker := filepath.Join(directory, "docker")
	if err := os.Symlink(executable, docker); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(directory, "nginx")
	if err := os.WriteFile(input, []byte("fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	var metadata map[string]struct {
		Env map[string]string `json:"env"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatal(err)
	}
	base := "registry.access.redhat.com/ubi10/ubi-minimal:" + metadata["nginx-upstream-healthcheck"].Env["UBI10_MINIMAL_VERSION"]
	for _, test := range []struct {
		name   string
		status int
	}{
		{"success", 0}, {"load_error", 41}, {"create_error", 42}, {"load_signal", 143}, {"create_signal", 143}, {"foreign_image", 41}, {"foreign_container", 42}, {"suite_error", 17},
	} {
		t.Run(test.name, func(t *testing.T) {
			statePath := filepath.Join(t.TempDir(), "state.json")
			data, err := json.Marshal(dockerState{Scenario: test.name})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(statePath, data, 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, runner, input)
			command.Dir = root
			command.Env = append(os.Environ(), "PATH="+directory+string(os.PathListSeparator)+os.Getenv("PATH"), "HEALTHCHECK_DOCKER_FIXTURE="+statePath, "METADATA_PATH_OVERRIDE="+filepath.Join(root, "metadata.json"))
			var stdout, stderr strings.Builder
			command.Stdout, command.Stderr = &stdout, &stderr
			err = command.Run()
			if ctx.Err() != nil {
				t.Fatalf("timed out: %v", ctx.Err())
			}
			if err != nil && command.ProcessState == nil {
				t.Fatal(err)
			}
			if command.ProcessState.ExitCode() != test.status {
				t.Fatalf("status=%d want=%d stdout=%q stderr=%q", command.ProcessState.ExitCode(), test.status, stdout.String(), stderr.String())
			}
			if !strings.HasPrefix(stdout.String(), docker+"\nmock buildx\n") {
				t.Fatalf("command-check output missing: %q", stdout.String())
			}
			data, err = os.ReadFile(statePath)
			if err != nil {
				t.Fatal(err)
			}
			var state dockerState
			if err := json.Unmarshal(data, &state); err != nil {
				t.Fatal(err)
			}
			if state.BaseImage != base {
				t.Errorf("base image=%q want=%q", state.BaseImage, base)
			}
			var image, owner string
			for _, args := range state.Calls {
				if len(args) >= 2 && args[0] == "buildx" && args[1] == "build" {
					for i, arg := range args[:len(args)-1] {
						if arg == "--tag" {
							image = args[i+1]
						}
						if arg == "--label" {
							owner = strings.TrimPrefix(args[i+1], "static-builds.healthcheck.run=")
						}
					}
				}
				if len(args) > 0 && args[0] == "create" {
					if !strings.Contains(strings.Join(args, "\x00"), "static-builds.healthcheck.run="+owner) {
						t.Errorf("container label does not match image owner: %v", args)
					}
				}
			}
			if owner == "" || !strings.HasPrefix(image, "static-builds-healthcheck-suite:") || image != "static-builds-healthcheck-suite:"+owner {
				t.Errorf("invalid image ownership: image=%q owner=%q", image, owner)
			}
			for name, resource := range map[string]*fixtureResource{"image": state.Image, "container": state.Container} {
				if test.name == "foreign_"+name {
					if resource == nil || resource.Owner != "foreign" {
						t.Errorf("foreign %s removed: %+v", name, resource)
					}
				} else if resource != nil {
					t.Errorf("owned %s leaked: %+v", name, resource)
				}
			}
			if (test.name == "success" || test.name == "suite_error") && !strings.Contains(stdout.String(), "fixture suite output") {
				t.Errorf("suite output missing: %q", stdout.String())
			}
		})
	}
}

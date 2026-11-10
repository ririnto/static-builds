package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("BUILDX_GO_FIXTURE") != "1" {
		os.Exit(m.Run())
	}
	data, err := json.Marshal(os.Args[1:])
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(os.Getenv("BUILDX_ARGS"), data, 0600); err != nil {
		panic(err)
	}
	if os.Getenv("BUILDX_FAIL") == "1" {
		os.Exit(41)
	}
	os.Exit(0)
}

func TestBuildxRunner(t *testing.T) {
	repository, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "build-buildx")
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
		name, backend, fail string
		status              int
	}{{"local", "local", "0", 0}, {"gha_failure", "gha", "1", 41}} {
		t.Run(scenario.name, func(t *testing.T) {
			dir := t.TempDir()
			tool := filepath.Join(dir, "docker")
			if err := os.Symlink(self, tool); err != nil {
				t.Fatal(err)
			}
			argsPath, outputPath, cachePath := filepath.Join(dir, "args.json"), filepath.Join(dir, "output with spaces"), filepath.Join(dir, "cache with spaces")
			cmd := exec.Command(binary, "nginx", "--progress=plain")
			cmd.Dir = repository
			cmd.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"), "BUILDX_GO_FIXTURE=1", "BUILDX_ARGS="+argsPath, "BUILDX_FAIL="+scenario.fail, "BUILDKIT_CACHE_BACKEND="+scenario.backend, "BUILDKIT_NETWORK=host", "BUILDKIT_PLATFORM=linux/amd64", "BUILD_OUTPUT_DEST="+outputPath, "CACHE_DIR="+cachePath, "METADATA_PATH_OVERRIDE="+filepath.Join(repository, "metadata.json"))
			output, err := cmd.CombinedOutput()
			if status := cmd.ProcessState.ExitCode(); status != scenario.status {
				t.Fatalf("status=%d expected=%d err=%v output=%s", status, scenario.status, err, output)
			}
			data, err := os.ReadFile(argsPath)
			if err != nil {
				t.Fatal(err)
			}
			var args []string
			if err := json.Unmarshal(data, &args); err != nil {
				t.Fatal(err)
			}
			for _, expected := range []string{"buildx", "build", "--platform=linux/amd64", "--network=host", "--output=type=local,dest=" + outputPath, "--build-arg=NGINX_VERSION=" + metadata.Nginx.Env["NGINX_VERSION"], "--progress=plain", "--file=" + filepath.Join(repository, "nginx", "Dockerfile"), repository} {
				if !slices.Contains(args, expected) {
					t.Errorf("missing %q in %v", expected, args)
				}
			}
			cacheFrom, cacheTo := "--cache-from=type=local,src="+cachePath, "--cache-to=type=local,dest="+cachePath+",mode=max"
			if scenario.backend == "gha" {
				cacheFrom, cacheTo = "--cache-from=type=gha,scope=nginx", "--cache-to=type=gha,mode=max,scope=nginx"
			}
			if !slices.Contains(args, cacheFrom) || !slices.Contains(args, cacheTo) {
				t.Errorf("cache arguments: %v", args)
			}
			if info, err := os.Stat(outputPath); err != nil || !info.IsDir() {
				t.Errorf("output missing: %v", err)
			}
			_, err = os.Stat(cachePath)
			if (err == nil) != (scenario.backend == "local") {
				t.Errorf("cache existence: %v", err)
			}
		})
	}
}

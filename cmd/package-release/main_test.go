package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestReleaseCommands(t *testing.T) {
	directory := t.TempDir()
	commands := make(map[string]string)
	for _, name := range []string{"release-resolve", "release-verify", "package-release", "release-publish"} {
		commands[name] = filepath.Join(directory, name)
		if output, err := exec.Command("go", "build", "-o", commands[name], "../"+name).CombinedOutput(); err != nil {
			t.Fatalf("build %s: %v\n%s", name, err, output)
		}
	}
	metadata := filepath.Join(directory, "metadata.json")
	writeMetadata := func(value any) {
		t.Helper()
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(metadata, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	run := func(name string, args []string, outputPath *string) (int, string, string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, commands[name], args...)
		command.Dir = directory
		for _, entry := range os.Environ() {
			if !strings.HasPrefix(entry, "GITHUB_OUTPUT=") && !strings.HasPrefix(entry, "METADATA_PATH_OVERRIDE=") {
				command.Env = append(command.Env, entry)
			}
		}
		command.Env = append(command.Env, "METADATA_PATH_OVERRIDE="+metadata)
		if outputPath != nil {
			command.Env = append(command.Env, "GITHUB_OUTPUT="+*outputPath)
		}
		var stdout, stderr strings.Builder
		command.Stdout, command.Stderr = &stdout, &stderr
		err := command.Run()
		if ctx.Err() != nil {
			t.Fatalf("%s timed out: %v", name, ctx.Err())
		}
		if err != nil && command.ProcessState == nil {
			t.Fatalf("run %s: %v", name, err)
		}
		return command.ProcessState.ExitCode(), stdout.String(), stderr.String()
	}

	t.Run("resolve", func(t *testing.T) {
		target := map[string]any{"tag_prefix": "nginx", "version_env_var": "VERSION", "env": map[string]string{"VERSION": "1.2.3"}, "release_files": []string{"sbin/fixture", "conf"}}
		targets := map[string]any{"nginx": target, "nginx-upstream-healthcheck": map[string]any{"tag_prefix": "nginx-upstream-healthcheck", "version_env_var": "VERSION", "env": map[string]string{"VERSION": "1.2.3"}}, "apache-httpd": map[string]any{"tag_prefix": "httpd", "version_env_var": "VERSION", "env": map[string]string{"VERSION": "1.2.3"}}}
		writeMetadata(targets)
		output := filepath.Join(directory, "output")
		const sentinel = "existing=value\n"
		for _, test := range []struct{ tag, target string }{
			{"nginx-1.2.3.0", "nginx"}, {"nginx-1.2.3.001", "nginx"}, {"nginx-1.2.3-beta.0", "nginx"}, {"nginx-1.2.3-rc1.extra-2.12", "nginx"}, {"nginx-upstream-healthcheck-1.2.3.0", "nginx-upstream-healthcheck"}, {"httpd-1.2.3.0", "apache-httpd"},
			{"unknown-1.2.3.0", ""}, {"apache-httpd-1.2.3.0", ""}, {"nginx", ""}, {"nginx-1.2.4.0", ""}, {"nginx-1.2.3", ""}, {"nginx-1.2.3.", ""}, {"nginx-1.2.3.-1", ""}, {"nginx-1.2.3.x", ""}, {"nginx-1.2.3-.0", ""}, {"nginx-1.2.3-beta..rc.0", ""}, {"nginx-1.2.3-beta_1.0", ""},
		} {
			t.Run(test.tag, func(t *testing.T) {
				if err := os.WriteFile(output, []byte(sentinel), 0600); err != nil {
					t.Fatal(err)
				}
				status, _, stderr := run("release-resolve", []string{test.tag}, &output)
				want := sentinel
				if test.target != "" {
					want += "target=" + test.target + "\n"
				}
				data, err := os.ReadFile(output)
				if err != nil {
					t.Fatal(err)
				}
				if (status == 0) != (test.target != "") || string(data) != want {
					t.Fatalf("status=%d output=%q stderr=%q; want output=%q", status, data, stderr, want)
				}
			})
		}
		if err := os.WriteFile(output, []byte(sentinel), 0600); err != nil {
			t.Fatal(err)
		}
		targets["duplicate"] = target
		writeMetadata(targets)
		status, _, _ := run("release-resolve", []string{"nginx-1.2.3.0"}, &output)
		data, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		if status == 0 || string(data) != sentinel {
			t.Fatalf("ambiguous tag: status=%d output=%q", status, data)
		}
		delete(targets, "duplicate")
		writeMetadata(targets)
		empty := ""
		for _, binding := range []*string{nil, &empty} {
			status, stdout, _ := run("release-resolve", []string{"nginx-1.2.3.0"}, binding)
			data, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			if status == 0 || strings.Contains(stdout, "target=") || string(data) != sentinel {
				t.Fatalf("missing output binding: status=%d stdout=%q output=%q", status, stdout, data)
			}
		}
	})

	t.Run("verify and package", func(t *testing.T) {
		files := filepath.Join(directory, ".out", "fixture")
		if err := os.MkdirAll(filepath.Join(files, "sbin"), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(files, "sbin", "fixture"), []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(files, "conf"), 0700); err != nil {
			t.Fatal(err)
		}
		for _, test := range []struct {
			name   string
			target map[string]any
			status int
		}{
			{"file and directory", map[string]any{"release_files": []string{"sbin/fixture", "conf"}}, 0},
			{"missing first", map[string]any{"release_files": []string{"absent", "conf"}}, 1},
			{"missing second", map[string]any{"release_files": []string{"sbin/fixture", "absent"}}, 1},
			{"empty list", map[string]any{"release_files": []string{}}, 1},
			{"empty path", map[string]any{"release_files": []string{""}}, 1},
			{"missing list", map[string]any{}, 1},
		} {
			t.Run(test.name, func(t *testing.T) {
				writeMetadata(map[string]any{"fixture": test.target})
				status, _, stderr := run("release-verify", []string{"fixture"}, nil)
				if status != test.status {
					t.Fatalf("status=%d want=%d stderr=%q", status, test.status, stderr)
				}
			})
		}
		writeMetadata(map[string]any{"other": map[string]any{"release_files": []string{"sbin/fixture"}}})
		if status, _, _ := run("release-verify", []string{"fixture"}, nil); status == 0 {
			t.Fatal("absent target succeeded")
		}
		writeMetadata(map[string]any{"fixture": map[string]any{"release_files": []string{"sbin/fixture", "conf"}}})
		if status, _, stderr := run("package-release", []string{"fixture", "fixture-release"}, nil); status != 0 {
			t.Fatalf("package status=%d stderr=%q", status, stderr)
		}
		first, err := os.ReadFile(filepath.Join(directory, "fixture-release.tar.gz"))
		if err != nil {
			t.Fatal(err)
		}
		if status, _, stderr := run("package-release", []string{"fixture", "fixture-release"}, nil); status != 0 {
			t.Fatalf("repeat package status=%d stderr=%q", status, stderr)
		}
		second, err := os.ReadFile(filepath.Join(directory, "fixture-release.tar.gz"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(first, second) {
			t.Fatal("packaging unchanged output is nondeterministic")
		}
		archive, err := os.Open(filepath.Join(directory, "fixture-release.tar.gz"))
		if err != nil {
			t.Fatal(err)
		}
		defer archive.Close()
		compressed, err := gzip.NewReader(archive)
		if err != nil {
			t.Fatal(err)
		}
		defer compressed.Close()
		reader := tar.NewReader(compressed)
		var members []string
		for {
			header, err := reader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			members = append(members, header.Name)
		}
		if !reflect.DeepEqual(members, []string{"sbin/fixture", "conf/"}) {
			t.Fatalf("members=%v", members)
		}
	})

	t.Run("publish absent package", func(t *testing.T) {
		status, _, stderr := run("release-publish", []string{"absent"}, nil)
		if status != 1 || !strings.Contains(stderr, "not found") {
			t.Fatalf("status=%d stderr=%q", status, stderr)
		}
	})
}

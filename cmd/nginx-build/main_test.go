package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type invocation struct {
	Name string
	Args []string
	Dir  string
	Env  string
}

func TestMain(m *testing.M) {
	if os.Getenv("STATIC_BUILDS_FIXTURE") == "1" {
		name := filepath.Base(os.Args[0])
		dir, err := os.Getwd()
		if err != nil {
			os.Exit(99)
		}
		file, err := os.OpenFile(os.Getenv("FIXTURE_LOG"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			os.Exit(99)
		}
		if err := json.NewEncoder(file).Encode(invocation{name, os.Args[1:], dir, os.Getenv("FIXTURE_ENV")}); err != nil {
			os.Exit(99)
		}
		if err := file.Close(); err != nil {
			os.Exit(99)
		}
		key := name
		if len(os.Args) > 1 {
			key += ":" + os.Args[1]
		}
		if key == os.Getenv("FIXTURE_FAIL") {
			os.Exit(23)
		}
		if name == "nproc" {
			fmt.Println("4")
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func readInvocations(t *testing.T, path string) []invocation {
	t.Helper()
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	var result []invocation
	for decoder.More() {
		var entry invocation
		if err := decoder.Decode(&entry); err != nil {
			t.Fatal(err)
		}
		result = append(result, entry)
	}
	return result
}

func TestCommandScripts(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "bin")
	prefix := filepath.Join(root, "target")
	for _, dir := range []string{bin, filepath.Join(prefix, "bin"), filepath.Join(prefix, "sbin")} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{"nproc", "make", "cmake"} {
		if err := os.Symlink(executable, filepath.Join(bin, path)); err != nil {
			t.Fatal(err)
		}
	}
	work := filepath.Join(root, "work")
	if err := os.Mkdir(work, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(executable, filepath.Join(work, "configure")); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(root, "log")
	env := append(os.Environ(), "PATH="+bin, "STATIC_BUILDS_FIXTURE=1", "FIXTURE_LOG="+log,
		"FIXTURE_ENV=present", "TARGET_PREFIX="+prefix, "QUICKJS_VERSION=2024-01-13")
	commands := map[string]string{}
	for _, name := range []string{"nginx-build", "nginx-build-quickjs", "nginx-configure", "haproxy-apply-otel-patches"} {
		path := filepath.Join(root, name)
		command := exec.Command("go", "build", "-o", path, "../"+name)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("build %s: %v\n%s", name, err, output)
		}
		commands[name] = path
	}
	run := func(t *testing.T, name string, options []string, args ...string) ([]invocation, int) {
		t.Helper()
		if err := os.Remove(log); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		cmd := exec.Command(commands[name], args...)
		cmd.Dir = work
		cmd.Env = append(append([]string{}, env...), options...)
		output, err := cmd.CombinedOutput()
		status := 0
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				status = exit.ExitCode()
			} else {
				t.Fatalf("run %s: %v\n%s", name, err, output)
			}
		}
		return readInvocations(t, log), status
	}
	t.Run("nginx-build", func(t *testing.T) {
		got, status := run(t, "nginx-build", nil)
		want := []invocation{{"nproc", []string{}, work, "present"}, {"make", []string{"-j4"}, work, "present"}, {"make", []string{"install"}, work, "present"}}
		if status != 0 || !reflect.DeepEqual(got, want) {
			t.Fatalf("status %d: got %#v, want %#v", status, got, want)
		}
		for _, fail := range []string{"nproc", "make:-j4", "make:install"} {
			got, status := run(t, "nginx-build", []string{"FIXTURE_FAIL=" + fail})
			if status != 23 || len(got) != map[string]int{"nproc": 1, "make:-j4": 2, "make:install": 3}[fail] {
				t.Errorf("failure %s: status %d, calls %#v", fail, status, got)
			}
		}
	})
	t.Run("nginx-build-quickjs", func(t *testing.T) {
		got, status := run(t, "nginx-build-quickjs", nil)
		want := []invocation{
			{"cmake", []string{"-S", "/build/quickjs-2024-01-13", "-B", "/build/quickjs-build", "-DBUILD_SHARED_LIBS=OFF", "-DCMAKE_POSITION_INDEPENDENT_CODE=ON", "-DCMAKE_BUILD_TYPE=Release"}, work, "present"},
			{"nproc", []string{}, work, "present"},
			{"cmake", []string{"--build", "/build/quickjs-build", "-j4"}, work, "present"},
			{"cmake", []string{"--install", "/build/quickjs-build"}, work, "present"},
		}
		if status != 0 || !reflect.DeepEqual(got, want) {
			t.Fatalf("status %d: got %#v, want %#v", status, got, want)
		}
		for index, fail := range []string{"cmake:-S", "nproc", "cmake:--build", "cmake:--install"} {
			got, status := run(t, "nginx-build-quickjs", []string{"FIXTURE_FAIL=" + fail})
			if status != 23 || len(got) != index+1 {
				t.Errorf("failure %s: status %d, calls %#v", fail, status, got)
			}
		}
	})
	t.Run("nginx-configure", func(t *testing.T) {
		got, status := run(t, "nginx-configure", nil, "--add-module=/module one", "--with-example")
		if status != 0 || len(got) != 1 || got[0].Name != "configure" || got[0].Dir != work || got[0].Env != "present" {
			t.Fatalf("status %d: calls %#v", status, got)
		}
		args := got[0].Args
		if len(args) != 28 || args[0] != "--prefix="+prefix || !reflect.DeepEqual(args[23:25], []string{"--add-module=/module one", "--with-example"}) || !strings.HasPrefix(args[25], "--with-cc-opt=-O2 ") || !strings.HasPrefix(args[26], "--with-ld-opt=-L/usr/local/lib ") || args[27] != "--with-libatomic" {
			t.Errorf("configure arguments: %#v", args)
		}
		_, status = run(t, "nginx-configure", []string{"FIXTURE_FAIL=configure:--prefix=" + prefix})
		if status != 23 {
			t.Errorf("configure failure status %d", status)
		}
	})
}

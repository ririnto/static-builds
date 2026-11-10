package main

import (
	"bytes"
	"compress/gzip"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestDeterministicPackaging(t *testing.T) {
	directory := t.TempDir()
	if err := os.MkdirAll(filepath.Join(directory, ".out", "fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, ".out", "fixture", "binary"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	metadata := filepath.Join(directory, "metadata.json")
	if err := os.WriteFile(metadata, []byte(`{"fixture":{"release_files":["binary"]}}`), 0600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "package-release")
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, output)
	}
	var first []byte
	for range 2 {
		command := exec.Command(binary, "fixture", "release")
		command.Dir = directory
		command.Env = append(os.Environ(), "METADATA_PATH_OVERRIDE="+metadata)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("%v\n%s", err, output)
		}
		data, err := os.ReadFile(filepath.Join(directory, "release.tar.gz"))
		if err != nil {
			t.Fatal(err)
		}
		compressed, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		if !compressed.ModTime.IsZero() {
			t.Fatal("gzip timestamp is not zero")
		}
		compressed.Close()
		if first != nil && !bytes.Equal(first, data) {
			t.Fatal("packaging unchanged output is nondeterministic")
		}
		first = data
		time.Sleep(time.Second)
	}
}

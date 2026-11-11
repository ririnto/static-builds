package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCachedSource(t *testing.T) {

	path := filepath.Join(t.TempDir(), "source.tar.gz")
	if err := os.WriteFile(path, []byte("cached source"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := download(context.Background(), "https://example.invalid/source.tar.gz", path); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "cached source" {
		t.Fatalf("cached source changed: %q, %v", data, err)
	}
}

func TestBranchFailurePreservesSourceAndRemovesLock(t *testing.T) {

	path := filepath.Join(t.TempDir(), "branch.tar.gz")
	if err := os.WriteFile(path, []byte("previous source"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := download(context.Background(), "http://example.invalid/archive/refs/heads/main.tar.gz", path); err == nil {
		t.Fatal("accepted an insecure download URL")
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "previous source" {
		t.Fatalf("previous source changed: %q, %v", data, err)
	}
	if _, err := os.Stat(path + ".lock"); !os.IsNotExist(err) {
		t.Fatalf("branch lock remains: %v", err)
	}
}

func TestTargetRequired(t *testing.T) {

	if err := run(context.Background(), nil); err == nil {
		t.Fatal("accepted a missing target")
	}
}

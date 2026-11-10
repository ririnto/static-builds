package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func run() error {
	patches, err := filepath.Glob("/build/opentelemetry-c-wrapper/scripts/build/000*-opentelemetry-cpp-*.patch")
	if err != nil {
		return err
	}
	if len(patches) == 0 {
		return fmt.Errorf("no OpenTelemetry patches found")
	}
	for _, path := range patches {
		patch, err := os.Open(path)
		if err != nil {
			return err
		}
		command := exec.Command("patch", "-p1")
		command.Stdin, command.Stdout, command.Stderr = patch, os.Stdout, os.Stderr
		err = command.Run()
		closeErr := patch.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		if status, ok := err.(*exec.ExitError); ok {
			os.Exit(status.ExitCode())
		}
		os.Exit(1)
	}
}

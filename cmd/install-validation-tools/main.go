package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func run() error {
	architecture := ""
	switch runtime.GOARCH {
	case "amd64":
		architecture = "x86_64"
	case "arm64":
		architecture = "aarch64"
	default:
		return fmt.Errorf("Unsupported architecture: %s", runtime.GOARCH)
	}
	temporary, err := os.MkdirTemp("", "validation-tools.")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	commands := [][]string{{"apk", "add", "--no-cache", "ca-certificates", "git", "curl"}}
	for _, tool := range []struct{ name, url, format string }{
		{"rumdl", "https://github.com/rvben/rumdl/releases/download/v0.3.1/rumdl-v0.3.1-" + architecture + "-unknown-linux-musl.tar.gz", "--gzip"},
		{"lychee", "https://github.com/lycheeverse/lychee/releases/download/lychee-v0.24.2/lychee-" + architecture + "-unknown-linux-musl.tar.gz", "--gzip"},
	} {
		directory := filepath.Join(temporary, tool.name)
		if err := os.Mkdir(directory, 0700); err != nil {
			return err
		}
		commands = append(commands, []string{"curl", "--fail", "--location", "--silent", "--show-error", tool.url, "--output", directory + ".archive"}, []string{"tar", "--extract", tool.format, "--file", directory + ".archive", "--directory", directory})
	}
	for _, arguments := range commands {
		command := exec.Command(arguments[0], arguments[1:]...)
		command.Env = os.Environ()
		command.Stdout, command.Stderr = os.Stdout, os.Stderr
		if err := command.Run(); err != nil {
			return err
		}
	}
	for _, name := range []string{"rumdl", "lychee"} {
		var source string
		err := filepath.WalkDir(filepath.Join(temporary, name), func(path string, entry os.DirEntry, err error) error {
			if err == nil && !entry.IsDir() && entry.Name() == name && source == "" {
				source = path
			}
			return err
		})
		if err != nil {
			return err
		}
		command := exec.Command("install", "-m", "0755", source, "/usr/local/bin/"+name)
		command.Stdout, command.Stderr = os.Stdout, os.Stderr
		if err := command.Run(); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		status := 1
		if exit, ok := err.(*exec.ExitError); ok {
			status = exit.ExitCode()
		}
		os.Exit(status)
	}
}

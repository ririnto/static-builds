package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ririnto/static-builds/internal/buildconfig"
	"github.com/ririnto/static-builds/internal/metadata"
)

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: build-rootless <target>")
	}
	target := args[0]
	if !buildconfig.ValidTarget(target) {
		return fmt.Errorf("Invalid target %s", target)
	}
	config, err := buildconfig.Load(target)
	if err != nil {
		return err
	}
	if info, err := os.Stat(filepath.Join(config.Root, target, "Dockerfile")); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("Dockerfile not found in %s", target)
	}
	if os.Geteuid() == 0 {
		return fmt.Errorf("Rootless BuildKit must run as a non-root user")
	}
	data, err := metadata.LoadDefault()
	if err != nil {
		return err
	}
	entry, err := data.Target(target)
	if err != nil {
		return err
	}
	environment, err := entry.Environment()
	if err != nil {
		return err
	}
	path, err := exec.LookPath("buildctl-daemonless.sh")
	if err != nil {
		return fmt.Errorf("buildctl-daemonless.sh not found in PATH: %w", err)
	}
	fmt.Println(path)
	contained := false
	for _, prefix := range []string{config.Root + "/.out/" + target, ".out/" + target, "./.out/" + target} {
		if config.Output == prefix || strings.HasPrefix(config.Output, prefix+"/") {
			contained = true
		}
	}
	if !contained || strings.Contains("/"+config.Output+"/", "/../") {
		return fmt.Errorf("Build output must be under .out/%s", target)
	}
	if err := os.MkdirAll(config.Output, 0755); err != nil {
		return err
	}
	if config.CacheBackend != "local" {
		return fmt.Errorf("Invalid BUILDKIT_CACHE_BACKEND %s\nAllowed values: local", config.CacheBackend)
	}
	config.Cache = filepath.Join(config.Root, ".cache", target)
	if err := os.MkdirAll(config.Cache, 0755); err != nil {
		return err
	}
	cacheFrom, cacheTo, err := config.Caches()
	if err != nil {
		return err
	}
	if config.Network != "host" && config.Network != "default" {
		return fmt.Errorf("Rootless BuildKit only supports host/default network semantics in this repository")
	}
	arguments := []string{"build", "--frontend", "dockerfile.v0", "--local", "context=" + config.Root, "--local", "dockerfile=" + config.Root, "--opt", "filename=" + target + "/Dockerfile", "--opt", "platform=" + config.Platform, "--output", "type=local,dest=" + config.Output, "--import-cache", cacheFrom, "--export-cache", cacheTo}
	for _, argument := range buildconfig.Arguments(environment) {
		arguments = append(arguments, "--opt", "build-arg:"+argument)
	}
	command := exec.Command(path, arguments...)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	return command.Run()
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		status := 1
		if exit, ok := err.(*exec.ExitError); ok {
			status = exit.ExitCode()
		}
		os.Exit(status)
	}
}

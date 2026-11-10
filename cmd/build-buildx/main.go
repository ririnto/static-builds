package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ririnto/static-builds/internal/buildconfig"
	"github.com/ririnto/static-builds/internal/metadata"
)

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: build-buildx <target> [buildx args...]")
	}
	target := args[0]
	if !buildconfig.ValidTarget(target) {
		return fmt.Errorf("Invalid target '%s'\nAllowed characters: letters, numbers, dot, underscore, hyphen", target)
	}
	config, err := buildconfig.Load(target)
	if err != nil {
		return err
	}
	if info, err := os.Stat(filepath.Join(config.Root, target)); err != nil || !info.IsDir() {
		return fmt.Errorf("Target directory '%s' not found", target)
	}
	if info, err := os.Stat(filepath.Join(config.Root, target, "Dockerfile")); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("Dockerfile not found in '%s'", target)
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
	cacheFrom, cacheTo, err := config.Caches()
	if err != nil {
		return err
	}
	if config.CacheBackend == "local" {
		if err := os.MkdirAll(config.Cache, 0755); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(config.Output, 0755); err != nil {
		return err
	}
	arguments := []string{"buildx", "build", "--platform=" + config.Platform, "--network=" + config.Network, "--output=type=local,dest=" + config.Output, "--cache-from=" + cacheFrom, "--cache-to=" + cacheTo}
	buildArgs := buildconfig.Arguments(environment)
	for index := len(buildArgs) - 1; index >= 0; index-- {
		arguments = append(arguments, "--build-arg="+buildArgs[index])
	}
	arguments = append(arguments, args[1:]...)
	arguments = append(arguments, "--file="+filepath.Join(config.Root, target, "Dockerfile"), config.Root)
	command := exec.Command("docker", arguments...)
	command.Dir = config.Root
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

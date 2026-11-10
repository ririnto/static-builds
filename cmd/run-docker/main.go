package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ririnto/static-builds/internal/metadata"
)

type statusError int

func (status statusError) Error() string { return fmt.Sprintf("suite exited with status %d", status) }

func docker(ctx context.Context, capture bool, arguments ...string) (string, error) {
	if len(arguments) > 0 && (arguments[0] == "container" || arguments[0] == "image" || arguments[0] == "rm") {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}
	command := exec.CommandContext(ctx, "docker", arguments...)
	command.Stderr = os.Stderr
	command.WaitDelay = 2e9
	if capture {
		output, err := command.Output()
		return strings.TrimSpace(string(output)), err
	}
	command.Stdout = os.Stdout
	return "", command.Run()
}

func run(ctx context.Context, arguments []string) (err error) {
	platform := "linux/amd64"
	binary := ""
	if len(arguments) == 3 && arguments[0] == "--platform" {
		platform, binary = arguments[1], arguments[2]
	} else if len(arguments) == 1 {
		binary = arguments[0]
	} else {
		return statusError(2)
	}
	path, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("required command is missing: docker: %w", statusError(2))
	}
	fmt.Println(path)
	if _, err := docker(ctx, false, "buildx", "version"); err != nil {
		return fmt.Errorf("required Docker Buildx plugin is missing: %w", statusError(2))
	}
	info, err := os.Stat(binary)
	if err != nil || info.IsDir() || info.Mode()&0111 == 0 {
		return fmt.Errorf("NGINX binary is not executable: %s: %w", binary, statusError(2))
	}
	binary, err = filepath.Abs(binary)
	if err != nil {
		return err
	}
	binary, err = filepath.EvalSymlinks(binary)
	if err != nil {
		return err
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	data, err := metadata.LoadDefault()
	if err != nil {
		return err
	}
	target, err := data.Target("nginx-upstream-healthcheck")
	if err != nil {
		return err
	}
	version := ""
	for _, entry := range target.Env {
		if entry.Key == "UBI10_MINIMAL_VERSION" {
			version = entry.Value
		}
	}
	if version == "" {
		return fmt.Errorf("target metadata does not define UBI10_MINIMAL_VERSION: %w", statusError(2))
	}
	id := fmt.Sprintf("%x", rand.Text())
	image := "static-builds-healthcheck-suite:" + id
	container := "static-builds-healthcheck-suite-" + id
	label := "static-builds.healthcheck.run=" + id
	imageAttempted, containerAttempted := false, false
	defer func() {
		cleanup := context.Background()
		preserve := func(cleanupErr error) {
			if cleanupErr != nil {
				fmt.Fprintln(os.Stderr, "Cleanup:", cleanupErr)
				if err == nil {
					err = cleanupErr
				}
			}
		}
		if containerAttempted {
			owned, cleanupErr := docker(cleanup, true, "container", "ls", "--all", "--quiet", "--filter", "name=^/"+container+"$", "--filter", "label="+label)
			preserve(cleanupErr)
			if cleanupErr == nil && owned != "" {
				_, cleanupErr = docker(cleanup, false, "rm", "-f", owned)
				preserve(cleanupErr)
			}
		}
		if imageAttempted {
			owned, cleanupErr := docker(cleanup, true, "image", "ls", "--quiet", image)
			preserve(cleanupErr)
			if cleanupErr == nil && owned != "" {
				owner, cleanupErr := docker(cleanup, true, "image", "inspect", "--format", `{{ index .Config.Labels "static-builds.healthcheck.run" }}`, image)
				preserve(cleanupErr)
				if cleanupErr == nil {
					if owner == id {
						_, cleanupErr = docker(cleanup, false, "image", "rm", image)
						preserve(cleanupErr)
					} else {
						preserve(fmt.Errorf("refusing to remove an image without this runner ownership label: %s", image))
					}
				}
			}
		}
	}()
	imageAttempted = true
	if _, err = docker(ctx, false, "buildx", "build", "--load", "--platform", platform, "--build-arg", "FUNCTIONAL_BASE_IMAGE=registry.access.redhat.com/ubi10/ubi-minimal:"+version, "--tag", image, "--label", label, "--file", filepath.Join(root, "proxy/tests/Dockerfile"), root); err != nil {
		return err
	}
	containerAttempted = true
	if _, err = docker(ctx, false, "create", "--platform", platform, "--name", container, "--label", label, "--cpus", "4", "--memory", "8g", "--mount", "type=bind,source="+binary+",target=/input/nginx,readonly", image, "/input/nginx"); err != nil {
		return err
	}
	if _, err = docker(ctx, false, "start", container); err != nil {
		return err
	}
	output, err := docker(ctx, true, "wait", container)
	if err != nil {
		return err
	}
	status, err := strconv.Atoi(output)
	if err != nil {
		return err
	}
	if _, err = docker(ctx, false, "logs", container); err != nil {
		return err
	}
	if status != 0 {
		return statusError(status)
	}
	return nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	received := make(chan os.Signal, 1)
	signal.Notify(received, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(received)
	signalStatus := make(chan int, 1)
	go func() {
		select {
		case value := <-received:
			signalStatus <- 128 + int(value.(syscall.Signal))
			cancel()
		case <-ctx.Done():
		}
	}()
	err := run(ctx, os.Args[1:])
	cancel()
	select {
	case status := <-signalStatus:
		os.Exit(status)
	default:
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		var status statusError
		var exit *exec.ExitError
		if errors.As(err, &status) {
			os.Exit(int(status))
		}
		if errors.As(err, &exit) {
			os.Exit(exit.ExitCode())
		}
		os.Exit(1)
	}
}

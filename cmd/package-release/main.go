package main

import (
	"compress/gzip"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ririnto/static-builds/internal/metadata"
	"github.com/ririnto/static-builds/internal/release"
)

func run(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: package-release <target> <package-name>")
	}
	data, err := metadata.LoadDefault()
	if err != nil {
		return err
	}
	target, err := data.Target(args[0])
	if err != nil {
		return err
	}
	files, err := release.Files(target)
	if err != nil {
		return err
	}
	archive := args[1] + ".tar.gz"
	if err := os.Remove(archive); err != nil && !os.IsNotExist(err) {
		return err
	}
	directory, err := filepath.Abs(filepath.Join(".out", args[0]))
	if err != nil {
		return err
	}
	if err := release.Verify(directory, files); err != nil {
		return err
	}
	output, err := os.Create(archive)
	if err != nil {
		return err
	}
	compressed := gzip.NewWriter(output)
	command := exec.Command("tar", append([]string{"-cvf", "-", "-C", directory}, files...)...)
	command.Env = append(os.Environ(), "COPYFILE_DISABLE=1")
	command.Stdout, command.Stderr = compressed, os.Stderr
	err = command.Run()
	compressErr := compressed.Close()
	closeErr := output.Close()
	if err != nil {
		return err
	}
	if compressErr != nil {
		return compressErr
	}
	return closeErr
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

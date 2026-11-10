package main

import (
	"fmt"
	"github.com/ririnto/static-builds/internal/metadata"
	"github.com/ririnto/static-builds/internal/release"
	"os"
)

func run(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: release-guard <target> <tag>")
	}
	data, err := metadata.LoadDefault()
	if err != nil {
		return err
	}
	target, err := data.Target(args[0])
	if err != nil {
		return err
	}
	version, err := release.ValidateTag(target, args[1])
	if err != nil {
		return err
	}
	fmt.Printf("Tag validation passed: %s (version %s)\n", args[1], version)
	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

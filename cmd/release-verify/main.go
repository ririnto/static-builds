package main

import (
	"fmt"
	"github.com/ririnto/static-builds/internal/metadata"
	"github.com/ririnto/static-builds/internal/release"
	"os"
)

func run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: release-verify <target>")
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
	return release.Verify(".out/"+args[0], files)
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

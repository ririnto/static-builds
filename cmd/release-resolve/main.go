package main

import (
	"fmt"
	"github.com/ririnto/static-builds/internal/metadata"
	"github.com/ririnto/static-builds/internal/release"
	"os"
)

func run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: release-resolve <tag>")
	}
	data, err := metadata.LoadDefault()
	if err != nil {
		return err
	}
	target, err := data.TargetFromTag(args[0])
	if err != nil {
		return err
	}
	version, err := release.ValidateTag(target, args[0])
	if err != nil {
		return err
	}
	fmt.Printf("Tag validation passed: %s (version %s)\n", args[0], version)
	output, err := os.OpenFile(os.Getenv("GITHUB_OUTPUT"), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "target=%s\n", target.Name)
	closeErr := output.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

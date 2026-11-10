package main

import (
	"fmt"
	"os"

	"github.com/ririnto/static-builds/internal/metadata"
)

func run(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: metadata <command> <argument>")
	}
	switch args[0] {
	case "get-env", "get-official-version", "get-tag-prefix", "get-target-from-tag", "get-release-files", "get-downloads":
	default:
		return fmt.Errorf("usage: metadata <command> <argument>")
	}
	data, err := metadata.LoadDefault()
	if err != nil {
		return err
	}
	if args[0] == "get-target-from-tag" {
		target, err := data.TargetFromTag(args[1])
		if err != nil {
			return err
		}
		fmt.Println(target.Name)
		return nil
	}
	target, err := data.Target(args[1])
	if err != nil {
		return err
	}
	switch args[0] {
	case "get-env":
		entries, err := target.Environment()
		if err != nil {
			return err
		}
		for _, entry := range entries {
			fmt.Printf("%s=%s\n", entry.Key, entry.Value)
		}
	case "get-official-version":
		version, err := target.OfficialVersion()
		if err != nil {
			return err
		}
		fmt.Println(version)
	case "get-tag-prefix":
		prefix, err := target.Prefix()
		if err != nil {
			return err
		}
		fmt.Println(prefix)
	case "get-release-files":
		files, err := target.Files()
		if err != nil {
			return err
		}
		for _, file := range files {
			fmt.Println(file)
		}
	case "get-downloads":
		downloads, err := target.RenderDownloads()
		if err != nil {
			return err
		}
		for _, download := range downloads {
			fmt.Printf("%s\t%s\n", download.URL, download.Name)
		}
	}
	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

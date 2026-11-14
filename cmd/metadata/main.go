// Command metadata prints build and release values from target metadata.
package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/alecthomas/kong"

	"github.com/ririnto/static-builds/internal/metadata"
)

func run(args []string) error {
	var cli struct {
		Command  string `arg:"" enum:"get-env,get-official-version,get-tag-prefix,get-target-from-tag,get-release-files,get-downloads" help:"Metadata query to execute."`
		Argument string `arg:"" help:"Target name, or release tag for get-target-from-tag."`
	}
	parser, err := kong.New(&cli, kong.Name("metadata"))
	if err != nil {
		return err
	}
	if _, err := parser.Parse(args); err != nil {
		return err
	}
	data, err := metadata.LoadDefault()
	if err != nil {
		return err
	}
	if cli.Command == "get-target-from-tag" {
		target, err := data.TargetFromTag(cli.Argument)
		if err != nil {
			return err
		}
		fmt.Println(target.Name)
		return nil
	}
	target, err := data.Target(cli.Argument)
	if err != nil {
		return err
	}
	switch cli.Command {
	case "get-env":
		entries, err := target.Environment()
		if err != nil {
			return err
		}
		for entry := range slices.Values(entries) {
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
		for file := range slices.Values(files) {
			fmt.Println(file)
		}
	case "get-downloads":
		downloads, err := target.RenderDownloads()
		if err != nil {
			return err
		}
		for download := range slices.Values(downloads) {
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

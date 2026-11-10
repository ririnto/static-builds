package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	command := exec.Command("apk", "add", "--no-cache", "ca-certificates", "git", "curl")
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	err := command.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		status := 1
		if exit, ok := err.(*exec.ExitError); ok {
			status = exit.ExitCode()
		}
		os.Exit(status)
	}
}

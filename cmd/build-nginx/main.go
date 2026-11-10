package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func execute(name string, arguments ...string) error {
	command := exec.Command(name, arguments...)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	return command.Run()
}

func run() error {
	parallelism, err := exec.Command("nproc").Output()
	if err != nil {
		return err
	}
	for _, arguments := range [][]string{{"-j" + strings.TrimSpace(string(parallelism))}, {"install"}} {
		if err := execute("make", arguments...); err != nil {
			return err
		}
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		if status, ok := err.(*exec.ExitError); ok {
			os.Exit(status.ExitCode())
		}
		os.Exit(1)
	}
}

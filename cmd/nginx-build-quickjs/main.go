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
	if err := execute("cmake", "-S", "/build/quickjs-"+os.Getenv("QUICKJS_VERSION"), "-B", "/build/quickjs-build", "-DBUILD_SHARED_LIBS=OFF", "-DCMAKE_POSITION_INDEPENDENT_CODE=ON", "-DCMAKE_BUILD_TYPE=Release"); err != nil {
		return err
	}
	parallelism, err := exec.Command("nproc").Output()
	if err != nil {
		return err
	}
	if err := execute("cmake", "--build", "/build/quickjs-build", "-j"+strings.TrimSpace(string(parallelism))); err != nil {
		return err
	}
	return execute("cmake", "--install", "/build/quickjs-build")
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

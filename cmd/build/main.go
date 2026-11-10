package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

func main() {
	adapter := ""
	switch runtime.GOOS {
	case "linux":
		adapter = "build-daemonless"
	case "darwin":
		adapter = "build-buildx"
	default:
		fmt.Fprintln(os.Stderr, "Error: Unsupported build host:", runtime.GOOS)
		os.Exit(1)
	}
	goCommand, err := exec.LookPath("go")
	if err == nil {
		err = syscall.Exec(goCommand, append([]string{"go", "run", "./cmd/" + adapter}, os.Args[1:]...), os.Environ())
	}
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(1)
}

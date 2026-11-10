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
	binary := os.Getenv("TARGET_PREFIX") + "/coredns"
	for _, inspection := range []struct{ title, flag, phrases string }{{"ELF Header", "-h", "Type:|Machine:"}, {"Program Headers", "-l", "INTERP|DYNAMIC"}, {"Dynamic Section", "-d", ""}} {
		fmt.Println("=== " + inspection.title + " ===")
		command := exec.Command("readelf", inspection.flag, binary)
		command.Stderr = os.Stderr
		output, _ := command.Output()
		matched := false
		for _, line := range strings.Split(strings.TrimSuffix(string(output), "\n"), "\n") {
			if inspection.phrases == "" || strings.Contains(line, strings.Split(inspection.phrases, "|")[0]) || strings.Contains(line, strings.Split(inspection.phrases, "|")[1]) {
				fmt.Println(line)
				matched = true
			}
		}
		if inspection.flag == "-h" && !matched {
			return fmt.Errorf("ELF header has no type or machine")
		}
	}
	fileOutput, _ := exec.Command("file", binary).CombinedOutput()
	lddOutput, _ := exec.Command("ldd", binary).CombinedOutput()
	fmt.Println("=== file output ===")
	fmt.Println(strings.TrimRight(string(fileOutput), "\n"))
	fmt.Println("=== ldd output ===")
	fmt.Println(strings.TrimRight(string(lddOutput), "\n"))
	if !strings.Contains(string(fileOutput), "statically linked") && !strings.Contains(string(fileOutput), "static-pie linked") {
		return fmt.Errorf("file(1) did not report a static binary")
	}
	static := false
	for _, phrase := range []string{"statically linked", "static-pie linked", "not a dynamic executable", "not a valid dynamic program"} {
		static = static || strings.Contains(strings.ToLower(string(lddOutput)), phrase)
	}
	if !static {
		return fmt.Errorf("dynamic libraries found, expected a static build")
	}
	if err := execute("strace", binary, "-version"); err != nil {
		return err
	}
	fmt.Println("=== Installed Plugins ===")
	if err := execute(binary, "-plugins"); err != nil {
		return err
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

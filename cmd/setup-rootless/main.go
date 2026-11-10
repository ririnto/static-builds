package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
)

func run() error {
	if os.Geteuid() == 0 {
		return fmt.Errorf("Run rootless provisioning as the non-root build user.")
	}
	for _, arguments := range [][]string{{"apt-get", "update"}, {"apt-get", "install", "--yes", "--no-install-recommends", "rootlesskit", "runc", "uidmap"}} {
		command := exec.Command("sudo", arguments...)
		command.Stdout, command.Stderr = os.Stdout, os.Stderr
		if err := command.Run(); err != nil {
			return err
		}
	}
	for _, name := range []string{"rootlesskit", "runc", "newuidmap", "newgidmap", "setsid", "timeout"} {
		path, err := exec.LookPath(name)
		if err != nil {
			return err
		}
		fmt.Println(path)
	}
	current, err := user.Current()
	if err != nil {
		return err
	}
	for _, path := range []string{"/etc/subuid", "/etc/subgid"} {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		found := false
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			fields := strings.Split(scanner.Text(), ":")
			if len(fields) >= 3 && (fields[0] == current.Username || fields[0] == current.Uid) {
				start, startErr := strconv.ParseUint(fields[1], 10, 64)
				count, countErr := strconv.ParseUint(fields[2], 10, 64)
				if startErr == nil && countErr == nil && start > 0 && count >= 65536 {
					found = true
				}
			}
		}
		scanErr := scanner.Err()
		closeErr := file.Close()
		if scanErr != nil {
			return scanErr
		}
		if closeErr != nil {
			return closeErr
		}
		if !found {
			return fmt.Errorf("Provision a subordinate ID range of at least 65536 for %s in %s.", current.Username, path)
		}
	}
	program := `FNR == 1 { root = 0; subordinate = 0 }
$1 == 0 && $3 == 1 { root = 1 }
$1 > 0 && $3 >= 65536 { subordinate = 1 }
FNR == 2 && root && subordinate { valid++ }
END { exit valid != 2 }`
	command := exec.Command("timeout", "--kill-after=5", "30", "rootlesskit", "--pidns", "--reaper=true", "awk", program, "/proc/self/uid_map", "/proc/self/gid_map")
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("Rootless namespace preflight failed; provision mappings and permitted user namespaces without weakening AppArmor, seccomp, or host policy: %w", err)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		status := 1
		if exit, ok := err.(*exec.ExitError); ok {
			status = exit.ExitCode()
		}
		os.Exit(status)
	}
}

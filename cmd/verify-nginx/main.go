package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func execute(ctx context.Context, command *exec.Cmd) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		_ = syscall.Kill(-command.Process.Pid, syscall.SIGTERM)
		timer := time.NewTimer(time.Second)
		defer timer.Stop()
		select {
		case <-done:
			// The leader may have exited while descendants still share its group.
			_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		case <-timer.C:
			_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
			<-done
		}
		return ctx.Err()
	}
}

func run(ctx context.Context) error {
	binary := os.Getenv("TARGET_PREFIX") + "/sbin/nginx"
	for _, inspection := range []struct{ title, flag, phrases string }{{"ELF Header", "-h", "Type:|Machine:"}, {"Program Headers", "-l", "INTERP|DYNAMIC"}, {"Dynamic Section", "-d", ""}} {
		fmt.Println("=== " + inspection.title + " ===")
		command := exec.Command("readelf", inspection.flag, binary)
		var output bytes.Buffer
		command.Stdout, command.Stderr = &output, os.Stderr
		_ = execute(ctx, command)
		if err := ctx.Err(); err != nil {
			return err
		}
		matched := false
		for _, line := range strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n") {
			if inspection.phrases == "" || strings.Contains(line, strings.Split(inspection.phrases, "|")[0]) || strings.Contains(line, strings.Split(inspection.phrases, "|")[1]) {
				fmt.Println(line)
				matched = true
			}
		}
		if inspection.flag == "-h" && !matched {
			return fmt.Errorf("ELF header has no type or machine")
		}
	}
	var fileOutput, lddOutput bytes.Buffer
	for _, inspection := range []struct {
		name   string
		output *bytes.Buffer
	}{{"file", &fileOutput}, {"ldd", &lddOutput}} {
		command := exec.Command(inspection.name, binary)
		command.Stdout, command.Stderr = inspection.output, inspection.output
		_ = execute(ctx, command)
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	fmt.Println("=== file output ===")
	fmt.Println(strings.TrimRight(fileOutput.String(), "\n"))
	fmt.Println("=== ldd output ===")
	fmt.Println(strings.TrimRight(lddOutput.String(), "\n"))
	if !strings.Contains(fileOutput.String(), "statically linked") && !strings.Contains(fileOutput.String(), "static-pie linked") {
		return fmt.Errorf("file(1) did not report a static binary")
	}
	static := false
	for _, phrase := range []string{"statically linked", "static-pie linked", "not a dynamic executable", "not a valid dynamic program"} {
		static = static || strings.Contains(strings.ToLower(lddOutput.String()), phrase)
	}
	if !static {
		return fmt.Errorf("dynamic libraries found, expected a static build")
	}
	command := exec.Command("strace", binary, "-V")
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := execute(ctx, command); err != nil {
		return err
	}
	return verifyRuntime(ctx, binary)
}

func main() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	ctx, cancel := context.WithCancelCause(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case received := <-signals:
			cancel(interrupted{received})
		case <-ctx.Done():
		}
	}()
	err := run(ctx)
	cancel(nil)
	<-done
	if stopped, ok := context.Cause(ctx).(interrupted); ok {
		fmt.Fprintln(os.Stderr, "Error:", stopped)
		os.Exit(128 + int(stopped.signal.(syscall.Signal)))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		if status, ok := err.(*exec.ExitError); ok {
			if status.ExitCode() < 0 {
				os.Exit(128 + int(status.Sys().(syscall.WaitStatus).Signal()))
			}
			os.Exit(status.ExitCode())
		}
		os.Exit(1)
	}
}

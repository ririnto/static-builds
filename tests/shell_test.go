package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestShellCheckPrefersEchoForPlainOutput(t *testing.T) {
	t.Parallel()
	checkout := t.TempDir()
	env := slices.DeleteFunc(os.Environ(), func(variable string) bool {
		return strings.HasPrefix(variable, "GIT_")
	})
	run := func(name string, args ...string) (string, error) {
		command := exec.CommandContext(t.Context(), name, args...)
		command.Dir = checkout
		command.Env = env
		output, err := command.CombinedOutput()
		return string(output), err
	}
	mustRun := func(name string, args ...string) {
		t.Helper()
		if output, err := run(name, args...); err != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, err, output)
		}
	}
	write := func(name, contents string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(checkout, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"Taskfile.yaml", ".editorconfig", "go.mod", "go.sum"} {
		data, err := os.ReadFile(filepath.Join("..", name))
		if err != nil {
			t.Fatal(err)
		}
		write(name, string(data))
	}
	mustRun("git", "init", "-q")

	const controls = `echo 'Ready'
printf '%s' 'Ready'
printf 'Ready\tset\n'
printf 'Value: %d\n' 7
printf '%s\n' "$message"
printf '%s\n' '-n'
printf '%s\n' 'a\b'
printf -- 'Ready\n'
printf 'Ready\n' >"$output"
cat <<'OUTPUT'
printf 'Ready\n'
OUTPUT
`
	write("output.sh", controls)
	mustRun("git", "add", "--", "output.sh")
	mustRun("go", "tool", "task", "shfmt", "--", "-d")

	write("output.sh", controls+`printf 'Ready\n'
printf '%s\n' "All done"
`)
	mustRun("go", "tool", "task", "shfmt", "--", "-w")
	output, err := run("go", "tool", "task", "shfmt", "--", "-d")
	if err == nil {
		t.Fatalf("expected the shell check to fail for ordinary printf lines:\n%s", output)
	}
	if strings.Count(output, "R32/prefer-echo") != 2 ||
		!strings.Contains(output, "output.sh:13: R32/prefer-echo") ||
		!strings.Contains(output, "output.sh:14: R32/prefer-echo") {
		t.Fatalf("expected diagnostics for both ordinary printf lines:\n%s", output)
	}
}

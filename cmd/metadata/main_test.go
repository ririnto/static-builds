package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMetadataCommand(t *testing.T) {
	directory := t.TempDir()
	binary := filepath.Join(directory, "metadata")
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	path := filepath.Join(directory, "metadata.json")
	write := func(content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	query := func(command, argument string, status int, expected string) {
		t.Helper()
		process := exec.Command(binary, command, argument)
		process.Env = append(os.Environ(), "METADATA_PATH_OVERRIDE="+path)
		output, err := process.CombinedOutput()
		actual := 0
		if err != nil {
			actual = process.ProcessState.ExitCode()
		}
		if actual != status || status == 0 && string(output) != expected || status != 0 && !strings.Contains(string(output), expected) {
			t.Fatalf("%s %s: status=%d output=%q expected=%d %q", command, argument, actual, output, status, expected)
		}
	}
	write(`{"fixture":{"tag_prefix":"fixture","version_env_var":"VERSION","env":{"VERSION":"1.2","BRANCH":"main"},"release_files":["sbin/fixture","conf/fixture.conf"],"downloads":[{"url":"https://example.invalid/{BRANCH}/{VERSION}","name":"{VERSION}.tgz"}],"unknown":[{"nested":[null,true,false,-1.25e3,"escaped\\value\"/"]}]},"extended":{"tag_prefix":"fixture-extra"}}`)
	for _, test := range []struct{ command, argument, expected string }{
		{"get-env", "fixture", "VERSION=1.2\nBRANCH=main\n"},
		{"get-official-version", "fixture", "1.2\n"},
		{"get-tag-prefix", "fixture", "fixture\n"},
		{"get-release-files", "fixture", "sbin/fixture\nconf/fixture.conf\n"},
		{"get-downloads", "fixture", "https://example.invalid/main/1.2\t1.2.tgz\n"},
		{"get-target-from-tag", "fixture-extra-1.2.0", "extended\n"},
	} {
		query(test.command, test.argument, 0, test.expected)
	}
	query("get-env", "absent", 1, "")
	for _, template := range []string{"literal{", "literal}", "{VERSION}{BRANCH}", "{VERSION}{"} {
		write(`{"fixture":{"env":{"VERSION":"1.2","BRANCH":"main"},"downloads":[{"url":"https://example.invalid/{BRANCH}/{VERSION}","name":"` + template + `"}]}}`)
		query("get-downloads", "fixture", 0, "https://example.invalid/main/1.2\t"+strings.NewReplacer("{VERSION}", "1.2", "{BRANCH}", "main").Replace(template)+"\n")
	}
	write(`{"fixture":{"env":{"VERSION":"1.2"},"downloads":[{"url":"url","name":"{MISSING}"}]}}`)
	query("get-downloads", "fixture", 1, "unknown placeholder")
	write(`{"fixture":{"env":{"VERSION":"1.2"},"downloads":[]}}`)
	query("get-downloads", "fixture", 0, "")
	write(`{"fixture":{"version_env_var":"VERSION","env":{"VERSION":"first","VERSION":"last"},"downloads":[{"url":"{VERSION}","name":"archive"}]}}`)
	query("get-official-version", "fixture", 0, "first\n")
	query("get-downloads", "fixture", 0, "first\tarchive\n")
	for _, content := range []string{`{}`, `[]`, `{"fixture":0}`, `{"":{}}`, `{"fixture":{`, `{"fixture":{},}`, `{"fixture":{"env":{"A":"1",}}}`, `{"fixture":{"release_files":["file",]}}`, `{"fixture":{"downloads":[{"url":"url"}]}}`, `{"fixture":{"unknown":[1,]}}`, `{"fixture":{"unknown":{"A":1,}}}`, `{"fixture":{"unknown":1e}}`, `{"fixture":{"unknown":1.}}`, `{"fixture":{"unknown":"\u1234"}}`, `{"fixture":{"unknown":"\q"}}`, `{"fixture":{"unknown":"unterminated}}`, `{"fixture":{}} trailing`} {
		t.Run(content, func(t *testing.T) {
			write(content)
			query("get-env", "fixture", 1, "invalid metadata schema")
		})
	}
	write(`{"first":{"tag_prefix":"same"},"second":{"tag_prefix":"same"}}`)
	query("get-target-from-tag", "same-1.0.0", 1, "ambiguous tag")
}

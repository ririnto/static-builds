package main

import (
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestManifestQueries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata.json")
	if err := os.WriteFile(path, []byte(`{"healthcheck":{"tag_prefix":"nginx-healthcheck","version_env_var":"NGINX_VERSION","env":{"NGINX_VERSION":"1.30.6"},"release_files":["sbin/nginx","conf"],"downloads":[{"url":"https://example.invalid/nginx-{NGINX_VERSION}.tar.gz","name":"nginx-{NGINX_VERSION}.tar.gz"}]}}`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("METADATA_PATH_OVERRIDE", path)
	for test := range slices.Values([]struct{ query, argument, want string }{
		{"get-env", "healthcheck", "NGINX_VERSION=1.30.6\n"},
		{"get-official-version", "healthcheck", "1.30.6\n"},
		{"get-tag-prefix", "healthcheck", "nginx-healthcheck\n"},
		{"get-target-from-tag", "nginx-healthcheck-1.30.6.0", "healthcheck\n"},
		{"get-release-files", "healthcheck", "sbin/nginx\nconf\n"},
		{"get-downloads", "healthcheck", "https://example.invalid/nginx-1.30.6.tar.gz\tnginx-1.30.6.tar.gz\n"},
	}) {
		t.Run(test.query, func(t *testing.T) {
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := reader.Close(); err != nil {
					t.Error(err)
				}
			}()
			previous := os.Stdout
			os.Stdout = writer
			defer func() { os.Stdout = previous }()
			err = run([]string{test.query, test.argument})
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(reader)
			if err != nil || string(data) != test.want {
				t.Fatalf("output = %q, error = %v; want %q", data, err, test.want)
			}
		})
	}
	if err := run([]string{"get-env", "unknown"}); err == nil {
		t.Fatal("accepted an unknown target")
	}
	if err := run(nil); err == nil {
		t.Fatal("accepted a missing query")
	}
}

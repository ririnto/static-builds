package metadata_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ririnto/static-builds/internal/metadata"
)

func TestLoadPreservesOrderedQueries(t *testing.T) {

	t.Parallel()
	path := filepath.Join(t.TempDir(), "metadata.json")
	if err := os.WriteFile(path, []byte(`{"zeta":{"tag_prefix":"zeta","version_env_var":"VERSION","env":{"VERSION":"first","OTHER":"other","VERSION":"later"},"release_files":["sbin/zeta","conf","sbin/zeta"],"downloads":[{"url":"https://example.invalid/{VERSION}/{OTHER}/{VERSION}","name":"{VERSION}.tar.gz"},{"url":"https://example.invalid/second","name":"second.tar.gz"}]},"alpha":{"env":{"VERSION":"alpha"}},"zeta":{"env":{"VERSION":"duplicate target"}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := metadata.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Targets) != 3 || data.Targets[0].Name != "zeta" || data.Targets[1].Name != "alpha" || data.Targets[2].Name != "zeta" {
		t.Fatalf("target order = %#v", data.Targets)
	}
	target, err := data.Target("zeta")
	if err != nil || target != &data.Targets[0] {
		t.Fatalf("target = %p, error = %v; want first stored target %p", target, err, &data.Targets[0])
	}
	entries, err := target.Environment()
	if err != nil || !slices.Equal(entries, []metadata.Entry{{Key: "VERSION", Value: "first"}, {Key: "OTHER", Value: "other"}, {Key: "VERSION", Value: "later"}}) {
		t.Fatalf("environment = %#v, error = %v", entries, err)
	}
	version, err := target.OfficialVersion()
	if err != nil || version != "first" {
		t.Fatalf("version = %q, error = %v; want first", version, err)
	}
	files, err := target.Files()
	if err != nil || !slices.Equal(files, []string{"sbin/zeta", "conf", "sbin/zeta"}) {
		t.Fatalf("release files = %#v, error = %v", files, err)
	}
	downloads, err := target.RenderDownloads()
	if err != nil || !slices.Equal(downloads, []metadata.Download{{URL: "https://example.invalid/first/other/first", Name: "first.tar.gz"}, {URL: "https://example.invalid/second", Name: "second.tar.gz"}}) {
		t.Fatalf("downloads = %#v, error = %v", downloads, err)
	}
}

func TestTargetFromTag(t *testing.T) {

	t.Parallel()
	for test := range slices.Values([]struct {
		name    string
		targets []metadata.Target
		tag     string
		index   int
		wantErr string
	}{
		{name: "longest prefix", targets: []metadata.Target{{Name: "base", TagPrefix: "nginx"}, {Name: "healthcheck", TagPrefix: "nginx-healthcheck"}}, tag: "nginx-healthcheck-1.0.0", index: 1},
		{name: "later shorter prefix", targets: []metadata.Target{{Name: "healthcheck", TagPrefix: "nginx-healthcheck"}, {Name: "base", TagPrefix: "nginx"}}, tag: "nginx-healthcheck-1.0.0", index: 0},
		{name: "ambiguous prefix", targets: []metadata.Target{{Name: "one", TagPrefix: "nginx"}, {Name: "two", TagPrefix: "nginx"}}, tag: "nginx-1.0.0", wantErr: "ambiguous tag"},
		{name: "duplicate target name", targets: []metadata.Target{{Name: "one", TagPrefix: "nginx"}, {Name: "one", TagPrefix: "nginx"}}, tag: "nginx-1.0.0", index: 0},
		{name: "unknown tag", targets: []metadata.Target{{Name: "one", TagPrefix: "nginx"}}, tag: "other-1.0.0", wantErr: "unknown tag"},
		{name: "empty targets", tag: "nginx-1.0.0", wantErr: "unknown tag"},
	}) {
		t.Run(test.name, func(t *testing.T) {

			t.Parallel()
			data := metadata.Metadata{Targets: test.targets}
			target, err := data.TargetFromTag(test.tag)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) || target != nil {
					t.Fatalf("target = %#v, error = %v; want %s", target, err, test.wantErr)
				}
			} else if err != nil || target != &data.Targets[test.index] {
				t.Fatalf("target = %p, error = %v; want stored target %p", target, err, &data.Targets[test.index])
			}
		})
	}
}

func TestLoadRejectsInvalidMetadata(t *testing.T) {

	t.Parallel()
	for test := range slices.Values([]struct {
		name, input, wantErr string
	}{
		{"empty document", `{}`, "non-empty object"},
		{"invalid target", `{"valid":{"env":{"VERSION":"1"}},"invalid":[]}`, "entries must be objects"},
		{"invalid environment", `{"target":{"env":[]}}`, "malformed env object"},
		{"non-string environment", `{"target":{"env":{"VERSION":1}}}`, "expected string"},
		{"invalid download fields", `{"target":{"downloads":[{"url":"https://example.invalid"}]}}`, "invalid download fields in entry #1"},
		{"invalid second download", `{"target":{"downloads":[{"url":"https://example.invalid","name":"source"},false]}}`, "invalid download entry #2"},
		{"trailing content", `{"target":{}} {}`, "unexpected trailing content"},
		{"unicode in ignored field", `{"target":{"ignored":"\u0061"}}`, "unsupported unicode escape"},
	}) {
		t.Run(test.name, func(t *testing.T) {

			t.Parallel()
			path := filepath.Join(t.TempDir(), "metadata.json")
			if err := os.WriteFile(path, []byte(test.input), 0600); err != nil {
				t.Fatal(err)
			}
			data, err := metadata.Load(path)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) || data != nil {
				t.Fatalf("metadata = %#v, error = %v; want %s", data, err, test.wantErr)
			}
		})
	}
}

func TestDownloadPlaceholderErrors(t *testing.T) {

	t.Parallel()
	for test := range slices.Values([]struct {
		name      string
		downloads []metadata.Download
		wantErr   string
	}{
		{name: "unknown URL placeholder", downloads: []metadata.Download{{URL: "{MISSING}", Name: "source"}}, wantErr: "downloads[1].url"},
		{name: "unknown second name placeholder", downloads: []metadata.Download{{URL: "{VERSION}", Name: "first"}, {URL: "{VERSION}", Name: "{MISSING}"}}, wantErr: "downloads[2].name"},
	}) {
		t.Run(test.name, func(t *testing.T) {

			t.Parallel()
			target := metadata.Target{Name: "target", Env: []metadata.Entry{{Key: "VERSION", Value: "1"}}, Downloads: test.downloads}
			downloads, err := target.RenderDownloads()
			if err == nil || !strings.Contains(err.Error(), test.wantErr) || downloads != nil {
				t.Fatalf("downloads = %#v, error = %v; want %s", downloads, err, test.wantErr)
			}
		})
	}
}

func TestEmptyFirstEnvironmentValue(t *testing.T) {

	t.Parallel()
	target := metadata.Target{
		Name:          "target",
		VersionEnvVar: "VERSION",
		Env:           []metadata.Entry{{Key: "VERSION", Value: ""}, {Key: "VERSION", Value: "later"}},
		Downloads:     []metadata.Download{{URL: "https://example.invalid/{VERSION}", Name: "{VERSION}.tar.gz"}},
	}
	if version, err := target.OfficialVersion(); err == nil || !strings.Contains(err.Error(), "missing official version value") || version != "" {
		t.Fatalf("version = %q, error = %v; want missing official version value", version, err)
	}
	downloads, err := target.RenderDownloads()
	if err != nil || !slices.Equal(downloads, []metadata.Download{{URL: "https://example.invalid/", Name: ".tar.gz"}}) {
		t.Fatalf("downloads = %#v, error = %v; want empty placeholder substitution", downloads, err)
	}
}

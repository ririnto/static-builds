package buildconfig

import (
	"reflect"
	"testing"

	"github.com/ririnto/static-builds/internal/metadata"
)

func TestArguments(t *testing.T) {
	got := Arguments([]metadata.Entry{
		{Key: "FIRST", Value: "value\r\n"},
		{Key: "", Value: "ignored"},
		{Key: "#COMMENT", Value: "ignored"},
		{Key: "EMPTY", Value: ""},
		{Key: "lowercase", Value: "ignored"},
		{Key: "CR_ONLY", Value: "\r"},
		{Key: "LAST_2", Value: "a=b"},
	})
	if !reflect.DeepEqual(got, []string{"FIRST=value", "CR_ONLY=", "LAST_2=a=b"}) {
		t.Fatal(got)
	}
}

func TestDefaultsAndNetworks(t *testing.T) {
	t.Setenv("BUILDKIT_PLATFORM", "linux/arm64")
	t.Setenv("BUILDKIT_NETWORK", "host")
	t.Setenv("BUILDKIT_CACHE_BACKEND", "gha")
	t.Setenv("BUILD_OUTPUT_DEST", "output with spaces")
	t.Setenv("CACHE_DIR", "cache with spaces")
	config, err := Load("nginx")
	if err != nil {
		t.Fatal(err)
	}
	if config.Platform != "linux/arm64" || config.Output != "output with spaces" || config.Cache != "cache with spaces" {
		t.Fatal(config)
	}
	from, to, err := config.Caches()
	if err != nil || from != "type=gha,scope=nginx" || to != "type=gha,mode=max,scope=nginx" {
		t.Fatal(from, to, err)
	}
	for _, test := range []struct{ input, want string }{{"default", "sandbox"}, {"host", "host"}, {"none", "none"}, {"invalid", ""}} {
		config.Network = test.input
		got, err := config.FrontendNetwork()
		if got != test.want || (err != nil) != (test.want == "") {
			t.Fatal(test.input, got, err)
		}
	}
}

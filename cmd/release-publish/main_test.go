package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-github/v84/github"
)

func TestPublish(t *testing.T) {
	for _, test := range []struct {
		name    string
		lookup  int
		create  int
		list    int
		delete  int
		upload  int
		want    []string
		failure string
	}{
		{name: "existing replacement", lookup: 200, list: 200, delete: 204, upload: 201, want: []string{"lookup", "list", "delete", "upload"}},
		{name: "missing release", lookup: 404, create: 201, list: 200, delete: 204, upload: 201, want: []string{"lookup", "create", "list", "delete", "upload"}},
		{name: "lookup forbidden", lookup: 403, want: []string{"lookup"}, failure: "look up release"},
		{name: "lookup unavailable", lookup: 503, want: []string{"lookup"}, failure: "look up release"},
		{name: "create failure", lookup: 404, create: 422, want: []string{"lookup", "create"}, failure: "create release"},
		{name: "list failure", lookup: 200, list: 500, want: []string{"lookup", "list"}, failure: "list release assets"},
		{name: "delete failure", lookup: 200, list: 200, delete: 403, want: []string{"lookup", "list", "delete"}, failure: "delete release asset"},
		{name: "upload failure", lookup: 200, list: 200, delete: 204, upload: 422, want: []string{"lookup", "list", "delete", "upload"}, failure: "upload release asset"},
	} {
		t.Run(test.name, func(t *testing.T) {
			tag := "fixture-1.2.3.0"
			path := filepath.Join(t.TempDir(), tag+".tar.gz")
			if err := os.WriteFile(path, []byte("archive bytes"), 0600); err != nil {
				t.Fatal(err)
			}
			var calls []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer fixture-token" {
					t.Error("missing token")
				}
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == "GET" && r.URL.Path == "/repos/fixture/repository/releases/tags/"+tag:
					calls = append(calls, "lookup")
					w.WriteHeader(test.lookup)
					if _, err := fmt.Fprint(w, `{"id":7}`); err != nil {
						t.Error(err)
					}
				case r.Method == "POST" && r.URL.Path == "/repos/fixture/repository/releases":
					calls = append(calls, "create")
					var release github.RepositoryRelease
					if err := json.NewDecoder(r.Body).Decode(&release); err != nil {
						t.Error(err)
					}
					if release.GetTagName() != tag || release.GetName() != tag || release.GetBody() != "Automated static binary release from tag "+tag+"." {
						t.Errorf("incorrect release: %+v", release)
					}
					w.WriteHeader(test.create)
					if _, err := fmt.Fprint(w, `{"id":7}`); err != nil {
						t.Error(err)
					}
				case r.Method == "GET" && r.URL.Path == "/repos/fixture/repository/releases/7/assets":
					calls = append(calls, "list")
					w.WriteHeader(test.list)
					if _, err := fmt.Fprintf(w, `[{"id":8,"name":%q},{"id":9,"name":"unrelated"}]`, filepath.Base(path)); err != nil {
						t.Error(err)
					}
				case r.Method == "DELETE" && r.URL.Path == "/repos/fixture/repository/releases/assets/8":
					calls = append(calls, "delete")
					w.WriteHeader(test.delete)
				case r.Method == "POST" && r.URL.Path == "/repos/fixture/repository/releases/7/assets":
					calls = append(calls, "upload")
					body, err := io.ReadAll(r.Body)
					if err != nil || string(body) != "archive bytes" || r.URL.Query().Get("name") != filepath.Base(path) {
						t.Errorf("incorrect upload: %q, %s, %v", body, r.URL, err)
					}
					w.WriteHeader(test.upload)
					if _, err := fmt.Fprint(w, `{"id":10}`); err != nil {
						t.Error(err)
					}
				default:
					t.Errorf("unexpected request %s %s", r.Method, r.URL)
					http.Error(w, "unexpected", 500)
				}
			}))
			defer server.Close()
			client := github.NewClient(server.Client()).WithAuthToken("fixture-token")
			base, err := url.Parse(server.URL + "/")
			if err != nil {
				t.Fatal(err)
			}
			client.BaseURL, client.UploadURL = base, base
			err = publish(context.Background(), client, "fixture", "repository", tag, path)
			if test.failure == "" && err != nil || test.failure != "" && (err == nil || !strings.Contains(err.Error(), test.failure)) {
				t.Fatalf("unexpected result: %v", err)
			}
			if !reflect.DeepEqual(calls, test.want) {
				t.Fatalf("calls %v, want %v", calls, test.want)
			}
		})
	}
}

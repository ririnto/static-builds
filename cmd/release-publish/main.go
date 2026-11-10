package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/google/go-github/v84/github"
)

func publish(ctx context.Context, client *github.Client, owner, repository, tag, path string) (err error) {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("%s not found: %w", path, err)
	}
	defer func() {
		if closeErr := file.Close(); err == nil && !errors.Is(closeErr, os.ErrClosed) {
			err = closeErr
		}
	}()
	release, response, err := client.Repositories.GetReleaseByTag(ctx, owner, repository, tag)
	if err != nil {
		if response == nil || response.StatusCode != http.StatusNotFound {
			return fmt.Errorf("look up release: %w", err)
		}
		if _, err := fmt.Fprintln(os.Stdout, "Creating release", tag); err != nil {
			return err
		}
		release, _, err = client.Repositories.CreateRelease(ctx, owner, repository, &github.RepositoryRelease{
			TagName: github.Ptr(tag),
			Name:    github.Ptr(tag),
			Body:    github.Ptr("Automated static binary release from tag " + tag + "."),
		})
		if err != nil {
			return fmt.Errorf("create release: %w", err)
		}
	}
	name := filepath.Base(path)
	options := &github.ListOptions{PerPage: 100}
	for {
		assets, response, err := client.Repositories.ListReleaseAssets(ctx, owner, repository, release.GetID(), options)
		if err != nil {
			return fmt.Errorf("list release assets: %w", err)
		}
		for _, asset := range assets {
			if asset.GetName() == name {
				if _, err := fmt.Fprintln(os.Stdout, "Replacing asset", name); err != nil {
					return err
				}
				if _, err := client.Repositories.DeleteReleaseAsset(ctx, owner, repository, asset.GetID()); err != nil {
					return fmt.Errorf("delete release asset: %w", err)
				}
			}
		}
		if response.NextPage == 0 {
			break
		}
		options.Page = response.NextPage
	}
	if _, err := fmt.Fprintln(os.Stdout, "Uploading asset", name); err != nil {
		return err
	}
	if _, _, err := client.Repositories.UploadReleaseAsset(ctx, owner, repository, release.GetID(), &github.UploadOptions{Name: name}, file); err != nil {
		return fmt.Errorf("upload release asset: %w", err)
	}
	return nil
}

func run(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: release-publish <tag>")
	}
	path := args[0] + ".tar.gz"
	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("%s not found", path)
	}
	repository := os.Getenv("GH_REPO")
	if repository == "" {
		repository = os.Getenv("GITHUB_REPOSITORY")
	}
	owner, name, found := strings.Cut(repository, "/")
	if !found || owner == "" || name == "" || strings.Contains(name, "/") {
		return fmt.Errorf("GH_REPO or GITHUB_REPOSITORY must identify owner/repository")
	}
	token := os.Getenv("GH_TOKEN")
	if token == "" {
		return errors.New("GH_TOKEN is required")
	}
	client := github.NewClient(nil).WithAuthToken(token)
	return publish(ctx, client, owner, name, args[0], path)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

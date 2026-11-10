// Package release validates release tags and metadata-selected artifacts.
package release

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ririnto/static-builds/internal/metadata"
)

// ValidateTag checks the canonical prefix, decimal revision, and official version.
func ValidateTag(target *metadata.Target, tag string) (string, error) {
	prefix, err := target.Prefix()
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(tag, prefix+"-") {
		return "", fmt.Errorf("Invalid tag format '%s'\nExpected format: %s-<version>.<revision>", tag, prefix)
	}
	versionRevision := strings.TrimPrefix(tag, prefix+"-")
	version, revision := versionRevision, versionRevision
	if index := strings.LastIndexByte(versionRevision, '.'); index >= 0 {
		version, revision = versionRevision[:index], versionRevision[index+1:]
	}
	if revision == "" || strings.IndexFunc(revision, func(c rune) bool { return c < '0' || c > '9' }) >= 0 {
		return "", fmt.Errorf("Revision '%s' must contain only decimal digits", revision)
	}
	official, err := target.OfficialVersion()
	if err != nil {
		return "", err
	}
	if version != official {
		if strings.HasPrefix(version, official+"-") {
			prerelease := strings.TrimPrefix(version, official+"-")
			if !regexp.MustCompile(`^[A-Za-z0-9]+(?:[.-][A-Za-z0-9]+)*$`).MatchString(prerelease) {
				return "", fmt.Errorf("Invalid prerelease suffix '%s'", prerelease)
			}
		} else {
			return "", fmt.Errorf("Version mismatch\n  Tag version:      %s (from tag '%s')\n  Official version: %s (from metadata.json)", version, tag, official)
		}
	}
	return version, nil
}

// Files selects release files through the shared metadata contract.
func Files(target *metadata.Target) ([]string, error) {
	return target.Files()
}

// Verify checks that every declared release path names a file or directory.
func Verify(directory string, files []string) error {
	for _, file := range files {
		if file == "" {
			return fmt.Errorf("release path must not be empty")
		}
		path := filepath.Join(directory, file)
		info, err := os.Stat(path)
		if err != nil || !(info.Mode().IsRegular() || info.IsDir()) {
			return fmt.Errorf("Release path not found at %s", path)
		}
	}
	return nil
}

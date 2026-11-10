// Package buildconfig normalizes common build environment and adapter defaults.
package buildconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ririnto/static-builds/internal/metadata"
)

// Config contains common adapter settings without overriding adapter policy.
type Config struct {
	Root         string
	Target       string
	Platform     string
	Network      string
	CacheBackend string
	Output       string
	Cache        string
}

// Default resolves nonempty environment settings with shell-compatible precedence.
func Default(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// Load resolves defaults shared by the three build adapters.
func Load(target string) (Config, error) {
	root, err := os.Getwd()
	if err != nil {
		return Config{}, err
	}
	return Config{
		Root: root, Target: target,
		Platform:     Default("BUILDKIT_PLATFORM", "linux/amd64"),
		Network:      Default("BUILDKIT_NETWORK", "default"),
		CacheBackend: Default("BUILDKIT_CACHE_BACKEND", "local"),
		Output:       Default("BUILD_OUTPUT_DEST", filepath.Join(root, ".out", target)),
		Cache:        Default("CACHE_DIR", filepath.Join(root, ".cache", target)),
	}, nil
}

// ValidTarget checks the adapter target character allowlist.
func ValidTarget(target string) bool {
	return strings.IndexFunc(target, func(c rune) bool {
		return !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.ContainsRune("._-", c))
	}) < 0
}

// Arguments selects normalized build arguments in metadata order.
func Arguments(entries []metadata.Entry) []string {
	var result []string
	for _, entry := range entries {
		if entry.Key != "" && !strings.HasPrefix(entry.Key, "#") && entry.Value != "" && strings.IndexFunc(entry.Key, func(c rune) bool {
			return !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_')
		}) < 0 {
			result = append(result, entry.Key+"="+strings.TrimRight(strings.ReplaceAll(entry.Value, "\r", ""), "\n"))
		}
	}
	return result
}

// Caches returns the native local or Actions cache exporter configuration.
func (c Config) Caches() (string, string, error) {
	switch c.CacheBackend {
	case "local":
		return "type=local,src=" + c.Cache, "type=local,dest=" + c.Cache + ",mode=max", nil
	case "gha":
		return "type=gha,scope=" + c.Target, "type=gha,mode=max,scope=" + c.Target, nil
	default:
		return "", "", fmt.Errorf("Invalid BUILDKIT_CACHE_BACKEND: %s", c.CacheBackend)
	}
}

// FrontendNetwork maps the native adapter frontend network setting.
func (c Config) FrontendNetwork() (string, error) {
	switch c.Network {
	case "default":
		return "sandbox", nil
	case "none", "host":
		return c.Network, nil
	default:
		return "", fmt.Errorf("Invalid BUILDKIT_NETWORK: %s", c.Network)
	}
}

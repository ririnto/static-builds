// Package metadata loads ordered target metadata and resolves its command queries.
package metadata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Entry preserves environment order and duplicate-key lookup behavior.
type Entry struct {
	Key   string
	Value string
}

// Download describes one rendered source archive.
type Download struct {
	URL  string
	Name string
}

// Target contains the ordered values used by build and release commands.
type Target struct {
	Name          string
	TagPrefix     string
	VersionEnvVar string
	Env           []Entry
	ReleaseFiles  []string
	Downloads     []Download
}

// Metadata preserves target order from the input document.
type Metadata struct {
	Targets []Target
}

type member struct {
	key   string
	value value
}

type value struct {
	token  json.Token
	object []member
	array  []value
}

func readValue(decoder *json.Decoder) (value, error) {
	token, err := decoder.Token()
	if err != nil {
		return value{}, err
	}
	result := value{token: token}
	if token == json.Delim('{') {
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				return value{}, err
			}
			name, ok := key.(string)
			if !ok || name == "" {
				return value{}, fmt.Errorf("object key must not be empty")
			}
			child, err := readValue(decoder)
			if err != nil {
				return value{}, err
			}
			result.object = append(result.object, member{key: name, value: child})
		}
		_, err = decoder.Token()
	} else if token == json.Delim('[') {
		for decoder.More() {
			child, err := readValue(decoder)
			if err != nil {
				return value{}, err
			}
			result.array = append(result.array, child)
		}
		_, err = decoder.Token()
	}
	return result, err
}

func stringValue(v value) (string, error) {
	text, ok := v.token.(string)
	if !ok {
		return "", fmt.Errorf("expected string")
	}
	return text, nil
}

// Load reads and validates the entire metadata file before returning any queries.
func Load(path string) (*Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("metadata file not found: %s", path)
		}
		return nil, err
	}
	result, err := parse(data)
	if err != nil {
		return nil, fmt.Errorf("invalid metadata schema: %w", err)
	}
	return result, nil
}

func parse(data []byte) (*Metadata, error) {
	// The former parser rejects Unicode escapes, including in ignored fields.
	inString, escaped := false, false
	for character := range slices.Values(data) {
		if escaped {
			if character == 'u' {
				return nil, fmt.Errorf("unsupported unicode escape")
			}
			escaped = false
		} else if inString && character == '\\' {
			escaped = true
		} else if character == '"' {
			inString = !inString
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	root, err := readValue(decoder)
	if err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, fmt.Errorf("unexpected trailing content")
	}
	if root.token != json.Delim('{') || len(root.object) == 0 {
		return nil, fmt.Errorf("metadata must be a non-empty object")
	}
	result := &Metadata{}
	for item := range slices.Values(root.object) {
		if item.value.token != json.Delim('{') {
			return nil, fmt.Errorf("metadata entries must be objects keyed by target name")
		}
		target := Target{Name: item.key}
		for field := range slices.Values(item.value.object) {
			switch field.key {
			case "tag_prefix", "version_env_var":
				text, err := stringValue(field.value)
				if err != nil {
					return nil, err
				}
				if field.key == "tag_prefix" {
					target.TagPrefix = text
				} else {
					target.VersionEnvVar = text
				}
			case "env":
				if field.value.token != json.Delim('{') {
					return nil, fmt.Errorf("target '%s' has malformed env object", item.key)
				}
				for entry := range slices.Values(field.value.object) {
					text, err := stringValue(entry.value)
					if err != nil {
						return nil, err
					}
					target.Env = append(target.Env, Entry{Key: entry.key, Value: text})
				}
			case "release_files":
				if field.value.token != json.Delim('[') {
					return nil, fmt.Errorf("target '%s' has malformed release_files array", item.key)
				}
				for entry := range slices.Values(field.value.array) {
					text, err := stringValue(entry)
					if err != nil {
						return nil, err
					}
					target.ReleaseFiles = append(target.ReleaseFiles, text)
				}
			case "downloads":
				if field.value.token != json.Delim('[') {
					return nil, fmt.Errorf("target '%s' has invalid downloads metadata", item.key)
				}
				for index, entry := range field.value.array {
					download := Download{}
					urlFound, nameFound := false, false
					if entry.token != json.Delim('{') {
						return nil, fmt.Errorf("target '%s' has invalid download entry #%d", item.key, index+1)
					}
					for downloadField := range slices.Values(entry.object) {
						if downloadField.key == "url" || downloadField.key == "name" {
							text, err := stringValue(downloadField.value)
							if err != nil {
								return nil, err
							}
							if downloadField.key == "url" {
								download.URL, urlFound = text, true
							} else {
								download.Name, nameFound = text, true
							}
						}
					}
					if !urlFound || !nameFound {
						return nil, fmt.Errorf("target '%s' has invalid download fields in entry #%d", item.key, index+1)
					}
					target.Downloads = append(target.Downloads, download)
				}
			}
		}
		result.Targets = append(result.Targets, target)
	}
	return result, nil
}

// LoadDefault honors the existing metadata override and otherwise uses the repository root.
func LoadDefault() (*Metadata, error) {
	path := os.Getenv("METADATA_PATH_OVERRIDE")
	if path == "" {
		root, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(root, "metadata.json")
	}
	return Load(path)
}

// Target selects a target without changing input ordering.
func (m *Metadata) Target(name string) (*Target, error) {
	if index := slices.IndexFunc(m.Targets, func(target Target) bool {
		return target.Name == name
	}); index >= 0 {
		return &m.Targets[index], nil
	}
	return nil, fmt.Errorf("unknown target '%s'", name)
}

// TargetFromTag resolves the longest prefix and rejects ambiguous matches.
func (m *Metadata) TargetFromTag(tag string) (*Target, error) {
	var best *Target
	ambiguous := false
	for index := range m.Targets {
		target := &m.Targets[index]
		if target.TagPrefix != "" && strings.HasPrefix(tag, target.TagPrefix+"-") {
			if best == nil || len(target.TagPrefix) > len(best.TagPrefix) {
				best, ambiguous = target, false
			} else if len(target.TagPrefix) == len(best.TagPrefix) && target.Name != best.Name {
				ambiguous = true
			}
		}
	}
	if best == nil {
		return nil, fmt.Errorf("unknown tag '%s'", tag)
	}
	if ambiguous {
		return nil, fmt.Errorf("ambiguous tag '%s': targets share longest tag prefix '%s'", tag, best.TagPrefix)
	}
	return best, nil
}

// Environment returns ordered build environment records.
func (t *Target) Environment() ([]Entry, error) {
	if len(t.Env) == 0 {
		return nil, fmt.Errorf("target '%s' is missing env metadata", t.Name)
	}
	return t.Env, nil
}

// Prefix returns the required canonical release prefix.
func (t *Target) Prefix() (string, error) {
	if t.TagPrefix == "" {
		return "", fmt.Errorf("target '%s' is missing tag_prefix", t.Name)
	}
	return t.TagPrefix, nil
}

// OfficialVersion uses the first matching environment entry.
func (t *Target) OfficialVersion() (string, error) {
	if t.VersionEnvVar == "" {
		return "", fmt.Errorf("target '%s' is missing version_env_var", t.Name)
	}
	if index := slices.IndexFunc(t.Env, func(entry Entry) bool {
		return entry.Key == t.VersionEnvVar
	}); index >= 0 && t.Env[index].Value != "" {
		return t.Env[index].Value, nil
	}
	return "", fmt.Errorf("target '%s' is missing official version value", t.Name)
}

// Files selects the ordered release paths.
func (t *Target) Files() ([]string, error) {
	if len(t.ReleaseFiles) == 0 {
		return nil, fmt.Errorf("target '%s' is missing release_files", t.Name)
	}
	return t.ReleaseFiles, nil
}

func (t *Target) render(remainder string, index int, field string) (string, error) {
	var output strings.Builder
	for strings.Contains(remainder, "{") && strings.Contains(remainder[strings.IndexByte(remainder, '{')+1:], "}") {
		prefix, tail, _ := strings.Cut(remainder, "{")
		placeholder, rest, _ := strings.Cut(tail, "}")
		output.WriteString(prefix)
		if entry := slices.IndexFunc(t.Env, func(entry Entry) bool {
			return entry.Key == placeholder
		}); entry >= 0 {
			output.WriteString(t.Env[entry].Value)
		} else {
			return "", fmt.Errorf("target '%s' references unknown placeholder '%s' in downloads[%d].%s", t.Name, placeholder, index, field)
		}
		remainder = rest
	}
	output.WriteString(remainder)
	return output.String(), nil
}

// RenderDownloads substitutes environment placeholders in source records.
func (t *Target) RenderDownloads() ([]Download, error) {
	if _, err := t.Environment(); err != nil {
		return nil, err
	}
	var downloads []Download
	for index, entry := range t.Downloads {
		url, err := t.render(entry.URL, index+1, "url")
		if err != nil {
			return nil, err
		}
		name, err := t.render(entry.Name, index+1, "name")
		if err != nil {
			return nil, err
		}
		downloads = append(downloads, Download{URL: url, Name: name})
	}
	return downloads, nil
}

# Code Style and Conventions

## Shell Scripts

### General Rules

- Use **POSIX sh** (`#!/usr/bin/env sh`) for maximum portability
- Enable strict mode: `set -eu`
- Indent with **2 spaces** (as defined in `.editorconfig`)
- Max line length: 120 characters
- End files with a newline

### Variable Naming

- Use **UPPERCASE** for environment variables and constants
- Use **lowercase** for local variables
- Quote variables: `"${VAR}"` (not `$VAR`)

### Functions

- Document with comments above function
- Use descriptive names

### Example

```sh
#!/usr/bin/env sh
set -eu

# Downloads a file from URL to destination.
# @param $1 url - Source URL
# @param $2 dest - Destination path
download_file() {
    url="${1}"
    dest="${2}"
    
    wget -q -O "${dest}" "${url}"
}
```

## YAML Files

- Indent with **2 spaces**
- Spaces within braces and brackets: `{ key: value }`, `[ item ]`

## Dockerfiles

- Use `# syntax=docker/dockerfile:1.4` directive
- Use `ARG` for version parameters
- Comment configure/make options extensively
- Use `--mount=type=cache` for package manager caches

## .env Files

- Format: `KEY=value`
- No spaces around `=`
- No quotes needed for simple values
- Group related variables together

## EditorConfig

The project uses `.editorconfig` for consistent formatting:

- UTF-8 encoding
- LF line endings
- 2-space indentation for shell/YAML
- 4-space indentation for other files

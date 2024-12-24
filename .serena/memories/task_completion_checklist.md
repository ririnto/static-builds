# Task Completion Checklist

## Before Committing Changes

### 1. Validate Shell Scripts

```bash
# Check syntax (requires shellcheck - optional)
shellcheck build.sh
shellcheck nginx/pre-download.sh
shellcheck haproxy/pre-download.sh
shellcheck apache-httpd/pre-download.sh
```

### 2. Verify Script Permissions

```bash
# Ensure scripts are executable
chmod +x build.sh
chmod +x */pre-download.sh
```

### 3. Test the Build

```bash
# Test that the build runs successfully
./build.sh nginx      # or target you modified
```

### 4. Check EditorConfig Compliance

- Ensure files use LF line endings
- Verify 2-space indentation for shell/YAML files
- Confirm final newline is present

### 5. Review Changes

```bash
git diff
git status
```

### 6. Commit with Meaningful Message

```bash
git add .
git commit -m "type: description of changes"
```

## Commit Message Guidelines

- Use present tense: "Add feature" not "Added feature"
- Keep first line under 50 characters
- Use types: feat, fix, docs, style, refactor, test, chore

## Notes

- No automated CI/CD visible in the project
- No formal test suite - manual verification via build execution
- The project primarily uses Docker/Buildkit - no local linting required beyond shell syntax

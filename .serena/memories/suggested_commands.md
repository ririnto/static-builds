# Suggested Commands

## Building Targets

### General Build Command

```bash
./build.sh <target>
```

### Example Targets

```bash
./build.sh nginx
./build.sh haproxy
./build.sh apache-httpd
```

## Prerequisites

- Docker with Docker Compose installed
- Buildkit support (uses `moby/buildkit:rootless` image)

## Useful Git Commands (macOS/Darwin)

```bash
# Check status
git status

# View changes
git diff

# Stage and commit
git add .
git commit -m "message"

# Push changes
git push origin main
```

## File System Commands (macOS/Darwin)

```bash
# List files
ls -la

# Find files
find . -name "*.sh" -type f

# Search in files
grep -r "pattern" .

# Change directory
cd target_directory
```

## Docker Commands

```bash
# List images
docker images

# Remove build cache
docker builder prune

# Check compose config
docker compose -f .github/docker-compose.yaml config
```

## Notes

- The `.cache/` directory is used for Buildkit layer caching
- Build outputs are placed in the target directory
- Each target's `.env` file controls software versions

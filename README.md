# static-builds

Build statically-linked binaries using Docker multi-stage builds for portable, minimal container deployments.

## Features

- **Static Linking** - Produce fully statically-linked binaries using musl libc
- **Multi-stage Builds** - Leverage Docker BuildKit for efficient, cacheable builds
- **Minimal Images** - Target Red Hat UBI9 Micro or similar minimal runtime images
- **Extensible** - Add new build targets by following a simple directory structure
- **Reproducible** - Version-controlled configurations via `.env` files

## Prerequisites

- Docker with Docker Compose
- BuildKit support (automatically uses `moby/buildkit:rootless`)

## Usage

```bash
./build.sh <target>
```

### Example Targets

```bash
./build.sh nginx
./build.sh haproxy
./build.sh apache-httpd
```

## Project Structure

```text
.
├── build.sh              # Main build entry point
├── .github/
│   ├── docker-compose.yaml
│   └── scripts/
│       ├── build.sh      # BuildKit execution script
│       └── download.sh   # Source download utilities
└── <target>/             # Build target directory
    ├── .env              # Version configuration
    ├── Dockerfile        # Multi-stage build definition
    └── pre-download.sh   # Source download script (optional)
```

## Adding a New Target

1. Create a new directory with your target name
2. Add a `.env` file with version variables:

   ```text
   ALPINE_VERSION=3.23
   YOUR_SOFTWARE_VERSION=1.0.0
   ```

3. Add a `Dockerfile` with your multi-stage build configuration
4. Optionally add `pre-download.sh` for source downloads

> [!TIP]
> Check existing targets (`nginx/`, `haproxy/`, `apache-httpd/`) for reference implementations.

## How It Works

1. The `build.sh` script validates the target directory and required files
2. Docker Compose launches a BuildKit container
3. BuildKit executes the multi-stage Dockerfile
4. Built artifacts are output to the target directory

Build caching is automatically handled via the `.cache/` directory.

# Project Overview: static-builds

## Purpose

This project provides **static build configurations** for various software using Docker-based multi-stage builds. The goal is to produce statically-linked binaries that are portable and suitable for minimal container images (like Red Hat UBI9 Micro).

## Current Build Targets (Examples)

- nginx
- HAProxy
- Apache HTTP Server (apache-httpd)

*Note: These are example configurations. The project structure can be extended to support other software.*

## Tech Stack

- **Shell scripts** (POSIX sh) - Build orchestration and helper scripts
- **Docker/Buildkit** - Container-based build environment
- **Alpine Linux** - Build base image (for musl libc static builds)
- **Red Hat UBI9 Micro** - Target runtime image (optional)

## Project Structure

```text
.
├── build.sh                    # Main entry point for builds
├── .editorconfig               # Code formatting rules
├── README.md
├── .github/
│   ├── docker-compose.yaml     # Docker Compose for build service
│   └── scripts/
│       ├── build.sh            # Buildkit execution script
│       └── download.sh         # Source download utilities
└── <target>/                   # One directory per build target
    ├── .env                    # Version configuration
    ├── Dockerfile              # Multi-stage build definition
    └── pre-download.sh         # Source pre-download script (optional)
```

## How It Works

1. Each target has its own directory with:
   - `.env` - Version numbers for software components
   - `Dockerfile` - Multi-stage build with detailed configure options
   - `pre-download.sh` - Script to download source tarballs (optional)

2. The `build.sh` script:
   - Validates target directory and required files
   - Uses Docker Compose with Buildkit to execute the build
   - Outputs built artifacts to the target directory

3. Builds use Alpine Linux for compilation and can target minimal runtime images.

## Adding New Targets

To add a new build target:

1. Create a new directory with the target name
2. Add `.env` with required version variables
3. Add `Dockerfile` with multi-stage build configuration
4. Optionally add `pre-download.sh` for source downloads

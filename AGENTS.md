# PROJECT KNOWLEDGE BASE

## OVERVIEW

Docker multi-stage build system for statically-linked binaries. Builds nginx, haproxy, apache-httpd, coredns, dnsmasq, vector, monit using musl libc.

## STRUCTURE

```text
static-builds/
├── build.sh              # Main build entry
├── download.sh           # Source download dispatcher
├── Makefile              # Build orchestration (7 targets)
├── .github/
│   ├── docker-compose.yaml  # BuildKit container
│   ├── scripts/common.sh    # Shared functions
│   └── workflows/           # Tag-triggered release
├── nginx/                # Target dirs: Dockerfile + .env
├── haproxy/
├── apache-httpd/
├── coredns/
├── dnsmasq/
├── monit/
└── vector/
```


## WHERE TO LOOK

| Task | Location | Notes |
| --- | --- | --- |
| Add new target | Root + create dir | Follow nginx/ pattern |
| CI release config | .github/workflows/ | Tag-triggered only |
| Build definition | */Dockerfile | Multi-stage Alpine |
| Version config | */.env | ALPINE_VERSION, *_VERSION |

## CONVENTIONS

- EditorConfig: 4-space indent (2 for .sh/.yaml)
- Makefile targets: nginx, haproxy, apache-httpd, coredns, dnsmasq, vector, monit
- Target dir: Must have Dockerfile + .env; download.sh optional

## ANTI-PATTERNS (THIS PROJECT)

- No regular CI (push/PR workflows absent)
- docker-compose.yaml hidden in .github/ (unusual location)
- No test framework - build infrastructure only

## UNIQUE STYLES

- Tag-triggered release: `nginx-1.25.0` → builds + uploads artifact
- Release tags MUST follow `<target>-<official_version>.<x>`.
  - `official_version`: version from target `.env` (for example `NGINX_VERSION`, `HTTPD_VERSION`, `HAPROXY_VERSION`)
  - `x`: release revision suffix starting at `0` and incrementing (`.0`, `.1`, `.2`, ...)
  - examples: `nginx-1.28.2.18`, `httpd-2.4.66.5`, `haproxy-3.2.13.0`
- Per-target caller workflows + reusable template pattern
- Artifacts: `<target>/sbin/nginx`, `sbin/haproxy`, `bin/httpd`, etc.

## COMMANDS

```bash
make help
make list-targets
make build nginx
make download nginx
```

## NOTES

- Build caching via .cache/ directory
- Uses moby/buildkit:rootless container


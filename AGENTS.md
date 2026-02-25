# PROJECT KNOWLEDGE BASE

## OVERVIEW

Docker multi-stage build system for statically-linked binaries. Builds nginx, haproxy, apache-httpd, coredns, dnsmasq, vector using musl libc.

## STRUCTURE

```text
static-builds/
├── build.sh              # Main build entry
├── download.sh           # Source download dispatcher
├── Makefile              # Build orchestration (6 targets)
├── .github/
│   ├── docker-compose.yaml  # BuildKit container
│   ├── scripts/common.sh    # Shared functions
│   └── workflows/           # Tag-triggered release
├── nginx/                # Target dirs: Dockerfile + .env
├── haproxy/
├── apache-httpd/
├── coredns/
├── dnsmasq/
├── vector/
└── third-party/          # Git submodules (empty by default)
    ├── apache-apr/          # Apache Portable Runtime
    ├── apache-apr-util/     # Apache Portable Runtime Utilities
    ├── apache-httpd/       # Apache HTTPD source
    ├── coredns/             # CoreDNS DNS server
    ├── dnsmasq/             # DNS forwarder/ DHCP server
    ├── haproxy/             # HAProxy load balancer
    ├── lua-nginx-module/    # Nginx Lua module
    ├── lua-resty-upstream-healthcheck/ # Lua upstream healthcheck
    ├── nginx-vts-module/    # Nginx Virtual Traffic Statistics
    ├── nginx/               # Nginx source
    └── vector/              # Vector observability pipeline
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
- Makefile targets: nginx, haproxy, apache-httpd, coredns, dnsmasq, vector
- Target dir: Must have Dockerfile + .env; download.sh optional

## ANTI-PATTERNS (THIS PROJECT)

- No regular CI (push/PR workflows absent)
- docker-compose.yaml hidden in .github/ (unusual location)
- No test framework - build infrastructure only

## UNIQUE STYLES

- Tag-triggered release: `nginx-1.25.0` → builds + uploads artifact
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
- third-party/ submodules need `git submodule update --init`

## THIRD-PARTY MODULES

The third-party/ directory contains external sources as git submodules:

- apache-apr: Apache Portable Runtime for Apache HTTPD
- apache-apr-util: APR utilities for Apache HTTPD
- apache-httpd: Apache HTTPD source code
- coredns: CoreDNS DNS server and plugin system
- dnsmasq: Lightweight DNS forwarder and DHCP server
- haproxy: High availability TCP/HTTP load balancer
- lua-nginx-module: Nginx Lua scripting engine
- lua-resty-upstream-healthcheck: Lua upstream healthcheck library
- nginx-vts-module: Nginx virtual traffic statistics module
- nginx: Nginx web server source code
- vector: High-performance observability pipeline

> [!IMPORTANT]
> These third-party modules are external dependencies and MUST NOT be modified. They are used as-is for reference only. Changes to these components SHOULD be made in their respective upstream repositories, not in this project.

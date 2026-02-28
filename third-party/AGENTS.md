# THIRD-PARTY MODULES

This document defines the rules and conventions for third-party dependencies in this project.

## OVERVIEW

The `third-party/` directory contains external sources as git submodules. These are upstream projects that this build system uses to create statically-linked binaries.

## MODULE LIST

The following git submodules are included in this project:

- `third-party/apache/apr`: Apache Portable Runtime for Apache HTTPD
- `third-party/apache/apr-util`: APR utilities for Apache HTTPD
- `third-party/apache/httpd`: Apache HTTPD source code
- `third-party/nginx/nginx`: Nginx web server source code
- `third-party/nginx/lua-nginx-module`: Nginx Lua scripting engine
- `third-party/nginx/lua-resty-core`: Lua resty core library
- `third-party/nginx/lua-resty-upstream-healthcheck`: Lua upstream healthcheck library
- `third-party/nginx/nginx-vts-module`: Nginx virtual traffic statistics module
- `third-party/haproxy/haproxy`: High availability TCP/HTTP load balancer
- `third-party/coredns/coredns`: CoreDNS DNS server and plugin system
- `third-party/dnsmasq/dnsmasq`: Lightweight DNS forwarder and DHCP server
- `third-party/vector/vector`: High-performance observability pipeline
- `third-party/monit/monit`: Process monitoring and management tool

## RULES

### Module Management

- The `.gitmodules` file is the source of truth for the submodule list
- All submodules MUST be initialized with `git submodule update --init`
- Submodule versions SHOULD be pinned to specific commits or tags

### Modification Policy

- Third-party submodule source files MUST NOT be modified
- These modules are for reference only and MUST be used as-is
- Changes to third-party components SHOULD be made in their respective upstream repositories
- Patches MAY be applied during the build process via build scripts, but MUST NOT be committed to the submodule

### Adding New Modules

- New submodules MUST be added under an appropriate category directory (e.g., `nginx/`, `apache/`)
- The `.gitmodules` file MUST be updated with the new submodule path and URL
- This document SHOULD be updated to include the new module in the list above

### Build Integration

- Build scripts MAY read from third-party sources
- Build scripts MAY apply patches or modifications during compilation
- Build scripts MUST NOT modify the third-party source files in place

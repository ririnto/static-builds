# nginx-resty-upstream-healthcheck

This target builds nginx with a native C healthcheck module and VTS traffic metrics as one statically linked musl PIE binary.
The historical target name and release tag prefix remain unchanged.
LuaJIT, Lua runtime files, and the repository-owned `resty.core` bridge are removed.
Existing Lua healthcheck configurations must migrate to the native directives below.

## Implementation Choice

The upstream Lua library cannot use its normal installation path with this target's static musl linking model.
[LuaJIT `ffi.C`](https://luajit.org/ext_ffi_api.html#ffi_C) resolves exported symbols through the process symbol namespace.
[musl's static `dlsym`](https://git.musl-libc.org/cgit/musl/tree/src/ldso/dlsym.c) cannot provide that lookup.
The upstream [`lua-resty-core`](https://github.com/openresty/lua-resty-core) depends on FFI rather than the native Lua C API.
Export linker flags do not supply a dynamic loader to a static executable.

| Approach | Decision |
| --- | --- |
| Native upstream OpenResty libraries | Their FFI dependency conflicts with this target's static musl contract. |
| Local `resty.core` compatibility runtime | It replaces upstream APIs and is removed entirely. |
| [njs](https://nginx.org/en/docs/http/ngx_http_js_module.html) | Static compilation and JavaScript HTTP checks work, but the public API does not expose resolved peer identity or raw TCP probing. |
| [ngx_dynamic_healthcheck](https://github.com/ZigzagAK/ngx_dynamic_healthcheck/tree/1.3.8) | Its handcrafted TLS hello probe cannot verify HTTPS response status, certificates, or SNI. |
| Native healthcheck module | It uses nginx peer locks, references, event connections, and OpenSSL without a compatibility runtime. |

The native implementation uses nginx's existing upstream resolver rather than introducing a second DNS implementation.
It operates on the peers nginx actually routes to, including multiple DNS addresses and backup peers.

### Native Installation and njs Validation

A startup probe against the previous static artifact attempted the normal LuaJIT `ffi.C.ngx_http_lua_ffi_now` lookup.
It failed with `Symbol not found: ngx_http_lua_ffi_now`.
This confirms the static-symbol limitation independently of the compatibility bridge's API coverage.

A separate Linux arm64 prototype statically compiled nginx 1.30.5 with njs 1.0.1 through `--add-module` using this target's security flags.
Its ELF verification found a static PIE without an interpreter or dynamic library dependencies.
Pure JavaScript `js_import`, worker-zero `js_periodic`, shared dictionaries, checks of two literal IPs, failure exclusion, recovery, all-down rejection, and basic health metrics passed real local runtime checks.

That prototype established the feasibility of static njs and JavaScript for HTTP checks against known IPs.
It did not establish dynamic DNS or stream TCP support.
The njs [Fetch API](https://nginx.org/en/docs/njs/reference.html#ngx_fetch) provides HTTP and HTTPS requests.
Its [response URL implementation](https://github.com/nginx/njs/blob/1.0.1/nginx/ngx_js_fetch.c) preserves the original request URL rather than exposing the selected resolved address.
The [resolver connection state](https://github.com/nginx/njs/blob/1.0.1/nginx/ngx_js_http.c) remains internal C state.
The documented [integration model](https://nginx.org/en/docs/njs/integration.html) does not provide arbitrary socket access.
Checking a hostname and proxying to that hostname again would therefore not guarantee that traffic uses the IP that passed the check.
Adding native DNS and TCP helper APIs would reintroduce substantial C implementation around the JavaScript layer.
The selected native module handles those requirements directly.

## Build and Packaging

```bash
make build nginx-resty-upstream-healthcheck
```

The build installs the native module through nginx's standard `--add-module` configure option.
The C module and its dependencies are linked into `sbin/nginx`.
Release selection in root `metadata.json` includes only that binary.
The complete local output remains under `.out/nginx-resty-upstream-healthcheck/`.
`BUILD_OUTPUT_DEST` can override the output directory.

Protocol modules include HTTP TLS, HTTP/2, HTTP/3, stream TLS, real IP, and stream TLS preread.
Utility modules include gzip static, stub status, VTS, threads, and file AIO.
FastCGI, uWSGI, SCGI, SSI, autoindex, split clients, userid, memcached, empty GIF, and browser modules remain disabled.

## Allowed Target-Specific Profile

This target adds native active healthchecks for HTTP and stream upstream groups with shared memory zones.
Its verification stage runs real local upstream servers, DNS responses, TLS sessions, and client traffic.
The shared static ELF verification contract remains the same as other targets.
The verification Python script and generated certificates are build tools and are excluded from release artifacts.
Both interfaces use nginx's native peer lists and route through the checked peer instances.

## Configuration

Replace `lua_shared_dict`, `lua_package_path`, and `spawn_checker` blocks with native upstream directives.
No `load_module`, Lua path, or runtime library installation is needed.

```nginx
worker_processes 2;
error_log stderr notice;
events {}
http {
    resolver 127.0.0.53 valid=10s;
    resolver_timeout 2s;
    vhost_traffic_status_zone shared:vhost_traffic_status:16m;

    upstream api {
        zone api 256k;
        server api.internal:8080 resolve;
        server fallback.internal:8080 resolve backup;
        server 192.0.2.10:8080 down;
        healthcheck type=http interval=2s timeout=1s fall=3 rise=2
                    uri=/readyz host=api.internal;
        healthcheck_statuses 200 204;
    }
    upstream secure_api {
        zone secure_api 256k;
        server secure-api.internal:8443 resolve;
        healthcheck type=https interval=5s timeout=2s fall=3 rise=2
                    uri=/readyz host=secure-api.internal;
        healthcheck_tls_name secure-api.internal;
        healthcheck_trusted_certificate conf/backend-ca.pem;
    }
    server {
        listen 8080;
        location /api/ {
            proxy_pass http://api;
        }
        location /secure/ {
            proxy_pass https://secure_api;
            proxy_ssl_server_name on;
            proxy_ssl_name secure-api.internal;
            proxy_ssl_verify on;
            proxy_ssl_trusted_certificate conf/backend-ca.pem;
        }
    }
    server {
        listen 127.0.0.1:18080;
        location = /health-metrics {
            healthcheck_metrics;
        }
        location = /traffic-metrics {
            vhost_traffic_status_display;
            vhost_traffic_status_display_format prometheus;
        }
    }
}
```

Replace the resolver, backend names, CA file, and listener addresses for your environment.
Healthcheck TLS verification and proxy TLS verification are configured separately.
Keep both enabled for HTTPS backends.

### Healthcheck Parameters

`healthcheck` is configured inside each checked `upstream` block.
A `zone` is required so all workers route using the same health state.
The module uses nginx's built-in round-robin peer storage.
Round robin, least connections, hash, random, and HTTP keepalive retain that storage and can use the checks.
Third-party balancers with different peer storage are outside the supported target profile.

| Parameter | Meaning |
| --- | --- |
| `type=http` | Send an HTTP readiness request and evaluate its status code. |
| `type=https` | Verify the certificate and TLS name before sending the HTTP readiness request. |
| `type=tcp` | Check whether a connection to the actual peer address succeeds. |
| `interval` | Set the delay between checks using nginx time syntax. |
| `timeout` | Bound the complete connection, TLS, request, and response operation. |
| `fall` | Mark the peer down after this many consecutive failed checks. |
| `rise` | Admit a new or recovering peer after this many consecutive successful checks. |
| `uri` | Set the readiness request path for HTTP and HTTPS checks. |
| `host` | Set the HTTP Host header independently of the resolved peer address. |
| `concurrency` | Limit simultaneous probes within one upstream group. |
| `shm_size` | Bound the shared health state storage for one upstream group. |

`healthcheck_statuses` lists accepted final HTTP response codes.
The default accepted status is `200`.
HTTPS checks require `healthcheck_trusted_certificate` and `healthcheck_tls_name`.
`healthcheck_tls_name` sets the certificate hostname and SNI name.
No option disables certificate or hostname verification.

### Stream TCP and TLS Checks

Configure `healthcheck_tcp` inside a `stream` upstream with a shared zone.
The default `type=tcp` succeeds after connecting to the peer's actual address.
Use `type=tls` to require a verified TLS handshake with the configured certificate name and CA.
The interval, complete-operation timeout, failure threshold, recovery threshold, concurrency, and shared-memory settings have the same meaning as their HTTP counterparts.

```nginx
stream {
    resolver 127.0.0.53 valid=10s;
    resolver_timeout 2s;

    upstream cache {
        zone cache 256k;
        server cache.internal:6379 resolve;
        server cache-backup.internal:6379 resolve backup;
        healthcheck_tcp type=tcp interval=2s timeout=1s fall=3 rise=2
                        concurrency=16 shm_size=1m max_response=4096;
        healthcheck_send "PING\r\n";
        healthcheck_expect "+PONG\r\n" min_recv=7;
    }
    server {
        listen 6379;
        proxy_pass cache;
    }
}
```

The example uses Redis's inline PING command and requires a complete positive response.
Use a protocol request appropriate for the backend and its authentication requirements.
`healthcheck_send_hex` and `healthcheck_expect_hex` accept hexadecimal bytes for binary protocols.
Text and hexadecimal forms are alternatives for the same send or expectation setting.
An expectation must be present in the received bytes, and `min_recv` must be satisfied before success.
The default response buffer limit is 4096 bytes and can be set with `max_response`.
Response timeout, early EOF, an unmet expectation, or a full unmatched buffer fails the check.
Sending a payload without an expectation confirms successful transmission rather than application readiness.

TLS checks require `healthcheck_tls_name` and `healthcheck_trusted_certificate` inside the stream upstream.
These control the active probe independently of stream `proxy_ssl` settings for client traffic.
Both transports support the same payload and expectation directives.
The implementation follows HAProxy's connect, send, expect, timeout, and threshold behavior for this bounded interface.
HAProxy's arbitrary multi-step TCP scripts and regular-expression expectations are outside this interface.

### DNS and Peer Lifecycle

Use nginx's native [`resolve`, `zone`, and `resolver`](https://nginx.org/en/docs/http/ngx_http_upstream_module.html) directives.
A domain with several A or AAAA records produces independently checked peers.
Multiple `server` directives and several upstream groups are supported.
The same mechanism can consume nginx's SRV-resolved peer list.
DNS TTL and `valid` control nginx's refresh schedule rather than the healthcheck interval.

New peers start unavailable and require `rise` successful checks before receiving client traffic.
Failure and recovery streaks belong to a peer instance, not its position in the peer list.
A probe holds a native peer reference while the check is outstanding.
Results for removed peers are discarded.
Recreated peers receive fresh health state even when their address matches a removed peer.
An administratively configured `down` server remains unavailable after successful checks.

Worker zero performs the probes and publishes state for all workers.
Checks resume when that worker is replaced.
Configuration reload creates fresh health state and resets its counters.
New workers require the configured recovery threshold before admitting peers.
Graceful shutdown stops new probes and bounds outstanding checks by their deadlines.

## Metrics Integration

`healthcheck_metrics` exports Prometheus health metrics with upstream, configured server, resolved peer, and backup identity.
Use the native metrics together with VTS traffic metrics from two scrape endpoints.
Separate endpoints preserve each module's metric names and avoid custom response-merging code.

```yaml
scrape_configs:
  - job_name: nginx-health
    metrics_path: /health-metrics
    static_configs:
      - targets: ["127.0.0.1:18080"]
  - job_name: nginx-traffic
    metrics_path: /traffic-metrics
    static_configs:
      - targets: ["127.0.0.1:18080"]
```

The example assumes the collector can reach the loopback listener.
A remote collector requires a private reachable listener and appropriate access controls.

The health metrics follow [HAProxy's server metrics](https://github.com/haproxy/haproxy/blob/master/src/stats-proxy.c) for failures, state transitions, last change, response code, and duration.
Durations and ages use seconds, cumulative counts use counters, and current state uses gauges.
An unknown or untested peer must not be interpreted as a successful check.
Check freshness alongside availability so a stopped checker cannot silently appear healthy.
Removed DNS peers disappear from health metrics rather than remaining as stale available servers.

HTTP peer metrics use the `nginx_healthcheck_` prefix.
Stream peer metrics use the `nginx_stream_healthcheck_` prefix on the same `healthcheck_metrics` endpoint.
Their labels identify the upstream, configured server, resolved peer address, and backup role.

| Suffix | Type | Meaning |
| --- | --- | --- |
| `peer_up` | Gauge | The checker currently permits this peer to receive traffic. |
| `admin_down` | Gauge | The server is administratively unavailable. |
| `checks_total` | Counter | Completed checks. |
| `check_failures_total` | Counter | Completed checks that failed. |
| `check_up_down_total` | Counter | Transitions from ready to down, including checker resource failures. |
| `check_status` | Gauge | `-1` before any result, `0` for failure, or `1` for success. |
| `check_code` | Gauge | Parsed HTTP status, or `0` when unavailable. |
| `check_duration_seconds` | Gauge | Duration of the last completed check. |
| `check_last_change_seconds` | Gauge | Time since initialization or the last availability change. |
| `last_check_timestamp_seconds` | Gauge | Unix time of the last completed check, or `0` before any result. |
| `rise_streak`, `fall_streak` | Gauge | Consecutive successful or failed checks. |

Upstream metrics expose checker ownership, active probes, configured concurrency, scan freshness, concurrency limits, and memory or state failures.
The failure-transition counter follows HAProxy's downward-transition meaning rather than counting successful recovery as another failure.

## Runtime Verification

The Docker verify stage runs `verify-runtime.py` against the built binary inside UBI10 Minimal.
It checks local failure and recovery, DNS membership changes, TLS verification, multiple workers, and metrics.
Fixture services and certificates are temporary and never use external backend traffic.
A failed assertion fails the build.

The exported binary targets Linux and cannot run directly on macOS.
Inspect its installed modules with `nginx -V` in a Linux environment.
Use `-p <prefix>` and `-c <config>` to select the deployment directory and configuration.
Only the native interface above is supported after migration.

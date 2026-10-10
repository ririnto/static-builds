# NGINX Upstream Healthcheck

This target builds NGINX Open Source stable release 1.30.5 as a static musl PIE with active HTTP and stream health checks, njs, and VTS metrics.
The healthcheck code is compiled into the binary as `ngx_http_upstream_healthcheck_module`.

## Build and packaged files

Build this target with:

```sh
make build nginx-upstream-healthcheck
```

Build metadata selects the NGINX, njs, QuickJS-NG, and VTS release versions.

| Component | Version | Purpose |
| --- | --- | --- |
| NGINX Open Source | 1.30.5 | Provides the HTTP and stream servers. |
| njs | 1.0.1 | Provides JavaScript scripting in HTTP and stream. |
| QuickJS-NG | 0.17.0 | Provides the recommended JavaScript engine for njs. |
| nginx-module-vts | 0.2.7 | Provides NGINX traffic metrics. |

The build statically compiles the official njs NGINX module and selects the supported `qjs` engine explicitly.
QuickJS-NG is linked statically through its normal CMake build.
The build disables optional libxslt support.
The build output is `.out/nginx-upstream-healthcheck/`.
Release tags use the `nginx-upstream-healthcheck` prefix and take their version from `NGINX_VERSION`.

Release packages include these paths beneath the target prefix:

- `sbin/nginx`
- `conf/scripting.conf`
- `conf/scripting.js`
- `conf/upstream-healthcheck.conf`

The packaged healthcheck configuration shows HTTP, stream TCP, and protected localhost metrics examples.
Replace its resolver and backend names for your environment before using it.

## njs scripting

Both NGINX targets include njs 1.0.1 for HTTP and stream scripting.
Set `js_engine qjs;` in each context that imports an njs script.
The [official engine guidance](https://nginx.org/en/docs/njs/engine.html) recommends QuickJS for new configurations.
The packaged `conf/scripting.conf` and `conf/scripting.js` provide the scripting example.

The healthcheck C module remains responsible for active probes that must use NGINX's selected peer.
njs does not expose that resolved peer address or arbitrary raw TCP sockets to JavaScript.
Pure njs checks therefore cannot safely cover dynamic multi-address DNS peers and stream TCP protocols.

## HTTP health checks

Configure `healthcheck` inside each checked HTTP `upstream` block.
The upstream requires a shared `zone` and a built-in round-robin-compatible peer list.

```nginx
http {
    resolver 127.0.0.53 valid=10s;
    resolver_timeout 2s;
    vhost_traffic_status_zone;

    upstream api {
        zone api 256k;
        server api.internal:8080 resolve;
        server fallback.internal:8080 backup;
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
Healthcheck TLS settings and proxy TLS settings are independent.
Keep certificate and hostname verification enabled for HTTPS backends.

| Type | Probe behavior |
| --- | --- |
| `http` | Sends an HTTP GET request and accepts a configured status code. |
| `https` | Verifies the TLS certificate and name, then sends the HTTP GET request. |
| `tls` | Verifies the TLS handshake, certificate, and name without sending HTTP data. |
| `tcp` | Succeeds when a connection to the selected peer succeeds. |

`https` and `tls` require both `healthcheck_tls_name` and `healthcheck_trusted_certificate`.
The TLS name supplies SNI and certificate hostname verification.
The TLS checks require a trusted CA and do not support disabling verification.
HTTP `type=tls` succeeds after a verified handshake and does not synthesize an HTTP response.
`healthcheck_statuses` lists accepted HTTP statuses and defaults to `200`.

| Parameter | Meaning |
| --- | --- |
| `type` | Selects `http`, `https`, `tls`, or `tcp` probe behavior. |
| `interval` | Sets the delay between checks using NGINX time syntax. |
| `timeout` | Bounds the complete connection, TLS, request, and response operation. |
| `fall` | Marks a peer unavailable after this many consecutive failed checks. |
| `rise` | Admits a new or recovering peer after this many successful checks. |
| `uri` | Sets the readiness request path for `http` and `https` checks. |
| `host` | Sets the HTTP `Host` header independently of the resolved peer address. |
| `concurrency` | Limits simultaneous probes within the upstream group. |
| `shm_size` | Sets the shared health state size for the upstream group. |

The HTTP check interval defaults to five seconds and the timeout defaults to one second.
The default failure threshold is three checks and the recovery threshold is two checks.
The default concurrency is 16 probes and the health state zone defaults to one mebibyte.

## Stream TCP and TLS checks

Configure `healthcheck_tcp` inside a checked `stream` `upstream` block with a shared `zone`.
The default `type=tcp` probe succeeds after connecting to the selected peer.

```nginx
stream {
    resolver 127.0.0.53 valid=10s;
    resolver_timeout 2s;

    upstream cache {
        zone cache 256k;
        server cache.internal:6379 resolve;
        server cache-backup.internal:6379 backup;
        healthcheck_tcp type=tcp interval=2s timeout=1s fall=3 rise=2
                        concurrency=16 shm_size=1m max_response=4096;
        healthcheck_send "PING\r\n";
        healthcheck_expect "+PONG\r\n" min_recv=7;
    }

    upstream secure_service {
        zone secure_service 256k;
        server secure-service.internal:443 resolve;
        healthcheck_tcp type=tls interval=5s timeout=2s fall=3 rise=2;
        healthcheck_tls_name secure-service.internal;
        healthcheck_trusted_certificate conf/backend-ca.pem;
    }

    server {
        listen 6379;
        proxy_pass cache;
    }
}
```

For TLS-only stream checks, omit send and expect directives to stop after the verified handshake.
With payload and expectation directives, the probe continues over TLS and checks the application response.
TLS healthcheck settings are independent of stream `proxy_ssl` settings for client traffic.

`healthcheck_send` and `healthcheck_expect` configure one request and one expected byte sequence.
The `_hex` variants provide the same settings for binary protocols.
An expectation succeeds when its bytes appear in the response after the `min_recv` threshold is met.
The default response buffer is 4096 bytes, and `max_response` can change its size.
A connection failure, timeout, early close, unmet expectation, or full unmatched buffer fails the check.
A send without an expectation or `min_recv` confirms transport and write success only.

The stream interval defaults to five seconds and its timeout defaults to one second.
The default failure threshold is three checks and the recovery threshold is two checks.
The default concurrency is 16 probes, the shared health state defaults to one mebibyte, and the default response buffer is 4096 bytes.

## DNS and peer lifecycle

Use NGINX's `resolve`, `zone`, and `resolver` directives for dynamic upstream names.
The checker walks NGINX's current peer lists, including resolved A and AAAA peers, backup peers, and SRV peers.
Each probe connects to the selected peer address that NGINX uses for client traffic.

New peers remain unavailable until they meet the configured recovery threshold.
Failure and recovery streaks belong to a peer instance instead of its position in the list.
Removed peers disappear from metrics, and results for removed instances are discarded.
Recreated peers start with fresh health state even when an address matches a removed peer.
An administratively configured `down` peer stays unavailable after successful checks.

Worker zero runs the probes and publishes health state for all workers through shared memory.
Worker replacement resumes checks, and a configuration reload creates fresh health state and resets counters.
New workers require the configured recovery threshold before admitting peers.
Graceful shutdown stops new probes and bounds active probes by their deadlines.

The checker requires NGINX's built-in round-robin-compatible peer storage.
Round robin, least connections, hash, random, and HTTP keepalive use this peer storage.
Third-party balancers with different peer storage are outside this target's supported profile.

## Metrics

`healthcheck_metrics` exports Prometheus metrics for HTTP and stream upstream peers from one HTTP location.
The metric labels include the upstream, configured server, resolved peer address, and backup role.
Use a separate VTS endpoint when you also need traffic metrics.

| Metric family | Examples |
| --- | --- |
| HTTP peer state | `nginx_healthcheck_peer_up`, `nginx_healthcheck_check_status`, and `nginx_healthcheck_check_code`. |
| Stream peer state | `nginx_stream_healthcheck_peer_up` and `nginx_stream_healthcheck_check_status`. |
| Per-peer counters | Completed checks, failed checks, and transitions from ready to down. |
| Per-peer timing | Last check duration, time since state change, and last check timestamp. |
| Upstream state | Active probes, scheduler freshness, configured concurrency, and checker errors. |

HTTP peer metrics use the `nginx_healthcheck_` prefix and stream peer metrics use `nginx_stream_healthcheck_`.
Both families include peer readiness, administrative state, checks, failures, transitions, result status, duration, state age, last check time, and rise and fall streaks.
HTTP metrics also expose the last parsed HTTP status code.
Upstream metrics expose the last peer scan, active probes, checker worker PID, concurrency, concurrency-limited scans, and memory failures.

An untested peer reports check status `-1` and must not be treated as a successful check.
Peer check status is `-1` before a result, `0` for failure, and `1` for success.
The status gauge does not classify failures by reason.
TLS-only checks report no HTTP response code because they do not send an HTTP request.
The transition counter records failures and resource failures that change a ready peer to down.
It does not count a successful recovery as another failure.

## HAProxy behavior comparison

This target covers connect checks and bounded send/expect checks for its supported peer model.
It does not claim full HAProxy healthcheck parity.

| Area | This target | HAProxy behavior outside this target |
| --- | --- | --- |
| L4 and L6 reachability | Checks connect to the actual selected NGINX peer address. | HAProxy exposes check transport controls such as [`check-ssl`](https://docs.haproxy.org/3.0/configuration.html#check-ssl). |
| TCP exchange | Supports one send payload, one expected byte sequence, and a minimum receive length. | HAProxy supports multiple TCP check steps through [`tcp-check connect`](https://docs.haproxy.org/3.0/configuration.html#tcp-check%20connect). |
| HTTP checks | Sends a GET request and checks accepted response status codes. | HAProxy also supports body and header expectations through [`http-check expect`](https://docs.haproxy.org/3.0/configuration.html#http-check%20expect). |
| Check routing | Checks the peer selected by NGINX's current peer list. | Independent check address and port overrides are not available here. |
| TLS controls | Verifies CA trust and hostname for TLS and HTTPS probes. | ALPN selection is not implemented here. |
| Protocol checks | Supports a single protocol payload and byte expectation. | Dedicated LDAP, MySQL, PostgreSQL, Redis, SMTP, and SPOP checks are not implemented here. |
| Check modes | Supports active checks. | Passive checks, agent checks, and external checks are outside this target. |

## Build verification

The Docker verify stage checks the ELF type and dynamic dependencies, prints `nginx -V`, and runs `nginx -t` on the scripting configuration.
HTTP and stream scripting responses report the actual `QuickJS` engine and selected njs version.
It starts the binary and uses `curl` for HTTP njs and Bash `/dev/tcp` for stream njs.
The healthcheck scenario generates a temporary certificate with OpenSSL and starts local NGINX, DNS, and socket fixtures.
It runs `nginx -t`, HTTP requests with `curl`, stream requests with Bash `/dev/tcp`, and OpenSSL-backed TLS probes.
The scenario covers TLS-only checks in HTTP and stream, wrong hostnames and CAs, TLS-to-plaintext failures, handshake timeouts, and failed L4 connections.
It also exercises IPv6 connectivity, multi-address DNS replacement, TCP send/expect routing, HTTP and TLS peer recovery, health metrics, and VTS metrics.

These checks run as part of `make build nginx-upstream-healthcheck`.
The packaged binary targets Linux and cannot run directly on macOS.

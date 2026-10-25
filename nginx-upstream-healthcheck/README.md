# NGINX Upstream Healthcheck

This target builds NGINX Open Source stable release 1.30.6 as a static musl PIE with active HTTP and stream health checks, njs, and VTS metrics.
The healthcheck code is compiled into the binary as `ngx_http_upstream_healthcheck_module`.

## Build and packaged files

Build this target with `make build nginx-upstream-healthcheck`.

Build metadata selects the NGINX, njs, QuickJS-NG, and VTS release versions.

| Component | Version | Purpose |
| --- | --- | --- |
| NGINX Open Source | 1.30.6 | Provides the HTTP and stream servers. |
| njs | 1.0.2 | Provides JavaScript scripting in HTTP and stream. |
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
- `README.md`

The packaged healthcheck configuration shows HTTP, HTTPS, TLS-only, stream TCP, TLS stream, and protected localhost metrics examples.
It uses `conf/backend-ca.pem` as an example trust bundle, which you must provide for your backends.
Replace its resolver and backend names for your environment before using it.
Canonical `metadata.json` selects the configuration and this README for release packaging.
The healthcheck profile needs no additional runtime library.

## njs scripting

Both NGINX targets include njs 1.0.2 for HTTP and stream scripting.
The release fixes XML security issues CVE-2026-101025 and CVE-2026-101019 and changes XML node-mutation compatibility and Fetch response validation.
Review the [official njs 1.0.2 release notes](https://nginx.org/en/docs/njs/changes.html#njs1.0.2) before upgrading existing scripts that use XML or Fetch.
Set `js_engine qjs;` in each context that imports an njs script.
The [official engine guidance](https://nginx.org/en/docs/njs/engine.html) recommends QuickJS for new configurations.
The packaged `conf/scripting.conf` and `conf/scripting.js` provide the scripting example.

The existing C module performs active probes against NGINX's selected peer and shares health state across workers.
The target already compiles njs with QuickJS-NG, but the available JavaScript APIs do not provide the peer and socket operations this checker needs.

| Capability | njs support | Fit for these checks |
| --- | --- | --- |
| Periodic execution | [`js_periodic`](https://nginx.org/en/docs/http/ngx_http_js_module.html#js_periodic) and the [stream equivalent](https://nginx.org/en/docs/stream/ngx_stream_js_module.html#js_periodic) run handlers on a schedule. | Scheduling exists, but a handler cannot select NGINX's current resolved upstream peer. |
| Shared state | [njs shared dictionaries](https://nginx.org/en/docs/njs/reference.html) share values between workers. | Shared storage does not provide peer selection, asynchronous socket operations, or healthcheck lifecycle hooks. |
| HTTPS requests | [`ngx.fetch()`](https://nginx.org/en/docs/njs/reference.html#ngx_fetch) supports asynchronous HTTP requests with TLS verification controls in [HTTP](https://nginx.org/en/docs/http/ngx_http_js_module.html) and [stream](https://nginx.org/en/docs/stream/ngx_stream_js_module.html) contexts. | It can check an HTTPS URL, but it does not target the exact resolved peer selected by NGINX and it sends an HTTP request. |
| Arbitrary TCP and TLS-only probes | Stream njs supports session callbacks and HTTPS fetches. | It exposes no general outbound TCP client or TLS-only client that stops after certificate verification. |
| Resolved peer and routing APIs | NGINX exposes upstream peer storage to native modules. | njs has no API for the actual resolved peer address or the peer used by client routing. |
| Native module loading | [`js_load_http_native_module`](https://nginx.org/en/docs/http/ngx_http_js_module.html#js_load_http_native_module) and [`js_load_stream_native_module`](https://nginx.org/en/docs/stream/ngx_stream_js_module.html#js_load_stream_native_module) load native modules. | A custom C module would still be required to add peer and socket APIs, so this does not reduce maintenance. |

The target uses QuickJS-NG 0.17.0 with njs 1.0.2 and links both into the static NGINX binary.
Using njs for the current probe contract would still require custom C integration for upstream peer selection and event-driven socket work.
The existing C implementation therefore remains the smaller maintained path for dynamic multi-address peers and arbitrary stream protocols.

## HTTP health checks

Configure `healthcheck` inside each checked HTTP `upstream` block.
The upstream requires a shared `zone` and a built-in round-robin-compatible peer list.

Use the HTTP examples in [upstream-healthcheck.conf](conf/upstream-healthcheck.conf).

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

Use the stream examples in [upstream-healthcheck.conf](conf/upstream-healthcheck.conf).

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
| HTTP peer state | `nginx_healthcheck_peer_up`, `nginx_healthcheck_check_status`, `nginx_healthcheck_check_code`, and `nginx_healthcheck_check_result`. |
| Stream peer state | `nginx_stream_healthcheck_peer_up`, `nginx_stream_healthcheck_check_status`, and `nginx_stream_healthcheck_check_result`. |
| Per-peer counters | Completed checks, failed checks, and transitions from ready to down. |
| Per-peer timing | Last check duration, time since state change, and last check timestamp. |
| Upstream state | Active probes, scheduler freshness, configured concurrency, and checker errors. |

HTTP peer metrics use the `nginx_healthcheck_` prefix and stream peer metrics use `nginx_stream_healthcheck_`.
Both families include peer readiness, administrative state, checks, failures, transitions, one-hot result classification, duration, state age, last check time, and rise and fall streaks.
HTTP metrics also expose the last parsed HTTP status code.
Upstream metrics expose the last peer scan, active probes, checker worker PID, concurrency, concurrency-limited scans, and memory failures.

An untested peer reports check status `-1` and must not be treated as a successful check.
Peer check status is `-1` before a result, `0` for failure, and `1` for success.
This boolean describes the last completed probe, while `peer_up` describes current routing eligibility.
Local resource failures or shutdown aborts can classify the peer as `SOCKERR` while retaining its previous completed boolean result.
The `check_result` metric families emit one series for each result name on every peer, with one current result set to `1` and all others set to `0`.
The older boolean status and HTTP code gauges remain available.
TLS-only checks report no HTTP response code because they do not send an HTTP request.
The transition counter records failures and resource failures that change a ready peer to down.
It does not count a successful recovery as another failure.

| Result | Meaning |
| --- | --- |
| `UNKNOWN` | No probe or local failure has classified the peer yet. |
| `SOCKERR` | The checker could not create or operate its local socket or event state. |
| `L4OK` | The TCP connection succeeded for a connect-only check. |
| `L4TOUT` | The TCP connection did not complete before its timeout. |
| `L4CON` | The TCP connection failed, including a refused connection. |
| `L6OK` | The TLS handshake, trusted certificate chain, and hostname verification succeeded. |
| `L6TOUT` | The TLS handshake did not complete before its timeout. |
| `L6RSP` | The TLS handshake or certificate/name verification failed. |
| `L7OK` | The HTTP status was accepted, the stream response requirement was met, or a send-only payload was written. |
| `L7TOUT` | The HTTP or stream application exchange did not complete before its timeout. |
| `L7RSP` | The HTTP response was malformed or missing, or a stream response ended without a match. |
| `L7STS` | The HTTP response was valid but its status was not configured as accepted. |

## HAProxy behavior comparison

This repository tracks HAProxy 3.4.7 on the 3.4 series.
The result names follow layer distinctions documented by the [HAProxy 3.4 configuration manual](https://docs.haproxy.org/3.4/configuration.html).
The NGINX results classify the point where each probe succeeded or failed while retaining NGINX's selected-peer routing.

| Area | This target | HAProxy 3.4 behavior |
| --- | --- | --- |
| L4 and L6 | `L4OK`, `L4TOUT`, `L4CON`, `L6OK`, `L6TOUT`, and `L6RSP` distinguish connect and verified TLS outcomes. | HAProxy's [`tcp-check expect`](https://docs.haproxy.org/3.4/configuration.html#tcp-check%20expect) exposes corresponding L4 and L6 status classes. |
| L7 | `L7OK`, `L7TOUT`, `L7RSP`, and `L7STS` distinguish application success, timeout, response failure, and rejected HTTP status. | HAProxy exposes L7 check statuses and configurable status, body, and header expectations. |
| Peer routing | Checks each peer from NGINX's current upstream peer list. | HAProxy supports explicit check addresses, ports, connection steps, and [`check-ssl`](https://docs.haproxy.org/3.4/configuration.html#check-ssl) options. |
| TLS verification | Requires a trusted CA and hostname for TLS and HTTPS checks. | HAProxy exposes certificate verification, hostname checks, and separate SNI controls. |
| TCP exchange | Supports one send payload, one expected byte sequence, and a minimum receive length. | HAProxy supports multiple TCP check steps and protocol-specific check rules. |
| Scope | Supports active HTTP, HTTPS, TCP, TLS-only, and bounded stream send/expect checks. | This target does not claim complete HAProxy healthcheck parity. |

## Build verification

Both NGINX Docker verify stages reference shared scripts for common static-binary checks and njs HTTP and stream checks.
Build scripts and fixture configurations live in separate source files rather than embedded Dockerfile or documentation scripts.
The source-checkout verification commands below use scripts and fixtures that are not included in release packages.
The shared `nginx/nginx-verify.sh` script preserves the existing static-build verification contract.
Each HTTP probe has a 15-second GNU `timeout` deadline, with curl request limits of two seconds for the retrying probe and five seconds for the second probe.
On success, failure, `INT`, or `TERM`, it cancels active probes and forwards `QUIT` to the supervised nginx process group.
GNU `timeout` escalates resistant probes to `KILL` after one second and resistant nginx processes after two seconds.
The verifier reaps both supervisors and removes its temporary response file.
The nginx supervisor also imposes a 40-second server lifetime bound.
Run `python3 proxy/tests/verify-nginx.py` from the repository root to check cancellation, failed probes, deadlines, and shutdown with local process fixtures.
This standalone regression requires Python 3 and GNU Coreutils `timeout` or `gtimeout` and does not invoke Docker.
It does not run healthcheck functional scenarios.
The standalone suite lives under `proxy/tests/`, which is a test layout rather than a build target.
The `proxy/tests/fixtures/nginx.conf.template` and `proxy/tests/fixtures/ca.cnf.template` files provide separate fixture sources.

Validate runner cleanup offline with `python3 proxy/tests/verify-runner.py`.
Check shell syntax with `bash -n proxy/tests/run.sh proxy/tests/run-docker.sh`.
Run `make lint` and `make shfmt-check` for shell lint and formatting.

The Python 3 check uses a local Docker CLI fixture to exercise partial creation, signal cleanup, ownership guards, and exit-status retention.
It runs offline without a Docker daemon.

Run `proxy/tests/run.sh` against an already-built Linux binary with `proxy/tests/run.sh .out/nginx-upstream-healthcheck/sbin/nginx`.

The local runner needs Bash, core utilities, grep, sed, curl, dnsmasq, OpenSSL, `setsid`, and socat.
It creates loopback fixtures and certificate files under a temporary directory, then removes the fixture process groups and files on exit or signal.
The fixtures cover HTTP and stream L4 connection results, verified TLS-only checks with no client application bytes, HTTPS status and timeout results, TLS protocol and certificate failures, stream TLS payload expectations, dynamic DNS replacement, fall/rise and recovery, IPv6, metrics visibility across workers, and VTS metrics.
The suite also checks that HTTP and stream TLS configurations fail when either the TLS name or trusted CA is missing.
It validates literal HTTP and stream proxy destinations that create implicit upstreams without healthcheck settings.

On macOS or another host without the required Linux runtime tools, use `proxy/tests/run-docker.sh`.
Run `proxy/tests/run-docker.sh .out/nginx-upstream-healthcheck/sbin/nginx`.

The wrapper reads `UBI10_MINIMAL_VERSION` from `metadata.json` and passes the matching `registry.access.redhat.com/ubi10/ubi-minimal` image to the test build.
It mounts the binary read-only and creates one uniquely named container with a four-CPU and eight-GiB cap.
It defaults to the repository's `linux/amd64` build platform.
Pass `--platform linux/arm64` when the binary was built for another platform.
It removes its container and image on normal completion or interruption.
Neither runner is part of `make build nginx-upstream-healthcheck`, so build verification does not imply functional-suite acceptance.

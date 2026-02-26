# Lua Runtime Modules

This build keeps nginx and C modules statically linked, and ships Lua modules as runtime files inside the release artifact/image.

## What Is Shipped

The build installs Lua runtime files under `nginx/lualib`:

- `nginx/lualib/resty/core.lua`
- `nginx/lualib/resty/core/`
- `nginx/lualib/resty/upstream/healthcheck.lua`

`LUA_PATH` is configured to load modules from `${TARGET_PREFIX}/lualib` at runtime.

## Build and Runtime Model

1. nginx is built as static PIE with statically linked C dependencies and nginx modules.
2. `lua-resty-core` and `lua-resty-upstream-healthcheck` Lua files are copied into `${TARGET_PREFIX}/lualib` during image build.
3. Runtime `require "resty.upstream.healthcheck"` loads from packaged Lua files instead of embedded bytecode.

## Scope

This target packages the runtime Lua modules required for `resty.upstream.healthcheck` operation while preserving static nginx/module builds.

## Prefix and Path Resolution

- `-p <prefix>` sets the nginx runtime prefix.
- If `-c` is omitted, nginx loads `${prefix}/conf/nginx.conf` by default.
- `nginx.conf` cannot set prefix itself like `-p`; it can only reference `$prefix`.
- Relative `lua_package_path` entries are not evaluated from binary path or `nginx.conf` file path.
- `lua_package_path` default value follows `LUA_PATH` or Lua compiled-in defaults.
- `;;` appends those default search paths after your custom path.

Use `$prefix` in `nginx.conf` for portable module loading:

```nginx
http {
    lua_package_path "$prefix/lualib/?.lua;$prefix/lualib/?/init.lua;;";
}
```

Portable bundle example:

```text
/opt/mybundle/
  conf/nginx.conf
  lualib/resty/...
  sbin/nginx
```

```bash
/opt/mybundle/sbin/nginx -p /opt/mybundle -c conf/nginx.conf
```

## Comprehensive Runtime Example

Use this single `http {}` example for multiple upstream checks, automatic peer down/up handling, VTS traffic stats, and merged Prometheus output.

```nginx
http {
    lua_shared_dict healthcheck 64m;
    vhost_traffic_status_filter_by_host on;
    vhost_traffic_status_zone shared:vhost_traffic_status:64m;
    upstream foo {
        server 10.0.1.11:8080;
        server 10.0.1.12:8080;
    }
    upstream bar {
        server 10.0.2.11:8080;
        server 10.0.2.12:8080;
    }
    init_worker_by_lua_block {
        local hc = require "resty.upstream.healthcheck"
        local ok, err = hc.spawn_checker({
            shm = "healthcheck",
            upstream = "foo",
            type = "http",
            http_req = "GET /readyz HTTP/1.0\r\nHost: foo.internal\r\n\r\n",
            interval = 2000,
            timeout = 1000,
            fall = 3,
            rise = 2,
            valid_statuses = {200, 204},
            concurrency = 10,
        })
        if not ok then
            ngx.log(ngx.ERR, "failed to spawn health checker for foo: ", err)
        end
        ok, err = hc.spawn_checker({
            shm = "healthcheck",
            upstream = "bar",
            type = "https",
            http_req = "GET /healthz HTTP/1.0\r\nHost: bar.internal\r\n\r\n",
            interval = 5000,
            timeout = 2000,
            fall = 5,
            rise = 3,
            valid_statuses = {200},
            concurrency = 5,
            ssl_verify = true,
            host = "bar.internal",
        })
        if not ok then
            ngx.log(ngx.ERR, "failed to spawn health checker for bar: ", err)
        end
    }
    server {
        listen 80;
        location /api/ {
            proxy_pass http://foo;
        }
        location / {
            proxy_pass http://bar;
        }
    }
    server {
        listen 18080;
        location = /status/format/prometheus {
            internal;
            vhost_traffic_status_display;
            vhost_traffic_status_display_format prometheus;
        }
        location = /status {
            access_log off;
            default_type text/plain;
            content_by_lua_block {
                local hc = require "resty.upstream.healthcheck"
                ngx.print(hc.status_page())
            }
        }
        location = /metrics {
            access_log off;
            default_type text/plain;
            content_by_lua_block {
                local chunks = {}
                local healthcheck_up = 0
                local vts_up = 0
                local ok_mod, hc_or_err = pcall(require, "resty.upstream.healthcheck")
                if ok_mod then
                    local hc_out, hc_err = hc_or_err.prometheus_status_page()
                    if hc_out then
                        healthcheck_up = 1
                        table.insert(chunks, hc_out)
                    else
                        table.insert(chunks, "# healthcheck module unavailable: " .. tostring(hc_err))
                    end
                else
                    table.insert(chunks, "# healthcheck module unavailable: " .. tostring(hc_or_err))
                end
                local vts = ngx.location.capture("/status/format/prometheus")
                if vts.status == ngx.HTTP_OK then
                    vts_up = 1
                    table.insert(chunks, vts.body)
                else
                    table.insert(chunks, "# vts metrics unavailable: status=" .. tostring(vts.status))
                end
                table.insert(chunks, "# HELP nginx_metrics_source_up Source availability for merged /metrics endpoint")
                table.insert(chunks, "# TYPE nginx_metrics_source_up gauge")
                table.insert(chunks, string.format("nginx_metrics_source_up{source=\"healthcheck_module\"} %d", healthcheck_up))
                table.insert(chunks, string.format("nginx_metrics_source_up{source=\"vts\"} %d", vts_up))
                ngx.print(table.concat(chunks, "\n"))
            }
        }
    }
}
```

`spawn_checker` updates peer state via `ngx.upstream.set_peer_down`. Failed peers are marked down after `fall` consecutive failures and recovered after `rise` consecutive successes.

`nginx_metrics_source_up` indicates metric source availability, not backend health status.

## Reference

- [lua-nginx-module: lua_package_path](https://github.com/openresty/lua-nginx-module#lua_package_path)
- [lua-resty-upstream-healthcheck](https://github.com/openresty/lua-resty-upstream-healthcheck)
- [lua-resty-core](https://github.com/openresty/lua-resty-core)

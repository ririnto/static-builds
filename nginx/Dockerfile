ARG ALPINE_VERSION
ARG UBI9_MICRO_VERSION
ARG TARGET_PREFIX=/home/nobody


FROM alpine:${ALPINE_VERSION} AS build
ARG NGINX_VERSION
ARG NGINX_VTS_MODULE_VERSION
ARG NGINX_LUA_RESTY_CORE_VERSION
ARG NGINX_LUA_MODULE_VERSION
ARG NGINX_LUA_UPSTREAM_MODULE_VERSION
ARG NGINX_LUA_RESTY_UPSTREAM_HEALTHCHECK_MODULE_VERSION
ARG TARGET_PREFIX

WORKDIR /build
RUN --mount=type=cache,target=/var/cache/apk \
    apk add \
        build-base \
        libatomic_ops-dev \
        libatomic_ops-static \
        linux-headers \
        luajit \
        luajit-dev \
        openssl-dev \
        openssl-libs-static \
        pcre2-dev \
        pcre2-static \
        zlib-dev \
        zlib-static
ADD "src/nginx-${NGINX_VERSION}.tar.gz" /build/
ADD "src/nginx-module-vts-${NGINX_VTS_MODULE_VERSION}.tar.gz" "/build/nginx-${NGINX_VERSION}/"
ADD "src/lua-nginx-module-${NGINX_LUA_MODULE_VERSION}.tar.gz" "/build/nginx-${NGINX_VERSION}/"
ADD "src/lua-resty-core-${NGINX_LUA_RESTY_CORE_VERSION}.tar.gz" "/build/"
ADD "src/lua-upstream-nginx-module-${NGINX_LUA_UPSTREAM_MODULE_VERSION}.tar.gz" "/build/nginx-${NGINX_VERSION}/"
ADD "src/lua-resty-upstream-healthcheck-${NGINX_LUA_RESTY_UPSTREAM_HEALTHCHECK_MODULE_VERSION}.tar.gz" "/build/"
WORKDIR "/build/nginx-${NGINX_VERSION}"
RUN LUAJIT_LIB='/usr/lib' LUAJIT_INC='/usr/include/luajit-2.1' ./configure \
    # --help \
    --prefix=${TARGET_PREFIX} \
    # --sbin-path=PATH \
    # --modules-path=PATH \
    # --conf-path=PATH \
    # --error-log-path=PATH \
    # --pid-path=PATH \
    # --lock-path=PATH \
    # --user=USER \
    # --group=GROUP \
    # --build=NAME \
    # --builddir=DIR \
    # --with-select_module \
    # --without-select_module \
    # --with-poll_module \
    # --without-poll_module \
    --with-threads \
    --with-file-aio \
    # --without-quic_bpf_module \
    --with-http_ssl_module \
    --with-http_v2_module \
    --with-http_v3_module \
    --with-http_realip_module \
    # --with-http_addition_module \
    # --with-http_xslt_module \
    # --with-http_xslt_module=dynamic \
    # --with-http_image_filter_module \
    # --with-http_image_filter_module=dynamic \
    # --with-http_geoip_module \
    # --with-http_geoip_module=dynamic \
    # --with-http_sub_module \
    # --with-http_dav_module \
    # --with-http_flv_module \
    # --with-http_mp4_module \
    # --with-http_gunzip_module \
    --with-http_gzip_static_module \
    # --with-http_auth_request_module \
    # --with-http_random_index_module \
    # --with-http_secure_link_module \
    # --with-http_degradation_module \
    # --with-http_slice_module \
    --with-http_stub_status_module \
    # --without-http_charset_module \
    # --without-http_gzip_module \
    --without-http_ssi_module \
    --without-http_userid_module \
    # --without-http_access_module \
    # --without-http_auth_basic_module \
    # --without-http_mirror_module \
    --without-http_autoindex_module \
    # --without-http_geo_module \
    # --without-http_map_module \
    --without-http_split_clients_module \
    # --without-http_referer_module \
    # --without-http_rewrite_module \
    # --without-http_proxy_module \
    --without-http_fastcgi_module \
    --without-http_uwsgi_module \
    --without-http_scgi_module \
    # --without-http_grpc_module \
    --without-http_memcached_module \
    # --without-http_limit_conn_module \
    # --without-http_limit_req_module \
    --without-http_empty_gif_module \
    --without-http_browser_module \
    # --without-http_upstream_hash_module \
    # --without-http_upstream_ip_hash_module \
    # --without-http_upstream_least_conn_module \
    # --without-http_upstream_random_module \
    # --without-http_upstream_keepalive_module \
    # --without-http_upstream_zone_module \
    # --with-http_perl_module \
    # --with-http_perl_module=dynamic \
    # --with-perl_modules_path=PATH \
    # --with-perl=PATH \
    # --http-log-path=PATH \
    # --http-client-body-temp-path=PATH \
    # --http-proxy-temp-path=PATH \
    # --http-fastcgi-temp-path=PATH \
    # --http-uwsgi-temp-path=PATH \
    # --http-scgi-temp-path=PATH \
    # --without-http \
    # --without-http-cache \
    # --with-mail \
    # --with-mail=dynamic \
    # --with-mail_ssl_module \
    # --without-mail_pop3_module \
    # --without-mail_imap_module \
    # --without-mail_smtp_module \
    --with-stream \
    # --with-stream=dynamic \
    --with-stream_ssl_module \
    --with-stream_realip_module \
    # --with-stream_geoip_module \
    # --with-stream_geoip_module=dynamic \
    --with-stream_ssl_preread_module \
    # --without-stream_limit_conn_module \
    # --without-stream_access_module \
    # --without-stream_geo_module \
    # --without-stream_map_module \
    # --without-stream_split_clients_module \
    # --without-stream_return_module \
    # --without-stream_pass_module \
    # --without-stream_set_module \
    # --without-stream_upstream_hash_module \
    # --without-stream_upstream_least_conn_module \
    # --without-stream_upstream_random_module \
    # --without-stream_upstream_zone_module \
    # --with-google_perftools_module \
    # --with-cpp_test_module \
    # --add-module=PATH \
    --add-module="nginx-module-vts-${NGINX_VTS_MODULE_VERSION}" \
    --add-module="lua-nginx-module-${NGINX_LUA_MODULE_VERSION}" \
    --add-module="lua-upstream-nginx-module-${NGINX_LUA_UPSTREAM_MODULE_VERSION}" \
    # --add-dynamic-module=PATH \
    # --with-compat \
    # --with-cc=PATH \
    # --with-cpp=PATH \
    --with-cc-opt="-O2 -pipe -fPIE -fstack-protector-strong -fstack-clash-protection -ffunction-sections -fdata-sections -fno-delete-null-pointer-checks -fno-strict-overflow -fno-strict-aliasing -ftrivial-auto-var-init=zero -Wformat -Wformat=2 -Werror=format-security" \
    --with-ld-opt="-static -static-pie -static-libgcc -Wl,-E -rdynamic -Wl,-z,relro -Wl,-z,now -Wl,-z,noexecstack -Wl,-z,separate-code -Wl,--as-needed" \
    # --with-cpu-opt=CPU \
    # --without-pcre \
    # --with-pcre \
    # --with-pcre=DIR \
    # --with-pcre-opt=OPTIONS \
    # --with-pcre-jit \
    # --without-pcre2 \
    # --with-zlib=DIR \
    # --with-zlib-opt=OPTIONS \
    # --with-zlib-asm=CPU \
    --with-libatomic \
    # --with-libatomic=DIR \
    # --with-openssl=DIR \
    # --with-openssl-opt=OPTIONS \
    # --with-debug \
    ;
RUN make "-j$(nproc)"
RUN make install
RUN mkdir -p "${TARGET_PREFIX}/lualib" && \
    cp -r "/build/lua-resty-core-${NGINX_LUA_RESTY_CORE_VERSION}/lib/resty" \
          "${TARGET_PREFIX}/lualib/resty" && \
    mkdir -p "${TARGET_PREFIX}/lualib/resty/upstream" && \
    cp "/build/lua-resty-upstream-healthcheck-${NGINX_LUA_RESTY_UPSTREAM_HEALTHCHECK_MODULE_VERSION}/lib/resty/upstream/healthcheck.lua" \
       "${TARGET_PREFIX}/lualib/resty/upstream/healthcheck.lua" && \
    cat > "${TARGET_PREFIX}/lualib/resty/core.lua" <<'EOF'
local ngx = ngx

local function patch_shared_dict()
    if type(ngx) ~= "table" or type(ngx.shared) ~= "table" then
        return
    end

    local _, any_dict = next(ngx.shared)
    local mt = any_dict and getmetatable(any_dict)
    local dict_mt = mt and mt.__index
    if type(dict_mt) ~= "table" then
        return
    end

    if type(dict_mt.get) == "function"
        and type(dict_mt.set) == "function"
        and type(dict_mt.add) == "function"
        and type(dict_mt.incr) == "function"
    then
        return
    end

    local now = ngx.now or os.time
    local sep = "|"

    local function normalize_key(key)
        if key == nil then
            return nil, "nil key"
        end
        if type(key) ~= "string" then
            key = tostring(key)
        end
        if #key == 0 then
            return nil, "empty key"
        end
        if #key > 65535 then
            return nil, "key too long"
        end
        return key
    end

    local function clear_key(dict, key)
        while true do
            local v = dict:lpop(key)
            if v == nil then
                return
            end
        end
    end

    local function encode_value(value, exptime)
        local exp = 0
        if type(exptime) == "number" and exptime > 0 then
            exp = now() + exptime
        end

        if value == nil then
            return "nil" .. sep .. tostring(exp) .. sep
        end

        local t = type(value)
        if t == "number" then
            return "num" .. sep .. tostring(exp) .. sep .. tostring(value)
        end
        if t == "boolean" then
            return "bool" .. sep .. tostring(exp) .. sep .. (value and "1" or "0")
        end
        if t == "string" then
            return "str" .. sep .. tostring(exp) .. sep .. value
        end

        return nil, "bad value type"
    end

    local function decode_value(raw)
        local p1 = string.find(raw, sep, 1, true)
        if not p1 then
            return nil, "bad value"
        end
        local p2 = string.find(raw, sep, p1 + 1, true)
        if not p2 then
            return nil, "bad value"
        end

        local typ = string.sub(raw, 1, p1 - 1)
        local exp = tonumber(string.sub(raw, p1 + 1, p2 - 1)) or 0
        local payload = string.sub(raw, p2 + 1)

        if exp > 0 and now() > exp then
            return nil, "expired"
        end

        if typ == "nil" then
            return nil
        end
        if typ == "num" then
            local n = tonumber(payload)
            if n == nil then
                return nil, "bad number"
            end
            return n
        end
        if typ == "bool" then
            return payload == "1"
        end
        if typ == "str" then
            return payload
        end

        return nil, "bad type"
    end

    local function dict_get(dict, key)
        local key_norm, err = normalize_key(key)
        if not key_norm then
            return nil, err
        end

        local raw = dict:lpop(key_norm)
        if raw == nil then
            return nil
        end

        local val, derr = decode_value(raw)
        if derr == "expired" then
            clear_key(dict, key_norm)
            return nil
        end
        if derr then
            return nil, derr
        end

        dict:lpush(key_norm, raw)
        return val
    end

    local function dict_set(dict, key, value, exptime)
        local key_norm, err = normalize_key(key)
        if not key_norm then
            return nil, err
        end

        clear_key(dict, key_norm)
        if value == nil then
            return true
        end

        local encoded, enc_err = encode_value(value, exptime)
        if not encoded then
            return nil, enc_err
        end

        local ok, push_err = dict:rpush(key_norm, encoded)
        if not ok then
            return nil, push_err
        end

        return true
    end

    local function dict_add(dict, key, value, exptime)
        local existing, get_err = dict_get(dict, key)
        if existing ~= nil then
            return nil, "exists"
        end
        if get_err and get_err ~= "expired" then
            return nil, get_err
        end

        return dict_set(dict, key, value, exptime)
    end

    local function dict_incr(dict, key, step)
        if type(step) ~= "number" then
            step = tonumber(step)
        end
        if type(step) ~= "number" then
            return nil, "bad increment value"
        end

        local current, err = dict_get(dict, key)
        if current == nil then
            if err then
                return nil, err
            end
            return nil, "not found"
        end
        if type(current) ~= "number" then
            return nil, "not a number"
        end

        local next_value = current + step
        local ok, set_err = dict_set(dict, key, next_value, 0)
        if not ok then
            return nil, set_err
        end
        return next_value
    end

    dict_mt.get = dict_get
    dict_mt.set = dict_set
    dict_mt.add = dict_add
    dict_mt.incr = dict_incr
    dict_mt.delete = function(dict, key)
        return dict_set(dict, key, nil, 0)
    end
end

local function patch_re_find()
    if type(ngx) ~= "table" then
        return
    end

    local re = ngx.re
    if type(re) ~= "table" then
        re = {}
        ngx.re = re
    end

    if type(re.find) == "function" then
        return
    end

    re.find = function(subject, regex, _, _, nth)
        if type(subject) ~= "string" then
            return nil, nil, "subject is not a string"
        end
        if nth ~= nil and nth ~= 1 then
            return nil, nil, "unsupported capture index"
        end

        if regex == [[^HTTP/\d+\.\d+\s+(\d+)]] then
            local _, end_pos, code = string.find(subject, "^HTTP/%d+%.%d+%s+(%d+)")
            if not end_pos then
                return nil
            end
            local cap_from = end_pos - #code + 1
            return cap_from, end_pos
        end

        if regex == [[^(.*):\d+$]] then
            local _, _, host = string.find(subject, "^(.*):%d+$")
            if not host then
                return nil
            end
            return 1, #host
        end

        return nil, nil, "unsupported regex"
    end
end

patch_shared_dict()
patch_re_find()

return {
    version = "runtime-minimal",
}
EOF


FROM redhat/ubi9-minimal:${UBI9_MICRO_VERSION} AS verify
ARG TARGET_PREFIX
ENV LUA_PATH="${TARGET_PREFIX}/lualib/?.lua;${TARGET_PREFIX}/lualib/?/init.lua;;"

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
COPY upstream-healthcheck.conf /tmp/upstream-healthcheck.conf
RUN set -eu; \
    echo "=== ELF Header ==="; \
    readelf -h ${TARGET_PREFIX}/sbin/nginx | egrep 'Type:|Machine:'; \
    echo "=== Program Headers ==="; \
    readelf -l ${TARGET_PREFIX}/sbin/nginx | egrep 'INTERP|DYNAMIC' || true; \
    echo "=== Dynamic Section ==="; \
    readelf -d ${TARGET_PREFIX}/sbin/nginx || true; \
    file_output="$(file ${TARGET_PREFIX}/sbin/nginx 2>&1 || true)"; \
    ldd_output="$(ldd ${TARGET_PREFIX}/sbin/nginx 2>&1 || true)"; \
    echo "=== file output ==="; echo "$file_output"; \
    echo "=== ldd output ==="; echo "$ldd_output"; \
    (echo "$file_output" | grep -Eq 'statically linked|static-pie linked') || (echo "Error: file(1) did not report a static binary." && exit 1); \
    (echo "$ldd_output" | grep -Eqi 'statically linked|static-pie linked|not a dynamic executable|not a valid dynamic program') || (echo "Error: Dynamic libraries found. This should be a static build!" && exit 1); \
    strace ${TARGET_PREFIX}/sbin/nginx -V; \
    echo "=== Packaging Check: lua runtime files installed ==="; \
    test ! -d ${TARGET_PREFIX}/lib/lua || (echo "Error: lib/lua directory should not exist" && exit 1); \
    test -f ${TARGET_PREFIX}/lualib/resty/core.lua || (echo "Error: missing lua-resty-core entrypoint" && exit 1); \
    test -f ${TARGET_PREFIX}/lualib/resty/upstream/healthcheck.lua || (echo "Error: missing lua-resty-upstream-healthcheck module" && exit 1)


FROM scratch

ARG TARGET_PREFIX
ENV LUA_PATH="${TARGET_PREFIX}/lualib/?.lua;${TARGET_PREFIX}/lualib/?/init.lua;;"

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 80 443
ENTRYPOINT ["./sbin/nginx"]

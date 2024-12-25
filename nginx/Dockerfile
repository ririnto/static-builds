# syntax=docker/dockerfile:1.4
ARG ALPINE_VERSION
ARG ALPINE_CURL_VERSION
ARG NGINX_VERSION
ARG NGINX_MODULE_VTS_VERSION
ARG UBI9_MICRO_VERSION

FROM alpine/curl:${ALPINE_CURL_VERSION} AS nginx-downloader
ARG NGINX_VERSION
WORKDIR /tmp
RUN --mount=type=bind,target=/tmp/mount \
    if [ -f "/tmp/mount/nginx-${NGINX_VERSION}.tar.gz" ]; then cat "/tmp/mount/nginx-${NGINX_VERSION}.tar.gz" | tar -zx; \
    else curl -L "https://nginx.org/download/nginx-${NGINX_VERSION}.tar.gz" | tar -zx; fi

FROM alpine/curl:${ALPINE_CURL_VERSION} AS nginx-module-vts-downloader
ARG NGINX_MODULE_VTS_VERSION
WORKDIR /tmp
RUN --mount=type=bind,target=/tmp/mount \
    if [ -f "/tmp/mount/nginx-module-vts-${NGINX_MODULE_VTS_VERSION}.tar.gz" ]; then cat "/tmp/mount/nginx-module-vts-${NGINX_MODULE_VTS_VERSION}.tar.gz" | tar -zx; \
    else curl -L "https://github.com/vozlt/nginx-module-vts/archive/refs/tags/v${NGINX_MODULE_VTS_VERSION}.tar.gz" | tar -zx; fi

FROM alpine:${ALPINE_VERSION} AS builder
ARG NGINX_VERSION
ARG NGINX_MODULE_VTS_VERSION
WORKDIR /tmp
RUN --mount=type=cache,target=/var/cache/apk \
    apk add \
        build-base \
        geoip-dev \
        libatomic_ops-dev \
        libatomic_ops-static \
        linux-headers \
        openssl-dev \
        openssl-libs-static \
        pcre2-dev \
        zlib-dev \
        zlib-static
COPY --from=nginx-downloader "/tmp/nginx-${NGINX_VERSION}" "/tmp/nginx-${NGINX_VERSION}"
COPY --from=nginx-module-vts-downloader "/tmp/nginx-module-vts-${NGINX_MODULE_VTS_VERSION}" "/tmp/nginx-${NGINX_VERSION}/nginx-module-vts-${NGINX_MODULE_VTS_VERSION}"
RUN cd "/tmp/nginx-${NGINX_VERSION}" && \
    ./configure \
        # prints a help message.
        # --help \
        # defines a directory that will keep server files. This same directory will also be used for all relative paths set by configure (except for paths to libraries sources) and in the nginx.conf configuration file. It is set to the /usr/local/nginx directory by default.
        # --prefix=path \
        # sets the name of an nginx executable file. This name is used only during installation. By default the file is named prefix/sbin/nginx.
        # --sbin-path=path \
        # defines a directory where nginx dynamic modules will be installed. By default the prefix/modules directory is used.
        # --modules-path=path \
        # sets the name of an nginx.conf configuration file. If needs be, nginx can always be started with a different configuration file, by specifying it in the command-line parameter -c file. By default the file is named prefix/conf/nginx.conf.
        # --conf-path=path \
        # sets the name of the primary error, warnings, and diagnostic file. After installation, the file name can always be changed in the nginx.conf configuration file using the error_log directive. By default the file is named prefix/logs/error.log.
        # --error-log-path=path \
        # sets the name of an nginx.pid file that will store the process ID of the main process. After installation, the file name can always be changed in the nginx.conf configuration file using the pid directive. By default the file is named prefix/logs/nginx.pid.
        # --pid-path=path \
        # sets a prefix for the names of lock files. After installation, the value can always be changed in the nginx.conf configuration file using the lock_file directive. By default the value is prefix/logs/nginx.lock.
        # --lock-path=path \
        # sets the name of an unprivileged user whose credentials will be used by worker processes. After installation, the name can always be changed in the nginx.conf configuration file using the user directive. The default user name is nobody.
        # --user=name \
        # sets the name of a group whose credentials will be used by worker processes. After installation, the name can always be changed in the nginx.conf configuration file using the user directive. By default, a group name is set to the name of an unprivileged user.
        # --group=name \
        # sets an optional nginx build name.
        # --build=name \
        # sets a build directory.
        # --builddir=path \
        # enables or disables building a module that allows the server to work with the select() method. This module is built automatically if the platform does not appear to support more appropriate methods such as kqueue, epoll, or /dev/poll.
        --with-select_module \
        # --without-select_module \
        # enables or disables building a module that allows the server to work with the poll() method. This module is built automatically if the platform does not appear to support more appropriate methods such as kqueue, epoll, or /dev/poll.
        --with-poll_module \
        # --without-poll_module \
        # enables the use of thread pools.
        --with-threads \
        # enables the use of asynchronous file I/O (AIO) on FreeBSD and Linux.
        --with-file-aio \
        # enables building a module that adds the HTTPS protocol support to an HTTP server. This module is not built by default. The OpenSSL library is required to build and run this module.
        --with-http_ssl_module \
        # enables building a module that provides support for HTTP/2. This module is not built by default.
        --with-http_v2_module \
        # enables building a module that provides support for HTTP/3. This module is not built by default. An SSL library that provides HTTP/3 support is recommended to build and run this module, such as BoringSSL, LibreSSL, or QuicTLS. Otherwise, if using the OpenSSL library, OpenSSL compatibility layer will be used that does not support QUIC early data.
        --with-http_v3_module \
        # enables building the ngx_http_realip_module module that changes the client address to the address sent in the specified header field. This module is not built by default.
        --with-http_realip_module \
        # enables building the ngx_http_addition_module module that adds text before and after a response. This module is not built by default.
        --with-http_addition_module \
        # enables building the ngx_http_xslt_module module that transforms XML responses using one or more XSLT stylesheets. This module is not built by default. The libxml2 and libxslt libraries are required to build and run this module.
        # --with-http_xslt_module \
        # --with-http_xslt_module=dynamic \
        # enables building the ngx_http_image_filter_module module that transforms images in JPEG, GIF, PNG, and WebP formats. This module is not built by default.
        # --with-http_image_filter_module \
        # --with-http_image_filter_module=dynamic \
        # enables building the ngx_http_geoip_module module that creates variables depending on the client IP address and the precompiled MaxMind databases. This module is not built by default.
        --with-http_geoip_module \
        # --with-http_geoip_module=dynamic \
        # enables building the ngx_http_sub_module module that modifies a response by replacing one specified string by another. This module is not built by default.
        --with-http_sub_module \
        # enables building the ngx_http_dav_module module that provides file management automation via the WebDAV protocol. This module is not built by default.
        --with-http_dav_module \
        # enables building the ngx_http_flv_module module that provides pseudo-streaming server-side support for Flash Video (FLV) files. This module is not built by default.
        --with-http_flv_module \
        # enables building the ngx_http_mp4_module module that provides pseudo-streaming server-side support for MP4 files. This module is not built by default.
        --with-http_mp4_module \
        # enables building the ngx_http_gunzip_module module that decompresses responses with “Content-Encoding: gzip” for clients that do not support “gzip” encoding method. This module is not built by default.
        --with-http_gunzip_module \
        # enables building the ngx_http_gzip_static_module module that enables sending precompressed files with the “.gz” filename extension instead of regular files. This module is not built by default.
        --with-http_gzip_static_module \
        # enables building the ngx_http_auth_request_module module that implements client authorization based on the result of a subrequest. This module is not built by default.
        --with-http_auth_request_module \
        # enables building the ngx_http_random_index_module module that processes requests ending with the slash character (‘/’) and picks a random file in a directory to serve as an index file. This module is not built by default.
        --with-http_random_index_module \
        # enables building the ngx_http_secure_link_module module. This module is not built by default.
        --with-http_secure_link_module \
        # enables building the ngx_http_degradation_module module. This module is not built by default.
        --with-http_degradation_module \
        # enables building the ngx_http_slice_module module that splits a request into subrequests, each returning a certain range of response. The module provides more effective caching of big responses. This module is not built by default.
        --with-http_slice_module \
        # enables building the ngx_http_stub_status_module module that provides access to basic status information. This module is not built by default.
        --with-http_stub_status_module \
        # disables building the ngx_http_charset_module module that adds the specified charset to the “Content-Type” response header field and can additionally convert data from one charset to another.
        # --without-http_charset_module \
        # disables building a module that compresses responses of an HTTP server. The zlib library is required to build and run this module.
        # --without-http_gzip_module \
        # disables building the ngx_http_ssi_module module that processes SSI (Server Side Includes) commands in responses passing through it.
        # --without-http_ssi_module \
        # disables building the ngx_http_userid_module module that sets cookies suitable for client identification.
        # --without-http_userid_module \
        # disables building the ngx_http_access_module module that allows limiting access to certain client addresses.
        # --without-http_access_module \
        # disables building the ngx_http_auth_basic_module module that allows limiting access to resources by validating the user name and password using the “HTTP Basic Authentication” protocol.
        # --without-http_auth_basic_module \
        # disables building the ngx_http_mirror_module module that implements mirroring of an original request by creating background mirror subrequests.
        # --without-http_mirror_module \
        # disables building the ngx_http_autoindex_module module that processes requests ending with the slash character (‘/’) and produces a directory listing in case the ngx_http_index_module module cannot find an index file.
        # --without-http_autoindex_module \
        # disables building the ngx_http_geo_module module that creates variables with values depending on the client IP address.
        # --without-http_geo_module \
        # disables building the ngx_http_map_module module that creates variables with values depending on values of other variables.
        # --without-http_map_module \
        # disables building the ngx_http_split_clients_module module that creates variables for A/B testing.
        # --without-http_split_clients_module \
        # disables building the ngx_http_referer_module module that can block access to a site for requests with invalid values in the “Referer” header field.
        # --without-http_referer_module \
        # disables building a module that allows an HTTP server to redirect requests and change URI of requests. The PCRE library is required to build and run this module.
        # --without-http_rewrite_module \
        # disables building an HTTP server proxying module.
        # --without-http_proxy_module \
        # disables building the ngx_http_fastcgi_module module that passes requests to a FastCGI server.
        # --without-http_fastcgi_module \
        # disables building the ngx_http_uwsgi_module module that passes requests to a uwsgi server.
        # --without-http_uwsgi_module \
        # disables building the ngx_http_scgi_module module that passes requests to an SCGI server.
        # --without-http_scgi_module \
        # disables building the ngx_http_grpc_module module that passes requests to a gRPC server.
        # --without-http_grpc_module \
        # disables building the ngx_http_memcached_module module that obtains responses from a memcached server.
        # --without-http_memcached_module \
        # disables building the ngx_http_limit_conn_module module that limits the number of connections per key, for example, the number of connections from a single IP address.
        # --without-http_limit_conn_module \
        # disables building the ngx_http_limit_req_module module that limits the request processing rate per key, for example, the processing rate of requests coming from a single IP address.
        # --without-http_limit_req_module \
        # disables building a module that emits single-pixel transparent GIF.
        # --without-http_empty_gif_module \
        # disables building the ngx_http_browser_module module that creates variables whose values depend on the value of the “User-Agent” request header field.
        # --without-http_browser_module \
        # disables building a module that implements the hash load balancing method.
        # --without-http_upstream_hash_module \
        # disables building a module that implements the ip_hash load balancing method.
        # --without-http_upstream_ip_hash_module \
        # disables building a module that implements the least_conn load balancing method.
        # --without-http_upstream_least_conn_module \
        # disables building a module that implements the random load balancing method.
        # --without-http_upstream_random_module \
        # disables building a module that provides caching of connections to upstream servers.
        # --without-http_upstream_keepalive_module \
        # disables building a module that makes it possible to store run-time state of an upstream group in a shared memory zone.
        # --without-http_upstream_zone_module \
        # enables building the embedded Perl module. This module is not built by default.
        # --with-http_perl_module \
        # --with-http_perl_module=dynamic \
        # defines a directory that will keep Perl modules.
        # --with-perl_modules_path=path \
        # sets the name of the Perl binary.
        # --with-perl=path \
        # sets the name of the primary request log file of the HTTP server. After installation, the file name can always be changed in the nginx.conf configuration file using the access_log directive. By default the file is named prefix/logs/access.log.
        # --http-log-path=path \
        # defines a directory for storing temporary files that hold client request bodies. After installation, the directory can always be changed in the nginx.conf configuration file using the client_body_temp_path directive. By default the directory is named prefix/client_body_temp.
        # --http-client-body-temp-path=path \
        # defines a directory for storing temporary files with data received from proxied servers. After installation, the directory can always be changed in the nginx.conf configuration file using the proxy_temp_path directive. By default the directory is named prefix/proxy_temp.
        # --http-proxy-temp-path=path \
        # defines a directory for storing temporary files with data received from FastCGI servers. After installation, the directory can always be changed in the nginx.conf configuration file using the fastcgi_temp_path directive. By default the directory is named prefix/fastcgi_temp.
        # --http-fastcgi-temp-path=path \
        # defines a directory for storing temporary files with data received from uwsgi servers. After installation, the directory can always be changed in the nginx.conf configuration file using the uwsgi_temp_path directive. By default the directory is named prefix/uwsgi_temp.
        # --http-uwsgi-temp-path=path \
        # defines a directory for storing temporary files with data received from SCGI servers. After installation, the directory can always be changed in the nginx.conf configuration file using the scgi_temp_path directive. By default the directory is named prefix/scgi_temp.
        # --http-scgi-temp-path=path \
        # disables the HTTP server.
        # --without-http \
        # disables HTTP cache.
        # --without-http-cache \
        # enables POP3/IMAP4/SMTP mail proxy server.
        --with-mail \
        # --with-mail=dynamic \
        # enables building a module that adds the SSL/TLS protocol support to the mail proxy server. This module is not built by default. The OpenSSL library is required to build and run this module.
        --with-mail_ssl_module \
        # disables the POP3 protocol in mail proxy server.
        # --without-mail_pop3_module \
        # disables the IMAP protocol in mail proxy server.
        # --without-mail_imap_module \
        # disables the SMTP protocol in mail proxy server.
        # --without-mail_smtp_module \
        # enables building the stream module for generic TCP/UDP proxying and load balancing. This module is not built by default.
        --with-stream \
        # --with-stream=dynamic \
        # enables building a module that adds the SSL/TLS protocol support to the stream module. This module is not built by default. The OpenSSL library is required to build and run this module.
        --with-stream_ssl_module \
        # enables building the ngx_stream_realip_module module that changes the client address to the address sent in the PROXY protocol header. This module is not built by default.
        --with-stream_realip_module \
        # enables building the ngx_stream_geoip_module module that creates variables depending on the client IP address and the precompiled MaxMind databases. This module is not built by default.
        --with-stream_geoip_module \
        # --with-stream_geoip_module=dynamic \
        # enables building the ngx_stream_ssl_preread_module module that allows extracting information from the ClientHello message without terminating SSL/TLS. This module is not built by default.
        --with-stream_ssl_preread_module \
        # disables building the ngx_stream_limit_conn_module module that limits the number of connections per key, for example, the number of connections from a single IP address.
        # --without-stream_limit_conn_module \
        # disables building the ngx_stream_access_module module that allows limiting access to certain client addresses.
        # --without-stream_access_module \
        # disables building the ngx_stream_geo_module module that creates variables with values depending on the client IP address.
        # --without-stream_geo_module \
        # disables building the ngx_stream_map_module module that creates variables with values depending on values of other variables.
        # --without-stream_map_module \
        # disables building the ngx_stream_split_clients_module module that creates variables for A/B testing.
        # --without-stream_split_clients_module \
        # disables building the ngx_stream_return_module module that sends some specified value to the client and then closes the connection.
        # --without-stream_return_module \
        # disables building the ngx_stream_set_module module that sets a value for a variable.
        # --without-stream_set_module \
        # disables building a module that implements the hash load balancing method.
        # --without-stream_upstream_hash_module \
        # disables building a module that implements the least_conn load balancing method.
        # --without-stream_upstream_least_conn_module \
        # disables building a module that implements the random load balancing method.
        # --without-stream_upstream_random_module \
        # disables building a module that makes it possible to store run-time state of an upstream group in a shared memory zone.
        # --without-stream_upstream_zone_module \
        # enables building the ngx_google_perftools_module module that enables profiling of nginx worker processes using Google Performance Tools. The module is intended for nginx developers and is not built by default.
        # --with-google_perftools_module \
        # enables building the ngx_cpp_test_module module.
        --with-cpp_test_module \
        # enables an external module.
        --add-module="nginx-module-vts-${NGINX_MODULE_VTS_VERSION}" \
        # enables an external dynamic module.
        # --add-dynamic-module=path \
        # enables dynamic modules compatibility.
        --with-compat \
        # sets the name of the C compiler.
        # --with-cc=path \
        # sets the name of the C preprocessor.
        # --with-cpp=path \
        # sets additional parameters that will be added to the CFLAGS variable. When using the system PCRE library under FreeBSD, --with-cc-opt="-I /usr/local/include" should be specified. If the number of files supported by select() needs to be increased it can also be specified here such as this: --with-cc-opt="-D FD_SETSIZE=2048".
        # --with-cc-opt=parameters \
        # sets additional parameters that will be used during linking. When using the system PCRE library under FreeBSD, --with-ld-opt="-L /usr/local/lib" should be specified.
        --with-ld-opt="-static -static-libgcc -static-libstdc++" \
        # enables building per specified CPU: pentium, pentiumpro, pentium3, pentium4, athlon, opteron, sparc32, sparc64, ppc64.
        # --with-cpu-opt=cpu \
        # disables the usage of the PCRE library.
        # --without-pcre \
        # forces the usage of the PCRE library.
        --with-pcre \
        # sets the path to the sources of the PCRE library. The library distribution needs to be downloaded from the PCRE site and extracted. The rest is done by nginx’s ./configure and make. The library is required for regular expressions support in the location directive and for the ngx_http_rewrite_module module.
        # --with-pcre=path \
        # sets additional build options for PCRE.
        # --with-pcre-opt=parameters \
        # builds the PCRE library with “just-in-time compilation” support (1.1.12, the pcre_jit directive).
        --with-pcre-jit \
        # disables use of the PCRE2 library instead of the original PCRE library (1.21.5).
        # --without-pcre2 \
        # sets the path to the sources of the zlib library. The library distribution needs to be downloaded from the zlib site and extracted. The rest is done by nginx’s ./configure and make. The library is required for the ngx_http_gzip_module module.
        # --with-zlib=path \
        # sets additional build options for zlib.
        # --with-zlib-opt=parameters \
        # enables the use of the zlib assembler sources optimized for one of the specified CPUs: pentium, pentiumpro.
        # --with-zlib-asm=cpu \
        # forces the libatomic_ops library usage.
        --with-libatomic \
        # sets the path to the libatomic_ops library sources.
        # --with-libatomic=path \
        # sets the path to the OpenSSL library sources.
        # --with-openssl=path \
        # sets additional build options for OpenSSL.
        # --with-openssl-opt=parameters \
        # enables the debugging log.
        # --with-debug \
        && \
    make "-j$(nproc)" && \
    make install
RUN ldd_output="$(ldd /usr/local/nginx/sbin/nginx 2>&1 || true)" && echo "ldd output: $ldd_output" && \
    ( \
        echo "$ldd_output" | grep -q "Not a valid dynamic program" && \
        echo "Static build verified: No shared libraries found" || \
        (echo "Error: Dynamic libraries found. This should be a static build!" && exit 1) \
    )
RUN file_output="$(file /usr/local/nginx/sbin/nginx 2>&1 || true)" && echo "file output: $file_output" && \
    ( \
        echo "$file_output" | grep -q "statically linked" && \
        echo "Static build verified" || \
        (echo "Error: Dynamic link found. This should be a static build!" && exit 1) \
    )

FROM redhat/ubi9-micro:${UBI9_MICRO_VERSION}
COPY --from=builder /usr/local/nginx /usr/local/nginx
RUN chmod +x /usr/local/nginx/sbin/nginx
EXPOSE 80 443
ENTRYPOINT ["/usr/local/nginx/sbin/nginx"]

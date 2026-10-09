#include <ngx_config.h>
#include <ngx_core.h>
#include <ngx_http.h>
#include <ngx_http_upstream.h>
#include <ngx_http_upstream_round_robin.h>
#include <ngx_event_connect.h>
#include <ngx_stream.h>
#include <ngx_stream_upstream.h>
#include <ngx_stream_upstream_round_robin.h>
#include <stdio.h>
#include <sys/socket.h>
#include <unistd.h>

static ngx_int_t harness_event_ready(ngx_event_t *event, ngx_uint_t flags);
static ngx_int_t harness_write_event_ready(ngx_event_t *event, size_t lowat);
static void harness_destroy_pool(ngx_pool_t *pool);
static void harness_close_connection(ngx_connection_t *connection);

#define ngx_handle_read_event(event, flags) \
    harness_event_ready((event), (flags))
#define ngx_handle_write_event(event, lowat) \
    harness_write_event_ready((event), (lowat))
#define ngx_shmtx_lock(mutex) ((void) (mutex))
#define ngx_shmtx_unlock(mutex) ((void) (mutex))
#define ngx_destroy_pool(pool) harness_destroy_pool(pool)
#define ngx_close_connection(connection) harness_close_connection(connection)
#undef ngx_time
#define ngx_time() ((time_t) 1)

#ifndef NGX_NATIVE_HEALTHCHECK_MODULE_SOURCE
#define NGX_NATIVE_HEALTHCHECK_MODULE_SOURCE \
    "ngx_http_native_healthcheck_module/ngx_http_native_healthcheck_module.c"
#endif
#include NGX_NATIVE_HEALTHCHECK_MODULE_SOURCE

ngx_module_t ngx_core_module;

static const u_char request_bytes[] = { 'P', 'I', 'N', 'G', '\r', '\n', 0 };
static const u_char response_bytes[] = { 'P', 'O', 'N', 'G' };
static u_char accepted_bytes[sizeof(request_bytes)];
static u_char received_bytes[sizeof(response_bytes)];
static size_t accepted_length;
static size_t received_length;
static ngx_uint_t send_calls;
static ngx_uint_t receive_calls;
static ngx_uint_t partial_send;
static ngx_uint_t retry_read;
static ngx_uint_t close_calls;
static ngx_uint_t destroyed_pool_calls;
static const char *case_name;

static ngx_int_t
harness_event_ready(ngx_event_t *event, ngx_uint_t flags)
{
    (void) event;
    (void) flags;
    return NGX_OK;
}

static ngx_int_t
harness_write_event_ready(ngx_event_t *event, size_t lowat)
{
    (void) event;
    (void) lowat;
    return NGX_OK;
}

static void
harness_destroy_pool(ngx_pool_t *pool)
{
    (void) pool;
    destroyed_pool_calls++;
}

static void
harness_close_connection(ngx_connection_t *connection)
{
    (void) connection;
    close_calls++;
}

static ssize_t
harness_send(ngx_connection_t *connection, u_char *data, size_t size)
{
    size_t accepted;

    (void) connection;
    send_calls++;
    if (partial_send && send_calls == 2) {
        return NGX_AGAIN;
    }
    accepted = partial_send && send_calls == 1 ? 2 : size;
    if (accepted > size || accepted_length + accepted > sizeof(accepted_bytes)) {
        return NGX_ERROR;
    }
    ngx_memcpy(accepted_bytes + accepted_length, data, accepted);
    accepted_length += accepted;
    return (ssize_t) accepted;
}

static ssize_t
harness_recv(ngx_connection_t *connection, u_char *data, size_t size)
{
    size_t available;

    (void) connection;
    receive_calls++;
    if (retry_read && receive_calls == 1) {
        return NGX_AGAIN;
    }
    available = sizeof(response_bytes) - received_length;
    if (available > size) {
        available = size;
    }
    if (available == 0) {
        return NGX_AGAIN;
    }
    ngx_memcpy(data, response_bytes + received_length, available);
    received_length += available;
    return (ssize_t) available;
}

static ngx_int_t
harness_require(ngx_uint_t condition, const char *message)
{
    if (!condition) {
        fprintf(stderr, "FAIL %s: %s\n", case_name, message);
        return NGX_ERROR;
    }
    fprintf(stdout, "PASS %s: %s\n", case_name, message);
    return NGX_OK;
}

static ngx_int_t
run_probe_case(const char *name, ngx_uint_t connect_start,
    ngx_uint_t partial, ngx_uint_t read_retry)
{
    ngx_stream_native_healthcheck_conf_t   conf;
    ngx_stream_native_healthcheck_state_t  state;
    ngx_stream_native_healthcheck_peer_t   peer_state;
    ngx_stream_upstream_rr_peers_t         group;
    ngx_stream_upstream_rr_peer_t          peer;
    ngx_slab_pool_t                        slab;
    ngx_pool_t                             pool;
    ngx_connection_t                       connection;
    ngx_event_t                            read_event;
    ngx_event_t                            write_event;
    ngx_stream_native_healthcheck_probe_t  probe;
    ngx_uint_t                             failed;
    int                                    sockets[2];

    case_name = name;
    sockets[0] = -1;
    sockets[1] = -1;
    if (connect_start && socketpair(AF_UNIX, SOCK_STREAM, 0, sockets) != 0) {
        fprintf(stderr, "FAIL %s: socketpair failed\n", case_name);
        return NGX_ERROR;
    }
    accepted_length = 0;
    received_length = 0;
    send_calls = 0;
    receive_calls = 0;
    partial_send = partial;
    retry_read = read_retry;
    close_calls = 0;
    destroyed_pool_calls = 0;
    ngx_memzero(&conf, sizeof(conf));
    ngx_memzero(&state, sizeof(state));
    ngx_memzero(&peer_state, sizeof(peer_state));
    ngx_memzero(&group, sizeof(group));
    ngx_memzero(&peer, sizeof(peer));
    ngx_memzero(&slab, sizeof(slab));
    ngx_memzero(&pool, sizeof(pool));
    ngx_memzero(&connection, sizeof(connection));
    ngx_memzero(&read_event, sizeof(read_event));
    ngx_memzero(&write_event, sizeof(write_event));
    ngx_memzero(&probe, sizeof(probe));

    state.shpool = &slab;
    state.active_probes = 1;
    conf.state = &state;
    conf.probes = &probe;
    conf.interval = 1000;
    conf.rise = 1;
    conf.fall = 3;
    conf.send_data.data = (u_char *) request_bytes;
    conf.send_data.len = sizeof(request_bytes);
    conf.type = NGX_STREAM_NATIVE_HEALTHCHECK_TCP;
    if (read_retry) {
        conf.expect_set = 1;
        conf.expect_data.data = (u_char *) response_bytes;
        conf.expect_data.len = sizeof(response_bytes);
        conf.min_recv = sizeof(response_bytes);
        conf.max_response = sizeof(received_bytes);
        probe.response = received_bytes;
    }
    group.shpool = NULL;
    peer.down = 1;
    peer_state.group = &group;
    peer_state.peer = &peer;
    peer_state.busy = 1;
    probe.prev = &conf.probes;
    probe.conf = &conf;
    probe.peer_state = &peer_state;
    probe.pool = &pool;
    probe.connection = &connection;
    probe.stage = connect_start ? NGX_HTTP_NATIVE_HEALTHCHECK_CONNECT
                                : NGX_HTTP_NATIVE_HEALTHCHECK_TLS;
    connection.data = &probe;
    connection.send = harness_send;
    connection.recv = harness_recv;
    connection.read = &read_event;
    connection.write = &write_event;
    connection.fd = sockets[0];
    read_event.data = &connection;
    read_event.handler = ngx_stream_native_healthcheck_probe_event;
    write_event.data = &connection;
    write_event.handler = ngx_stream_native_healthcheck_probe_event;
    write_event.write = 1;

    if (connect_start) {
        write_event.handler(&write_event);
    } else {
        ngx_stream_native_healthcheck_probe_send(&probe);
    }
    failed = 0;
    if (partial) {
        failed |= harness_require(!probe.done && probe.request_sent == 2
                                  && send_calls == 2,
                                  "partial send accepts two bytes and yields NGX_AGAIN") != NGX_OK;
        if (!connect_start) {
            failed |= harness_require(
                probe.stage == NGX_HTTP_NATIVE_HEALTHCHECK_SENDING,
                "TLS-completed probe enters the sending stage before yielding") != NGX_OK;
        }
        write_event.handler(&write_event);
    } else if (read_retry) {
        failed |= harness_require(!probe.done
                                  && probe.stage == NGX_HTTP_NATIVE_HEALTHCHECK_READING
                                  && receive_calls == 1,
                                  "TLS-completed probe waits after NGX_AGAIN on response read") != NGX_OK;
        read_event.handler(&read_event);
    }
    failed |= harness_require(probe.done && probe.request_sent == sizeof(request_bytes),
                              "event readiness resumes and completes the request") != NGX_OK;
    failed |= harness_require(accepted_length == sizeof(request_bytes)
                              && ngx_memcmp(accepted_bytes, request_bytes,
                                            sizeof(request_bytes)) == 0,
                              "accepted bytes exactly match the configured binary payload") != NGX_OK;
    if (read_retry) {
        failed |= harness_require(received_length == sizeof(response_bytes)
                                  && ngx_memcmp(received_bytes, response_bytes,
                                                sizeof(response_bytes)) == 0,
                                  "read readiness completes the configured response match") != NGX_OK;
    }
    failed |= harness_require(peer_state.checks_total == 1
                              && peer_state.check_failures_total == 0
                              && peer_state.last_result == 1
                              && peer_state.ready == 1 && peer.down == 0,
                              "one successful completion admits the previously down peer") != NGX_OK;
    failed |= harness_require(state.active_probes == 0 && !peer_state.busy
                              && conf.probes == NULL,
                              "completion releases the active slot and unlinks the probe") != NGX_OK;
    failed |= harness_require(close_calls == 1 && destroyed_pool_calls == 1,
                              "completion closes the connection and destroys its pool once") != NGX_OK;
    if (sockets[0] >= 0) {
        close(sockets[0]);
        close(sockets[1]);
    }
    return failed ? NGX_ERROR : NGX_OK;
}

/**
 * Exercise production stream probe callbacks through connect, send, and read
 * readiness paths with deterministic partial-write and retry behavior.
 *
 * The TLS cases start at the post-handshake stage so the harness isolates the
 * stream callback transitions while the runtime verifier covers real TLS.
 */
int
main(void)
{
    ngx_uint_t failed;

    failed = 0;
    failed |= run_probe_case("initial plain CONNECT", 1, 1, 0) != NGX_OK;
    failed |= run_probe_case("post-TLS partial write", 0, 1, 0) != NGX_OK;
    failed |= run_probe_case("post-TLS response read", 0, 0, 1) != NGX_OK;
    return failed ? 1 : 0;
}

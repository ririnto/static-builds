#include <ngx_config.h>
#include <ngx_core.h>
#include <ngx_http.h>
#include <ngx_http_upstream.h>
#include <ngx_http_upstream_round_robin.h>
#include <ngx_event_connect.h>
#include <ngx_stream.h>
#include <ngx_stream_upstream.h>
#include <ngx_stream_upstream_round_robin.h>
#if (NGX_HTTP_SSL || NGX_STREAM_SSL)
#include <ngx_event_openssl.h>
#include <openssl/ssl.h>
#include <openssl/x509v3.h>
#endif

typedef struct ngx_http_upstream_healthcheck_conf_s
    ngx_http_upstream_healthcheck_conf_t;
typedef struct ngx_http_upstream_healthcheck_state_s
    ngx_http_upstream_healthcheck_state_t;
typedef struct ngx_http_upstream_healthcheck_peer_s
    ngx_http_upstream_healthcheck_peer_t;
typedef struct ngx_http_upstream_healthcheck_template_s
    ngx_http_upstream_healthcheck_template_t;
typedef struct ngx_http_upstream_healthcheck_probe_s
    ngx_http_upstream_healthcheck_probe_t;
typedef struct ngx_stream_upstream_healthcheck_conf_s
    ngx_stream_upstream_healthcheck_conf_t;
typedef struct ngx_stream_upstream_healthcheck_state_s
    ngx_stream_upstream_healthcheck_state_t;
typedef struct ngx_stream_upstream_healthcheck_peer_s
    ngx_stream_upstream_healthcheck_peer_t;
typedef struct ngx_stream_upstream_healthcheck_template_s
    ngx_stream_upstream_healthcheck_template_t;
typedef struct ngx_stream_upstream_healthcheck_probe_s
    ngx_stream_upstream_healthcheck_probe_t;

typedef enum {
    NGX_HTTP_UPSTREAM_HEALTHCHECK_HTTP,
    NGX_HTTP_UPSTREAM_HEALTHCHECK_HTTPS,
    NGX_HTTP_UPSTREAM_HEALTHCHECK_TCP,
    NGX_HTTP_UPSTREAM_HEALTHCHECK_TLS_ONLY
} ngx_http_upstream_healthcheck_type_e;

struct ngx_http_upstream_healthcheck_conf_s {
    ngx_http_upstream_srv_conf_t          *uscf;
    ngx_http_upstream_rr_peers_t          *peers;
    ngx_pool_t                            *config_pool;
    ngx_shm_zone_t                        *shm_zone;
    ngx_http_upstream_healthcheck_state_t   *state;
    ngx_http_upstream_healthcheck_template_t *templates;
    ngx_http_upstream_healthcheck_probe_t   *probes;
    ngx_event_t                            timer;
    ngx_ssl_t                              ssl;
    ngx_array_t                           *statuses;
    ngx_str_t                              uri;
    ngx_str_t                              host;
    ngx_str_t                              tls_name;
    ngx_str_t                              trusted_certificate;
    ngx_str_t                              zone_name;
    ngx_msec_t                             interval;
    ngx_msec_t                             timeout;
    ssize_t                                shm_size;
    ngx_uint_t                             fall;
    ngx_uint_t                             rise;
    ngx_uint_t                             concurrency;
    ngx_http_upstream_healthcheck_type_e    type;
    ngx_uint_t                             enabled;
    ngx_uint_t                             statuses_set;
    ngx_uint_t                             tls_name_set;
    ngx_uint_t                             trusted_certificate_set;
    ngx_uint_t                             stopping;
};

struct ngx_http_upstream_healthcheck_state_s {
    ngx_slab_pool_t                       *shpool;
    ngx_http_upstream_healthcheck_peer_t    *peers;
    ngx_http_upstream_healthcheck_peer_t    *tail;
    ngx_uint_t                             peer_count;
    ngx_uint_t                             active_probes;
    ngx_uint_t                             scan_generation;
    ngx_uint_t                             next_peer_id;
    ngx_uint_t                             schedule_cursor;
    ngx_uint_t                             errors_total;
    ngx_uint_t                             concurrency_limited_total;
    ngx_uint_t                             memory_failures_total;
    ngx_pid_t                              worker_pid;
    time_t                                 last_scan;
    time_t                                 last_memory_log;
    time_t                                 last_limit_log;
};

struct ngx_http_upstream_healthcheck_peer_s {
    ngx_http_upstream_healthcheck_peer_t    *next;
    ngx_http_upstream_rr_peers_t          *group;
    ngx_http_upstream_rr_peer_t           *peer;
    ngx_uint_t                             id;
    ngx_uint_t                             seen_generation;
    ngx_uint_t                             backup;
    ngx_uint_t                             admin_down;
    ngx_uint_t                             ready;
    ngx_uint_t                             removed;
    ngx_uint_t                             busy;
    ngx_uint_t                             has_result;
    ngx_uint_t                             last_result;
    ngx_uint_t                             status_code;
    ngx_uint_t                             rise_streak;
    ngx_uint_t                             fall_streak;
    ngx_uint_t                             checks_total;
    ngx_uint_t                             check_failures_total;
    ngx_uint_t                             check_up_down_total;
    ngx_msec_t                             last_duration;
    ngx_msec_t                             next_due;
    time_t                                 last_check;
    time_t                                 last_change;
};

struct ngx_http_upstream_healthcheck_template_s {
    ngx_http_upstream_healthcheck_template_t *next;
    ngx_http_upstream_rr_peer_t            *peer;
    ngx_uint_t                              backup;
    ngx_uint_t                              admin_down;
};

struct ngx_http_upstream_healthcheck_probe_s {
    ngx_http_upstream_healthcheck_probe_t   *next;
    ngx_http_upstream_healthcheck_probe_t  **prev;
    ngx_http_upstream_healthcheck_conf_t    *conf;
    ngx_http_upstream_healthcheck_peer_t    *peer_state;
    ngx_pool_t                            *pool;
    ngx_connection_t                      *connection;
    ngx_event_t                            deadline;
    u_char                                *request;
    size_t                                 request_len;
    size_t                                 request_sent;
    size_t                                 response_len;
    u_char                                 response[128];
    ngx_msec_t                             started;
    ngx_uint_t                             stage;
    ngx_uint_t                             done;
};

typedef enum {
    NGX_STREAM_UPSTREAM_HEALTHCHECK_TCP,
    NGX_STREAM_UPSTREAM_HEALTHCHECK_TLS
} ngx_stream_upstream_healthcheck_type_e;

struct ngx_stream_upstream_healthcheck_conf_s {
    ngx_stream_upstream_srv_conf_t          *uscf;
    ngx_stream_upstream_rr_peers_t          *peers;
    ngx_pool_t                              *config_pool;
    ngx_shm_zone_t                          *shm_zone;
    ngx_stream_upstream_healthcheck_state_t   *state;
    ngx_stream_upstream_healthcheck_template_t *templates;
    ngx_stream_upstream_healthcheck_probe_t   *probes;
    ngx_event_t                              timer;
    ngx_ssl_t                                ssl;
    ngx_str_t                                send_data;
    ngx_str_t                                expect_data;
    ngx_str_t                                tls_name;
    ngx_str_t                                trusted_certificate;
    ngx_str_t                                zone_name;
    ngx_msec_t                               interval;
    ngx_msec_t                               timeout;
    ssize_t                                  shm_size;
    size_t                                   max_response;
    size_t                                   min_recv;
    ngx_uint_t                               fall;
    ngx_uint_t                               rise;
    ngx_uint_t                               concurrency;
    ngx_stream_upstream_healthcheck_type_e    type;
    ngx_uint_t                               enabled;
    ngx_uint_t                               send_set;
    ngx_uint_t                               expect_set;
    ngx_uint_t                               min_recv_set;
    ngx_uint_t                               tls_name_set;
    ngx_uint_t                               trusted_certificate_set;
    ngx_uint_t                               stopping;
};

struct ngx_stream_upstream_healthcheck_state_s {
    ngx_slab_pool_t                         *shpool;
    ngx_stream_upstream_healthcheck_peer_t    *peers;
    ngx_stream_upstream_healthcheck_peer_t    *tail;
    ngx_uint_t                               peer_count;
    ngx_uint_t                               active_probes;
    ngx_uint_t                               scan_generation;
    ngx_uint_t                               next_peer_id;
    ngx_uint_t                               schedule_cursor;
    ngx_uint_t                               errors_total;
    ngx_uint_t                               concurrency_limited_total;
    ngx_uint_t                               memory_failures_total;
    ngx_pid_t                                worker_pid;
    time_t                                   last_scan;
    time_t                                   last_memory_log;
    time_t                                   last_limit_log;
};

struct ngx_stream_upstream_healthcheck_peer_s {
    ngx_stream_upstream_healthcheck_peer_t    *next;
    ngx_stream_upstream_rr_peers_t           *group;
    ngx_stream_upstream_rr_peer_t            *peer;
    ngx_uint_t                                id;
    ngx_uint_t                                seen_generation;
    ngx_uint_t                                backup;
    ngx_uint_t                                admin_down;
    ngx_uint_t                                ready;
    ngx_uint_t                                removed;
    ngx_uint_t                                busy;
    ngx_uint_t                                has_result;
    ngx_uint_t                                last_result;
    ngx_uint_t                                status_code;
    ngx_uint_t                                rise_streak;
    ngx_uint_t                                fall_streak;
    ngx_uint_t                                checks_total;
    ngx_uint_t                                check_failures_total;
    ngx_uint_t                                check_up_down_total;
    ngx_msec_t                                last_duration;
    ngx_msec_t                                next_due;
    time_t                                   last_check;
    time_t                                   last_change;
};

struct ngx_stream_upstream_healthcheck_template_s {
    ngx_stream_upstream_healthcheck_template_t *next;
    ngx_stream_upstream_rr_peer_t            *peer;
    ngx_uint_t                                admin_down;
};

struct ngx_stream_upstream_healthcheck_probe_s {
    ngx_stream_upstream_healthcheck_probe_t   *next;
    ngx_stream_upstream_healthcheck_probe_t  **prev;
    ngx_stream_upstream_healthcheck_conf_t    *conf;
    ngx_stream_upstream_healthcheck_peer_t    *peer_state;
    ngx_pool_t                              *pool;
    ngx_connection_t                        *connection;
    ngx_event_t                              deadline;
    u_char                                  *response;
    size_t                                   request_sent;
    size_t                                   response_len;
    ngx_msec_t                               started;
    ngx_uint_t                               stage;
    ngx_uint_t                               done;
};

typedef struct {
    ngx_array_t                            upstreams;
} ngx_http_upstream_healthcheck_main_conf_t;

typedef struct {
    ngx_array_t                            upstreams;
} ngx_stream_upstream_healthcheck_main_conf_t;

typedef struct {
    ngx_str_t                              upstream;
    ngx_str_t                              server;
    ngx_str_t                              peer;
    ngx_uint_t                             backup;
    ngx_uint_t                             admin_down;
    ngx_uint_t                             ready;
    ngx_int_t                              last_result;
    ngx_uint_t                             status_code;
    ngx_uint_t                             rise_streak;
    ngx_uint_t                             fall_streak;
    ngx_uint_t                             checks_total;
    ngx_uint_t                             check_failures_total;
    ngx_uint_t                             check_up_down_total;
    ngx_msec_t                             last_duration;
    time_t                                 last_check;
    time_t                                 last_change;
} ngx_http_upstream_healthcheck_sample_t;

typedef struct {
    u_char                                *data;
    size_t                                 len;
    size_t                                 capacity;
    ngx_pool_t                            *pool;
} ngx_http_upstream_healthcheck_output_t;

#define NGX_HTTP_UPSTREAM_HEALTHCHECK_CONNECT  0
#define NGX_HTTP_UPSTREAM_HEALTHCHECK_SENDING  1
#define NGX_HTTP_UPSTREAM_HEALTHCHECK_READING  2
#define NGX_HTTP_UPSTREAM_HEALTHCHECK_TLS      3
#define NGX_HTTP_UPSTREAM_HEALTHCHECK_LINE_MAX 128
#define NGX_HTTP_UPSTREAM_HEALTHCHECK_MAX_CONCURRENCY 1024
#define NGX_HTTP_UPSTREAM_HEALTHCHECK_MAX_STREAK 1000000

static ngx_int_t ngx_http_upstream_healthcheck_postconfiguration(ngx_conf_t *cf);
static void *ngx_http_upstream_healthcheck_create_main_conf(ngx_conf_t *cf);
static void *ngx_http_upstream_healthcheck_create_srv_conf(ngx_conf_t *cf);
static void *ngx_http_upstream_healthcheck_create_loc_conf(ngx_conf_t *cf);
static char *ngx_http_upstream_healthcheck_directive(ngx_conf_t *cf,
    ngx_command_t *cmd, void *conf);
static char *ngx_http_upstream_healthcheck_statuses(ngx_conf_t *cf,
    ngx_command_t *cmd, void *conf);
static char *ngx_http_upstream_healthcheck_tls_name(ngx_conf_t *cf,
    ngx_command_t *cmd, void *conf);
static char *ngx_http_upstream_healthcheck_trusted_certificate(ngx_conf_t *cf,
    ngx_command_t *cmd, void *conf);
static char *ngx_http_upstream_healthcheck_metrics_directive(ngx_conf_t *cf,
    ngx_command_t *cmd, void *conf);
static ngx_int_t ngx_http_upstream_healthcheck_metrics(ngx_http_request_t *r);
static ngx_int_t ngx_http_upstream_healthcheck_init_process(ngx_cycle_t *cycle);
static void ngx_http_upstream_healthcheck_exit_process(ngx_cycle_t *cycle);
static ngx_int_t ngx_http_upstream_healthcheck_init_zone(ngx_shm_zone_t *shm_zone,
    void *data);
static void ngx_http_upstream_healthcheck_timer(ngx_event_t *ev);
static void ngx_http_upstream_healthcheck_scan(
    ngx_http_upstream_healthcheck_conf_t *conf);
static void ngx_http_upstream_healthcheck_scan_group(
    ngx_http_upstream_healthcheck_conf_t *conf,
    ngx_http_upstream_rr_peers_t *group, ngx_uint_t backup,
    ngx_uint_t generation);
static void ngx_http_upstream_healthcheck_schedule(
    ngx_http_upstream_healthcheck_conf_t *conf);
static ngx_http_upstream_healthcheck_peer_t *
ngx_http_upstream_healthcheck_next_due(
    ngx_http_upstream_healthcheck_state_t *state, ngx_msec_t now,
    ngx_uint_t after_id);
static ngx_int_t ngx_http_upstream_healthcheck_start(
    ngx_http_upstream_healthcheck_conf_t *conf,
    ngx_http_upstream_healthcheck_peer_t *peer_state);
static void ngx_http_upstream_healthcheck_probe_event(ngx_event_t *ev);
static void ngx_http_upstream_healthcheck_probe_deadline(ngx_event_t *ev);
static void ngx_http_upstream_healthcheck_probe_connected(
    ngx_http_upstream_healthcheck_probe_t *probe);
static void ngx_http_upstream_healthcheck_probe_start_tls(
    ngx_http_upstream_healthcheck_probe_t *probe);
static void ngx_http_upstream_healthcheck_probe_tls_done(ngx_connection_t *c);
static void ngx_http_upstream_healthcheck_probe_send(
    ngx_http_upstream_healthcheck_probe_t *probe);
static void ngx_http_upstream_healthcheck_probe_read(
    ngx_http_upstream_healthcheck_probe_t *probe);
static ngx_int_t ngx_http_upstream_healthcheck_parse_status(
    u_char *line, size_t len, ngx_uint_t *status);
static ngx_int_t ngx_http_upstream_healthcheck_status_allowed(
    ngx_http_upstream_healthcheck_conf_t *conf, ngx_uint_t status);
static void ngx_http_upstream_healthcheck_probe_finish(
    ngx_http_upstream_healthcheck_probe_t *probe, ngx_uint_t success,
    ngx_uint_t status, ngx_uint_t attempted, ngx_uint_t force_down);
static void ngx_http_upstream_healthcheck_probe_link(
    ngx_http_upstream_healthcheck_probe_t *probe);
static void ngx_http_upstream_healthcheck_probe_unlink(
    ngx_http_upstream_healthcheck_probe_t *probe);
static void ngx_http_upstream_healthcheck_complete(
    ngx_http_upstream_healthcheck_conf_t *conf,
    ngx_http_upstream_healthcheck_peer_t *peer_state, ngx_uint_t success,
    ngx_uint_t status, ngx_uint_t attempted, ngx_uint_t force_down,
    ngx_msec_t duration);
static void ngx_http_upstream_healthcheck_start_failed(
    ngx_http_upstream_healthcheck_conf_t *conf,
    ngx_http_upstream_healthcheck_peer_t *peer_state, ngx_uint_t force_down);
static void ngx_http_upstream_healthcheck_log_memory_failure(
    ngx_http_upstream_healthcheck_conf_t *conf,
    ngx_http_upstream_healthcheck_state_t *state);
static ngx_http_upstream_healthcheck_peer_t *
ngx_http_upstream_healthcheck_find_peer(
    ngx_http_upstream_healthcheck_state_t *state,
    ngx_http_upstream_rr_peers_t *group,
    ngx_http_upstream_rr_peer_t *peer);
static ngx_http_upstream_healthcheck_peer_t *
ngx_http_upstream_healthcheck_add_peer_locked(
    ngx_http_upstream_healthcheck_conf_t *conf,
    ngx_http_upstream_rr_peers_t *group,
    ngx_http_upstream_rr_peer_t *peer, ngx_uint_t backup);
static void ngx_http_upstream_healthcheck_remove_peer_locked(
    ngx_http_upstream_healthcheck_state_t *state,
    ngx_http_upstream_healthcheck_peer_t *peer_state);
static ngx_uint_t ngx_http_upstream_healthcheck_admin_down(
    ngx_http_upstream_healthcheck_conf_t *conf,
    ngx_http_upstream_rr_peer_t *peer, ngx_uint_t backup);
static ngx_str_t *ngx_http_upstream_healthcheck_server_name(
    ngx_http_upstream_rr_peer_t *peer);
static void ngx_http_upstream_healthcheck_set_down(
    ngx_http_upstream_rr_peers_t *group,
    ngx_http_upstream_rr_peer_t *peer);
static void ngx_http_upstream_healthcheck_set_up(
    ngx_http_upstream_rr_peers_t *group,
    ngx_http_upstream_rr_peer_t *peer);
static ngx_int_t ngx_http_upstream_healthcheck_prepare_zone(
    ngx_http_upstream_healthcheck_conf_t *conf);
static ngx_int_t ngx_http_upstream_healthcheck_collect_samples(
    ngx_http_request_t *r, ngx_http_upstream_healthcheck_conf_t *conf,
    ngx_http_upstream_rr_peers_t *group, ngx_uint_t backup,
    ngx_http_upstream_healthcheck_sample_t **samples, ngx_uint_t *count);
static ngx_int_t ngx_http_upstream_healthcheck_output_reserve(
    ngx_http_upstream_healthcheck_output_t *out, size_t extra);
static ngx_int_t ngx_http_upstream_healthcheck_output_append(
    ngx_http_upstream_healthcheck_output_t *out, const u_char *data,
    size_t len);
static ngx_int_t ngx_http_upstream_healthcheck_output_text(
    ngx_http_upstream_healthcheck_output_t *out, const char *text);
static ngx_int_t ngx_http_upstream_healthcheck_output_uint(
    ngx_http_upstream_healthcheck_output_t *out, ngx_uint_t value);
static ngx_int_t ngx_http_upstream_healthcheck_output_int(
    ngx_http_upstream_healthcheck_output_t *out, ngx_int_t value);
static ngx_int_t ngx_http_upstream_healthcheck_output_seconds(
    ngx_http_upstream_healthcheck_output_t *out, ngx_msec_t value);
static ngx_int_t ngx_http_upstream_healthcheck_output_label(
    ngx_http_upstream_healthcheck_output_t *out, const ngx_str_t *value);
static ngx_int_t ngx_http_upstream_healthcheck_output_labels(
    ngx_http_upstream_healthcheck_output_t *out,
    const ngx_http_upstream_healthcheck_sample_t *sample);
static ngx_int_t ngx_http_upstream_healthcheck_output_group_label(
    ngx_http_upstream_healthcheck_output_t *out, const ngx_str_t *upstream);
static ngx_int_t ngx_http_upstream_healthcheck_output_help(
    ngx_http_upstream_healthcheck_output_t *out, const char *name,
    const char *help, const char *type);
static ngx_int_t ngx_http_upstream_healthcheck_output_sample(
    ngx_http_upstream_healthcheck_output_t *out, const char *name,
    const ngx_http_upstream_healthcheck_sample_t *sample, const char *value);
static ngx_int_t ngx_http_upstream_healthcheck_output_sample_uint(
    ngx_http_upstream_healthcheck_output_t *out, const char *name,
    const ngx_http_upstream_healthcheck_sample_t *sample, ngx_uint_t value);
static ngx_int_t ngx_http_upstream_healthcheck_output_sample_int(
    ngx_http_upstream_healthcheck_output_t *out, const char *name,
    const ngx_http_upstream_healthcheck_sample_t *sample, ngx_int_t value);
static ngx_int_t ngx_http_upstream_healthcheck_output_group_uint(
    ngx_http_upstream_healthcheck_output_t *out, const char *name,
    const ngx_str_t *upstream, ngx_uint_t value);
static ngx_int_t ngx_http_upstream_healthcheck_output_group_time(
    ngx_http_upstream_healthcheck_output_t *out, const char *name,
    const ngx_str_t *upstream, time_t value);
static ngx_int_t ngx_http_upstream_healthcheck_render_peer_family(
    ngx_http_upstream_healthcheck_output_t *out, const char *name,
    ngx_array_t *samples, ngx_uint_t family, time_t now);
static ngx_int_t ngx_http_upstream_healthcheck_render_group(
    ngx_http_upstream_healthcheck_output_t *out,
    ngx_http_upstream_healthcheck_conf_t *conf);
static ngx_int_t ngx_http_upstream_healthcheck_copy_string(ngx_pool_t *pool,
    ngx_str_t *dst, const ngx_str_t *src);
static ngx_int_t ngx_http_upstream_healthcheck_parse_uint(ngx_str_t *value,
    ngx_uint_t min, ngx_uint_t max, ngx_uint_t *result);
static ngx_int_t ngx_http_upstream_healthcheck_parse_msec(ngx_str_t *value,
    ngx_msec_t *result);
static ngx_int_t ngx_http_upstream_healthcheck_has_crlf(const ngx_str_t *value);
static ngx_int_t ngx_http_upstream_healthcheck_same_string(const ngx_str_t *a,
    const ngx_str_t *b);
static ngx_int_t ngx_http_upstream_healthcheck_test_connect(
    ngx_connection_t *c);
#if (NGX_HTTP_SSL || NGX_STREAM_SSL)
static ngx_int_t ngx_upstream_healthcheck_ssl_context(ngx_conf_t *cf,
    ngx_ssl_t *ssl, ngx_str_t *trusted_certificate, ngx_str_t *upstream);
static ngx_int_t ngx_upstream_healthcheck_tls_start(ngx_connection_t *c,
    ngx_pool_t *pool, ngx_ssl_t *ssl, ngx_str_t *tls_name,
    ngx_connection_handler_pt handler, ngx_uint_t *force_down);
static ngx_int_t ngx_upstream_healthcheck_tls_verified(ngx_connection_t *c,
    ngx_str_t *tls_name);
static void ngx_upstream_healthcheck_tls_close(ngx_connection_t *c);
#endif
static void *ngx_stream_upstream_healthcheck_create_main_conf(ngx_conf_t *cf);
static void *ngx_stream_upstream_healthcheck_create_srv_conf(ngx_conf_t *cf);
static char *ngx_stream_upstream_healthcheck_directive(ngx_conf_t *cf,
    ngx_command_t *cmd, void *conf);
static char *ngx_stream_upstream_healthcheck_payload(ngx_conf_t *cf,
    ngx_command_t *cmd, void *conf);
static char *ngx_stream_upstream_healthcheck_payload_hex(ngx_conf_t *cf,
    ngx_command_t *cmd, void *conf);
static char *ngx_stream_upstream_healthcheck_expect(ngx_conf_t *cf,
    ngx_command_t *cmd, void *conf);
static char *ngx_stream_upstream_healthcheck_expect_hex(ngx_conf_t *cf,
    ngx_command_t *cmd, void *conf);
static char *ngx_stream_upstream_healthcheck_tls_name(ngx_conf_t *cf,
    ngx_command_t *cmd, void *conf);
static char *ngx_stream_upstream_healthcheck_trusted_certificate(
    ngx_conf_t *cf, ngx_command_t *cmd, void *conf);
static ngx_int_t ngx_stream_upstream_healthcheck_postconfiguration(
    ngx_conf_t *cf);
static ngx_int_t ngx_stream_upstream_healthcheck_init_zone(
    ngx_shm_zone_t *shm_zone, void *data);
static ngx_int_t ngx_stream_upstream_healthcheck_init_process(
    ngx_cycle_t *cycle);
static void ngx_stream_upstream_healthcheck_exit_process(ngx_cycle_t *cycle);
static void ngx_stream_upstream_healthcheck_timer(ngx_event_t *ev);
static void ngx_stream_upstream_healthcheck_scan(
    ngx_stream_upstream_healthcheck_conf_t *conf);
static void ngx_stream_upstream_healthcheck_scan_group(
    ngx_stream_upstream_healthcheck_conf_t *conf,
    ngx_stream_upstream_rr_peers_t *group, ngx_uint_t backup,
    ngx_uint_t generation);
static void ngx_stream_upstream_healthcheck_schedule(
    ngx_stream_upstream_healthcheck_conf_t *conf);
static ngx_stream_upstream_healthcheck_peer_t *
ngx_stream_upstream_healthcheck_next_due(
    ngx_stream_upstream_healthcheck_state_t *state, ngx_msec_t now,
    ngx_uint_t after_id);
static ngx_stream_upstream_healthcheck_peer_t *
ngx_stream_upstream_healthcheck_find_peer(
    ngx_stream_upstream_healthcheck_state_t *state,
    ngx_stream_upstream_rr_peers_t *group,
    ngx_stream_upstream_rr_peer_t *peer);
static ngx_stream_upstream_healthcheck_peer_t *
ngx_stream_upstream_healthcheck_add_peer_locked(
    ngx_stream_upstream_healthcheck_conf_t *conf,
    ngx_stream_upstream_rr_peers_t *group,
    ngx_stream_upstream_rr_peer_t *peer, ngx_uint_t backup);
static void ngx_stream_upstream_healthcheck_remove_peer_locked(
    ngx_stream_upstream_healthcheck_state_t *state,
    ngx_stream_upstream_healthcheck_peer_t *peer_state);
static ngx_uint_t ngx_stream_upstream_healthcheck_admin_down(
    ngx_stream_upstream_healthcheck_conf_t *conf,
    ngx_stream_upstream_rr_peer_t *peer, ngx_uint_t backup);
static ngx_str_t *ngx_stream_upstream_healthcheck_server_name(
    ngx_stream_upstream_rr_peer_t *peer);
static void ngx_stream_upstream_healthcheck_set_down(
    ngx_stream_upstream_rr_peers_t *group,
    ngx_stream_upstream_rr_peer_t *peer);
static void ngx_stream_upstream_healthcheck_set_up(
    ngx_stream_upstream_rr_peers_t *group,
    ngx_stream_upstream_rr_peer_t *peer);
static ngx_int_t ngx_stream_upstream_healthcheck_prepare_zone(
    ngx_stream_upstream_healthcheck_conf_t *conf);
static ngx_int_t ngx_stream_upstream_healthcheck_start(
    ngx_stream_upstream_healthcheck_conf_t *conf,
    ngx_stream_upstream_healthcheck_peer_t *peer_state);
static void ngx_stream_upstream_healthcheck_probe_event(ngx_event_t *ev);
static void ngx_stream_upstream_healthcheck_probe_deadline(ngx_event_t *ev);
static void ngx_stream_upstream_healthcheck_probe_start_tls(
    ngx_stream_upstream_healthcheck_probe_t *probe);
static void ngx_stream_upstream_healthcheck_probe_tls_done(
    ngx_connection_t *c);
static void ngx_stream_upstream_healthcheck_probe_send(
    ngx_stream_upstream_healthcheck_probe_t *probe);
static void ngx_stream_upstream_healthcheck_probe_read(
    ngx_stream_upstream_healthcheck_probe_t *probe);
static ngx_uint_t ngx_stream_upstream_healthcheck_response_matches(
    ngx_stream_upstream_healthcheck_probe_t *probe);
static void ngx_stream_upstream_healthcheck_probe_finish(
    ngx_stream_upstream_healthcheck_probe_t *probe, ngx_uint_t success,
    ngx_uint_t attempted, ngx_uint_t force_down);
static void ngx_stream_upstream_healthcheck_probe_link(
    ngx_stream_upstream_healthcheck_probe_t *probe);
static void ngx_stream_upstream_healthcheck_probe_unlink(
    ngx_stream_upstream_healthcheck_probe_t *probe);
static void ngx_stream_upstream_healthcheck_start_failed(
    ngx_stream_upstream_healthcheck_conf_t *conf,
    ngx_stream_upstream_healthcheck_peer_t *peer_state, ngx_uint_t memory);
static void ngx_stream_upstream_healthcheck_complete(
    ngx_stream_upstream_healthcheck_conf_t *conf,
    ngx_stream_upstream_healthcheck_peer_t *peer_state, ngx_uint_t success,
    ngx_uint_t attempted, ngx_uint_t force_down, ngx_msec_t duration);
static void ngx_stream_upstream_healthcheck_log_memory_failure(
    ngx_stream_upstream_healthcheck_conf_t *conf,
    ngx_stream_upstream_healthcheck_state_t *state);
static ngx_int_t ngx_stream_upstream_healthcheck_collect_samples(
    ngx_http_request_t *r, ngx_stream_upstream_healthcheck_conf_t *conf,
    ngx_stream_upstream_rr_peers_t *group, ngx_uint_t backup,
    ngx_http_upstream_healthcheck_sample_t **samples, ngx_uint_t *count);
static ngx_int_t ngx_stream_upstream_healthcheck_render_group(
    ngx_http_upstream_healthcheck_output_t *out,
    ngx_stream_upstream_healthcheck_conf_t *conf);
static ngx_int_t ngx_stream_upstream_healthcheck_set_payload(
    ngx_conf_t *cf, ngx_str_t *target, ngx_uint_t *set,
    ngx_str_t *value, ngx_uint_t hex);

static ngx_command_t ngx_http_upstream_healthcheck_commands[] = {
    { ngx_string("healthcheck"),
      NGX_HTTP_UPS_CONF|NGX_CONF_1MORE,
      ngx_http_upstream_healthcheck_directive,
      NGX_HTTP_SRV_CONF_OFFSET,
      0,
      NULL },

    { ngx_string("healthcheck_statuses"),
      NGX_HTTP_UPS_CONF|NGX_CONF_1MORE,
      ngx_http_upstream_healthcheck_statuses,
      NGX_HTTP_SRV_CONF_OFFSET,
      0,
      NULL },

    { ngx_string("healthcheck_tls_name"),
      NGX_HTTP_UPS_CONF|NGX_CONF_TAKE1,
      ngx_http_upstream_healthcheck_tls_name,
      NGX_HTTP_SRV_CONF_OFFSET,
      0,
      NULL },

    { ngx_string("healthcheck_trusted_certificate"),
      NGX_HTTP_UPS_CONF|NGX_CONF_TAKE1,
      ngx_http_upstream_healthcheck_trusted_certificate,
      NGX_HTTP_SRV_CONF_OFFSET,
      0,
      NULL },

    { ngx_string("healthcheck_metrics"),
      NGX_HTTP_LOC_CONF|NGX_CONF_NOARGS,
      ngx_http_upstream_healthcheck_metrics_directive,
      NGX_HTTP_LOC_CONF_OFFSET,
      0,
      NULL },

      ngx_null_command
};

static ngx_command_t ngx_stream_upstream_healthcheck_commands[] = {
    { ngx_string("healthcheck_tcp"),
      NGX_STREAM_UPS_CONF|NGX_CONF_1MORE,
      ngx_stream_upstream_healthcheck_directive,
      NGX_STREAM_SRV_CONF_OFFSET,
      0,
      NULL },

    { ngx_string("healthcheck_send"),
      NGX_STREAM_UPS_CONF|NGX_CONF_TAKE1,
      ngx_stream_upstream_healthcheck_payload,
      NGX_STREAM_SRV_CONF_OFFSET,
      0,
      NULL },

    { ngx_string("healthcheck_send_hex"),
      NGX_STREAM_UPS_CONF|NGX_CONF_TAKE1,
      ngx_stream_upstream_healthcheck_payload_hex,
      NGX_STREAM_SRV_CONF_OFFSET,
      0,
      NULL },

    { ngx_string("healthcheck_expect"),
      NGX_STREAM_UPS_CONF|NGX_CONF_1MORE,
      ngx_stream_upstream_healthcheck_expect,
      NGX_STREAM_SRV_CONF_OFFSET,
      0,
      NULL },

    { ngx_string("healthcheck_expect_hex"),
      NGX_STREAM_UPS_CONF|NGX_CONF_1MORE,
      ngx_stream_upstream_healthcheck_expect_hex,
      NGX_STREAM_SRV_CONF_OFFSET,
      0,
      NULL },

    { ngx_string("healthcheck_tls_name"),
      NGX_STREAM_UPS_CONF|NGX_CONF_TAKE1,
      ngx_stream_upstream_healthcheck_tls_name,
      NGX_STREAM_SRV_CONF_OFFSET,
      0,
      NULL },

    { ngx_string("healthcheck_trusted_certificate"),
      NGX_STREAM_UPS_CONF|NGX_CONF_TAKE1,
      ngx_stream_upstream_healthcheck_trusted_certificate,
      NGX_STREAM_SRV_CONF_OFFSET,
      0,
      NULL },

      ngx_null_command
};

static ngx_http_module_t ngx_http_upstream_healthcheck_module_ctx = {
    NULL,
    ngx_http_upstream_healthcheck_postconfiguration,
    ngx_http_upstream_healthcheck_create_main_conf,
    NULL,
    ngx_http_upstream_healthcheck_create_srv_conf,
    NULL,
    ngx_http_upstream_healthcheck_create_loc_conf,
    NULL
};

static ngx_stream_module_t ngx_stream_upstream_healthcheck_module_ctx = {
    NULL,
    ngx_stream_upstream_healthcheck_postconfiguration,
    ngx_stream_upstream_healthcheck_create_main_conf,
    NULL,
    ngx_stream_upstream_healthcheck_create_srv_conf,
    NULL
};

/**
 * Registers upstream asynchronous active upstream health checks.
 */
ngx_module_t ngx_http_upstream_healthcheck_module = {
    NGX_MODULE_V1,
    &ngx_http_upstream_healthcheck_module_ctx,
    ngx_http_upstream_healthcheck_commands,
    NGX_HTTP_MODULE,
    NULL,
    NULL,
    ngx_http_upstream_healthcheck_init_process,
    NULL,
    NULL,
    ngx_http_upstream_healthcheck_exit_process,
    NULL,
    NGX_MODULE_V1_PADDING
};

ngx_module_t ngx_stream_upstream_healthcheck_module = {
    NGX_MODULE_V1,
    &ngx_stream_upstream_healthcheck_module_ctx,
    ngx_stream_upstream_healthcheck_commands,
    NGX_STREAM_MODULE,
    NULL,
    NULL,
    ngx_stream_upstream_healthcheck_init_process,
    NULL,
    NULL,
    ngx_stream_upstream_healthcheck_exit_process,
    NULL,
    NGX_MODULE_V1_PADDING
};

static void *
ngx_http_upstream_healthcheck_create_main_conf(ngx_conf_t *cf)
{
    ngx_http_upstream_healthcheck_main_conf_t  *conf;

    conf = ngx_pcalloc(cf->pool, sizeof(ngx_http_upstream_healthcheck_main_conf_t));
    if (conf == NULL) {
        return NULL;
    }

    if (ngx_array_init(&conf->upstreams, cf->pool, 4,
                       sizeof(ngx_http_upstream_healthcheck_conf_t *))
        != NGX_OK)
    {
        return NULL;
    }

    return conf;
}

static void *
ngx_http_upstream_healthcheck_create_srv_conf(ngx_conf_t *cf)
{
    ngx_http_upstream_healthcheck_conf_t  *conf;

    conf = ngx_pcalloc(cf->pool, sizeof(ngx_http_upstream_healthcheck_conf_t));
    if (conf == NULL) {
        return NULL;
    }

    conf->interval = 5000;
    conf->timeout = 1000;
    conf->shm_size = 1024 * 1024;
    conf->fall = 3;
    conf->rise = 2;
    conf->concurrency = 16;
    conf->type = NGX_HTTP_UPSTREAM_HEALTHCHECK_HTTP;
    ngx_str_set(&conf->uri, "/");

    return conf;
}

static void *
ngx_http_upstream_healthcheck_create_loc_conf(ngx_conf_t *cf)
{
    return ngx_pcalloc(cf->pool, 1);
}

static char *
ngx_http_upstream_healthcheck_directive(ngx_conf_t *cf, ngx_command_t *cmd,
    void *conf)
{
    ngx_http_upstream_healthcheck_conf_t  *hc = conf;
    ngx_str_t                           *value, key, val;
    u_char                              *equal;
    ngx_uint_t                           seen, i, n;
    ngx_msec_t                           msec;
    ngx_uint_t                           number;
    ssize_t                              size;

    if (hc->enabled) {
        ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                           "duplicate healthcheck directive");
        return NGX_CONF_ERROR;
    }

    value = cf->args->elts;
    seen = 0;

    for (i = 1; i < cf->args->nelts; i++) {
        equal = ngx_strlchr(value[i].data, value[i].data + value[i].len, '=');
        if (equal == NULL || equal == value[i].data
            || equal == value[i].data + value[i].len - 1)
        {
            ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                               "healthcheck parameter must be key=value");
            return NGX_CONF_ERROR;
        }

        key.data = value[i].data;
        key.len = equal - value[i].data;
        val.data = equal + 1;
        val.len = value[i].len - key.len - 1;

        if (key.len == sizeof("type") - 1
            && ngx_strncmp(key.data, "type", key.len) == 0)
        {
            n = 1;
            if (val.len == sizeof("http") - 1
                && ngx_strncmp(val.data, "http", val.len) == 0)
            {
                hc->type = NGX_HTTP_UPSTREAM_HEALTHCHECK_HTTP;
            } else if (val.len == sizeof("https") - 1
                       && ngx_strncmp(val.data, "https", val.len) == 0)
            {
                hc->type = NGX_HTTP_UPSTREAM_HEALTHCHECK_HTTPS;
            } else if (val.len == sizeof("tls") - 1
                       && ngx_strncmp(val.data, "tls", val.len) == 0)
            {
                hc->type = NGX_HTTP_UPSTREAM_HEALTHCHECK_TLS_ONLY;
            } else if (val.len == sizeof("tcp") - 1
                       && ngx_strncmp(val.data, "tcp", val.len) == 0)
            {
                hc->type = NGX_HTTP_UPSTREAM_HEALTHCHECK_TCP;
            } else {
                n = 0;
            }
        } else if (key.len == sizeof("interval") - 1
                   && ngx_strncmp(key.data, "interval", key.len) == 0)
        {
            n = 2;
            if (ngx_http_upstream_healthcheck_parse_msec(&val, &msec) != NGX_OK) {
                n = 0;
            } else {
                hc->interval = msec;
            }
        } else if (key.len == sizeof("timeout") - 1
                   && ngx_strncmp(key.data, "timeout", key.len) == 0)
        {
            n = 4;
            if (ngx_http_upstream_healthcheck_parse_msec(&val, &msec) != NGX_OK) {
                n = 0;
            } else {
                hc->timeout = msec;
            }
        } else if (key.len == sizeof("fall") - 1
                   && ngx_strncmp(key.data, "fall", key.len) == 0)
        {
            n = 8;
            if (ngx_http_upstream_healthcheck_parse_uint(
                    &val, 1, NGX_HTTP_UPSTREAM_HEALTHCHECK_MAX_STREAK,
                    &number) != NGX_OK)
            {
                n = 0;
            } else {
                hc->fall = number;
            }
        } else if (key.len == sizeof("rise") - 1
                   && ngx_strncmp(key.data, "rise", key.len) == 0)
        {
            n = 16;
            if (ngx_http_upstream_healthcheck_parse_uint(
                    &val, 1, NGX_HTTP_UPSTREAM_HEALTHCHECK_MAX_STREAK,
                    &number) != NGX_OK)
            {
                n = 0;
            } else {
                hc->rise = number;
            }
        } else if (key.len == sizeof("uri") - 1
                   && ngx_strncmp(key.data, "uri", key.len) == 0)
        {
            n = 32;
            if (val.data[0] != '/' || ngx_http_upstream_healthcheck_has_crlf(&val)
                != NGX_OK)
            {
                n = 0;
            } else if (ngx_http_upstream_healthcheck_copy_string(
                           cf->pool, &hc->uri, &val) != NGX_OK)
            {
                return NGX_CONF_ERROR;
            }
        } else if (key.len == sizeof("host") - 1
                   && ngx_strncmp(key.data, "host", key.len) == 0)
        {
            n = 64;
            if (ngx_http_upstream_healthcheck_has_crlf(&val) != NGX_OK) {
                n = 0;
            } else if (ngx_http_upstream_healthcheck_copy_string(
                           cf->pool, &hc->host, &val) != NGX_OK)
            {
                return NGX_CONF_ERROR;
            }
        } else if (key.len == sizeof("shm_size") - 1
                   && ngx_strncmp(key.data, "shm_size", key.len) == 0)
        {
            n = 128;
            size = ngx_parse_size(&val);
            if (size < (ssize_t) (8 * ngx_pagesize)) {
                n = 0;
            } else {
                hc->shm_size = size;
            }
        } else if (key.len == sizeof("concurrency") - 1
                   && ngx_strncmp(key.data, "concurrency", key.len) == 0)
        {
            n = 256;
            if (ngx_http_upstream_healthcheck_parse_uint(
                    &val, 1, NGX_HTTP_UPSTREAM_HEALTHCHECK_MAX_CONCURRENCY,
                    &number) != NGX_OK)
            {
                n = 0;
            } else {
                hc->concurrency = number;
            }
        } else {
            ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                               "unknown healthcheck parameter \"%V\"", &key);
            return NGX_CONF_ERROR;
        }

        if (n == 0) {
            ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                               "invalid healthcheck parameter \"%V\"", &key);
            return NGX_CONF_ERROR;
        }

        if (seen & n) {
            ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                               "duplicate healthcheck parameter \"%V\"", &key);
            return NGX_CONF_ERROR;
        }

        seen |= n;
    }

    if (hc->timeout == 0 || hc->interval == 0) {
        ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                           "healthcheck interval and timeout must be positive");
        return NGX_CONF_ERROR;
    }

    hc->enabled = 1;
    hc->uscf = ngx_http_conf_get_module_srv_conf(cf,
                                                  ngx_http_upstream_module);
    return NGX_CONF_OK;
}

static char *
ngx_http_upstream_healthcheck_statuses(ngx_conf_t *cf, ngx_command_t *cmd,
    void *conf)
{
    ngx_http_upstream_healthcheck_conf_t  *hc = conf;
    ngx_str_t                           *value;
    ngx_uint_t                           i, j, status, *slot;

    if (hc->statuses_set) {
        ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                           "duplicate healthcheck_statuses directive");
        return NGX_CONF_ERROR;
    }

    hc->statuses = ngx_array_create(cf->pool, cf->args->nelts - 1,
                                    sizeof(ngx_uint_t));
    if (hc->statuses == NULL) {
        return NGX_CONF_ERROR;
    }

    value = cf->args->elts;
    for (i = 1; i < cf->args->nelts; i++) {
        if (ngx_http_upstream_healthcheck_parse_uint(&value[i], 200, 599,
                                                   &status) != NGX_OK)
        {
            ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                               "healthcheck status must be between 200 and 599");
            return NGX_CONF_ERROR;
        }

        for (j = 0; j < hc->statuses->nelts; j++) {
            slot = hc->statuses->elts;
            if (slot[j] == status) {
                ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                                   "duplicate healthcheck status \"%ui\"",
                                   status);
                return NGX_CONF_ERROR;
            }
        }

        slot = ngx_array_push(hc->statuses);
        if (slot == NULL) {
            return NGX_CONF_ERROR;
        }

        *slot = status;
    }

    hc->statuses_set = 1;
    return NGX_CONF_OK;
}

static char *
ngx_http_upstream_healthcheck_tls_name(ngx_conf_t *cf, ngx_command_t *cmd,
    void *conf)
{
    ngx_http_upstream_healthcheck_conf_t  *hc = conf;
    ngx_str_t                           *value;

    if (hc->tls_name_set) {
        ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                           "duplicate healthcheck_tls_name directive");
        return NGX_CONF_ERROR;
    }

    value = cf->args->elts;
    if (value[1].len == 0
        || ngx_http_upstream_healthcheck_has_crlf(&value[1]) != NGX_OK
        || ngx_http_upstream_healthcheck_copy_string(cf->pool, &hc->tls_name,
                                                    &value[1]) != NGX_OK)
    {
        ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                           "invalid healthcheck_tls_name");
        return NGX_CONF_ERROR;
    }

    hc->tls_name_set = 1;
    return NGX_CONF_OK;
}

static char *
ngx_http_upstream_healthcheck_trusted_certificate(ngx_conf_t *cf,
    ngx_command_t *cmd, void *conf)
{
    ngx_http_upstream_healthcheck_conf_t  *hc = conf;
    ngx_str_t                           *value;

    if (hc->trusted_certificate_set) {
        ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                           "duplicate healthcheck_trusted_certificate directive");
        return NGX_CONF_ERROR;
    }

    value = cf->args->elts;
    if (value[1].len == 0
        || ngx_http_upstream_healthcheck_copy_string(
               cf->pool, &hc->trusted_certificate, &value[1]) != NGX_OK)
    {
        ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                           "invalid healthcheck_trusted_certificate");
        return NGX_CONF_ERROR;
    }

    hc->trusted_certificate_set = 1;
    return NGX_CONF_OK;
}

static char *
ngx_http_upstream_healthcheck_metrics_directive(ngx_conf_t *cf,
    ngx_command_t *cmd, void *conf)
{
    ngx_http_core_loc_conf_t  *clcf;

    clcf = ngx_http_conf_get_module_loc_conf(cf, ngx_http_core_module);
    if (clcf->handler != NULL) {
        ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                           "healthcheck_metrics cannot replace another content handler");
        return NGX_CONF_ERROR;
    }

    clcf->handler = ngx_http_upstream_healthcheck_metrics;
    return NGX_CONF_OK;
}

static ngx_int_t
ngx_http_upstream_healthcheck_postconfiguration(ngx_conf_t *cf)
{
    ngx_http_upstream_healthcheck_main_conf_t  *mcf;
    ngx_http_upstream_main_conf_t            *umcf;
    ngx_http_upstream_srv_conf_t            **uscfp;
    ngx_http_upstream_healthcheck_conf_t       *hc, **slot;
    ngx_http_upstream_server_t               *servers;
    ngx_uint_t                                i, j, k;
    ngx_str_t                                 zone_name;
    u_char                                   *p;
    ngx_uint_t                               *status;

    mcf = ngx_http_conf_get_module_main_conf(cf,
                                              ngx_http_upstream_healthcheck_module);
    umcf = ngx_http_conf_get_module_main_conf(cf, ngx_http_upstream_module);
    uscfp = umcf->upstreams.elts;

    for (i = 0; i < umcf->upstreams.nelts; i++) {
        hc = uscfp[i]->srv_conf[ngx_http_upstream_healthcheck_module.ctx_index];
        if (hc == NULL || !hc->enabled) {
            if (hc != NULL && (hc->statuses_set || hc->tls_name_set
                               || hc->trusted_certificate_set))
            {
                ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                                   "healthcheck settings require healthcheck");
                return NGX_ERROR;
            }
            continue;
        }

        hc->uscf = uscfp[i];
        hc->config_pool = cf->pool;
        if (uscfp[i]->shm_zone == NULL) {
            ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                               "healthcheck requires an upstream zone in \"%V\"",
                               &uscfp[i]->host);
            return NGX_ERROR;
        }

        if (hc->statuses == NULL) {
            hc->statuses = ngx_array_create(cf->pool, 1, sizeof(ngx_uint_t));
            if (hc->statuses == NULL) {
                return NGX_ERROR;
            }
            status = ngx_array_push(hc->statuses);
            if (status == NULL) {
                return NGX_ERROR;
            }
            *status = 200;
        }

        if (hc->type == NGX_HTTP_UPSTREAM_HEALTHCHECK_HTTPS
            || hc->type == NGX_HTTP_UPSTREAM_HEALTHCHECK_TLS_ONLY)
        {
#if (NGX_HTTP_SSL)
            if (!hc->tls_name_set || !hc->trusted_certificate_set) {
                ngx_conf_log_error(
                    NGX_LOG_EMERG, cf, 0,
                    hc->type == NGX_HTTP_UPSTREAM_HEALTHCHECK_TLS_ONLY
                        ? "TLS healthcheck requires healthcheck_tls_name and healthcheck_trusted_certificate for \"%V\""
                        : "HTTPS healthcheck requires healthcheck_tls_name and healthcheck_trusted_certificate for \"%V\"",
                    &uscfp[i]->host);
                return NGX_ERROR;
            }

            if (ngx_upstream_healthcheck_ssl_context(
                    cf, &hc->ssl, &hc->trusted_certificate,
                    &uscfp[i]->host) != NGX_OK)
            {
                return NGX_ERROR;
            }
#else
            ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                               hc->type == NGX_HTTP_UPSTREAM_HEALTHCHECK_TLS_ONLY
                                   ? "TLS healthcheck requires nginx HTTP SSL support"
                                   : "HTTPS healthcheck requires nginx HTTP SSL support");
            return NGX_ERROR;
#endif
        } else if (hc->tls_name_set || hc->trusted_certificate_set) {
            ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                               "TLS healthcheck settings require type=https or type=tls in \"%V\"",
                               &uscfp[i]->host);
            return NGX_ERROR;
        }

        servers = uscfp[i]->servers->elts;
        for (j = 0; j < uscfp[i]->servers->nelts; j++) {
            for (k = 0; k < j; k++) {
                if (servers[j].backup == servers[k].backup
                    && ngx_http_upstream_healthcheck_same_string(
                           &servers[j].name, &servers[k].name))
                {
                    ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                                       "duplicate server address \"%V\" in healthchecked upstream \"%V\"",
                                       &servers[j].name, &uscfp[i]->host);
                    return NGX_ERROR;
                }
            }
        }

        zone_name.len = sizeof("nginx_healthcheck_") - 1
                        + uscfp[i]->host.len;
        if (zone_name.len > NGX_MAX_PATH) {
            ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                               "healthcheck upstream name is too long: \"%V\"",
                               &uscfp[i]->host);
            return NGX_ERROR;
        }
        zone_name.data = ngx_pnalloc(cf->pool, zone_name.len);
        if (zone_name.data == NULL) {
            return NGX_ERROR;
        }
        p = ngx_cpymem(zone_name.data, "nginx_healthcheck_",
                       sizeof("nginx_healthcheck_") - 1);
        ngx_memcpy(p, uscfp[i]->host.data, uscfp[i]->host.len);
        hc->zone_name = zone_name;

        hc->shm_zone = ngx_shared_memory_add(cf, &hc->zone_name,
                                              hc->shm_size,
                                              &ngx_http_upstream_healthcheck_module);
        if (hc->shm_zone == NULL) {
            return NGX_ERROR;
        }
        hc->shm_zone->init = ngx_http_upstream_healthcheck_init_zone;
        hc->shm_zone->data = hc;
        hc->shm_zone->noreuse = 1;

        slot = ngx_array_push(&mcf->upstreams);
        if (slot == NULL) {
            return NGX_ERROR;
        }
        *slot = hc;
    }

    return NGX_OK;
}

static ngx_int_t
ngx_http_upstream_healthcheck_init_zone(ngx_shm_zone_t *shm_zone, void *data)
{
    ngx_http_upstream_healthcheck_conf_t   *conf;
    ngx_http_upstream_healthcheck_state_t  *state;
    ngx_slab_pool_t                      *shpool;

    conf = shm_zone->data;
    shpool = (ngx_slab_pool_t *) shm_zone->shm.addr;

    conf->peers = conf->uscf->peer.data;
    if (conf->peers == NULL || conf->peers->shpool == NULL
        || conf->peers->shpool
           != (ngx_slab_pool_t *) conf->uscf->shm_zone->shm.addr)
    {
        ngx_log_error(NGX_LOG_EMERG, shm_zone->shm.log, 0,
                      "healthcheck requires a built-in round-robin-compatible balancer and shared upstream zone in \"%V\"",
                      &conf->uscf->host);
        return NGX_ERROR;
    }

    if (shm_zone->shm.exists) {
        conf->state = shpool->data;
        return NGX_OK;
    }

    state = ngx_slab_calloc(shpool,
                             sizeof(ngx_http_upstream_healthcheck_state_t));
    if (state == NULL) {
        ngx_log_error(NGX_LOG_EMERG, shm_zone->shm.log, 0,
                      "cannot allocate healthcheck state in zone \"%V\"",
                      &shm_zone->shm.name);
        return NGX_ERROR;
    }

    state->shpool = shpool;
    shpool->data = state;
    conf->state = state;

    if (ngx_http_upstream_healthcheck_prepare_zone(conf) != NGX_OK) {
        return NGX_ERROR;
    }

    return NGX_OK;
}

static ngx_int_t
ngx_http_upstream_healthcheck_prepare_zone(
    ngx_http_upstream_healthcheck_conf_t *conf)
{
    ngx_http_upstream_rr_peers_t          *group;
    ngx_http_upstream_rr_peer_t           *peer;
    ngx_http_upstream_healthcheck_state_t   *state;
    ngx_http_upstream_healthcheck_template_t **templatep, *template;
    ngx_uint_t                             backup;

    state = conf->state;
    group = conf->peers;
    backup = 0;
    conf->templates = NULL;

    do {
        ngx_http_upstream_rr_peers_rlock(group);
        for (peer = group->resolve; peer != NULL; peer = peer->next) {
            template = ngx_pcalloc(conf->config_pool,
                                   sizeof(ngx_http_upstream_healthcheck_template_t));
            if (template == NULL) {
                ngx_http_upstream_rr_peers_unlock(group);
                return NGX_ERROR;
            }

            template->peer = peer;
            template->backup = backup;
            template->admin_down = ngx_http_upstream_healthcheck_admin_down(
                                       conf, peer, backup);
            templatep = &conf->templates;
            while (*templatep != NULL) {
                templatep = &(*templatep)->next;
            }
            *templatep = template;
        }
        ngx_http_upstream_rr_peers_unlock(group);

        group = group->next;
        backup = 1;

    } while (group != NULL);

    group = conf->peers;
    backup = 0;

    do {
        ngx_http_upstream_rr_peers_wlock(group);
        ngx_shmtx_lock(&state->shpool->mutex);

        for (peer = group->peer; peer != NULL; peer = peer->next) {
            if (ngx_http_upstream_healthcheck_add_peer_locked(
                    conf, group, peer, backup) == NULL)
            {
                state->errors_total++;
                state->memory_failures_total++;
                if (!ngx_http_upstream_healthcheck_admin_down(conf, peer,
                                                             backup))
                {
                    ngx_http_upstream_healthcheck_set_down(group, peer);
                }
                ngx_http_upstream_healthcheck_log_memory_failure(conf, state);
                continue;
            }

            if (!ngx_http_upstream_healthcheck_admin_down(conf, peer, backup)) {
                ngx_http_upstream_healthcheck_set_down(group, peer);
            }
        }

        for (peer = group->resolve; peer != NULL; peer = peer->next) {
            if (!ngx_http_upstream_healthcheck_admin_down(conf, peer, backup)) {
                peer->down = 1;
            }
        }

        ngx_shmtx_unlock(&state->shpool->mutex);
        ngx_http_upstream_rr_peers_unlock(group);

        group = group->next;
        backup = 1;

    } while (group != NULL);

    return NGX_OK;
}

static ngx_int_t
ngx_http_upstream_healthcheck_init_process(ngx_cycle_t *cycle)
{
    ngx_http_upstream_healthcheck_main_conf_t  *mcf;
    ngx_http_upstream_healthcheck_conf_t      **hcs;
    ngx_http_upstream_healthcheck_conf_t       *hc;
    ngx_http_upstream_healthcheck_state_t     *state;
    ngx_http_upstream_healthcheck_peer_t       *peer_state;
    ngx_uint_t                                i;

    if (ngx_process != NGX_PROCESS_WORKER && ngx_process != NGX_PROCESS_SINGLE) {
        return NGX_OK;
    }

    if (ngx_worker != 0) {
        return NGX_OK;
    }

    mcf = ngx_http_cycle_get_module_main_conf(cycle,
                                              ngx_http_upstream_healthcheck_module);
    if (mcf == NULL) {
        return NGX_OK;
    }

    hcs = mcf->upstreams.elts;
    for (i = 0; i < mcf->upstreams.nelts; i++) {
        hc = hcs[i];
        state = hc->state;
        if (state == NULL) {
            ngx_log_error(NGX_LOG_ALERT, cycle->log, 0,
                          "healthcheck state is unavailable for upstream \"%V\"",
                          &hc->uscf->host);
            return NGX_ERROR;
        }

        ngx_shmtx_lock(&state->shpool->mutex);
        if (state->worker_pid != ngx_pid) {
            state->active_probes = 0;
            for (peer_state = state->peers; peer_state != NULL;
                 peer_state = peer_state->next)
            {
                if (peer_state->busy) {
                    peer_state->busy = 0;
                    peer_state->next_due = 0;
                }
            }
            state->worker_pid = ngx_pid;
        }
        ngx_shmtx_unlock(&state->shpool->mutex);

        hc->stopping = 0;
        ngx_memzero(&hc->timer, sizeof(ngx_event_t));
        hc->timer.data = hc;
        hc->timer.handler = ngx_http_upstream_healthcheck_timer;
        hc->timer.log = cycle->log;
        hc->timer.cancelable = 1;
        ngx_http_upstream_healthcheck_timer(&hc->timer);
    }

    return NGX_OK;
}

static void
ngx_http_upstream_healthcheck_exit_process(ngx_cycle_t *cycle)
{
    ngx_http_upstream_healthcheck_main_conf_t  *mcf;
    ngx_http_upstream_healthcheck_conf_t      **hcs;
    ngx_http_upstream_healthcheck_probe_t       *probe;
    ngx_uint_t                                i;

    if (ngx_worker != 0) {
        return;
    }

    mcf = ngx_http_cycle_get_module_main_conf(cycle,
                                              ngx_http_upstream_healthcheck_module);
    if (mcf == NULL) {
        return;
    }

    hcs = mcf->upstreams.elts;
    for (i = 0; i < mcf->upstreams.nelts; i++) {
        hcs[i]->stopping = 1;
        if (hcs[i]->timer.timer_set) {
            ngx_del_timer(&hcs[i]->timer);
        }
        while (hcs[i]->probes != NULL) {
            probe = hcs[i]->probes;
            ngx_http_upstream_healthcheck_probe_finish(probe, 0, 0, 0, 0);
        }
    }
}

static void
ngx_http_upstream_healthcheck_timer(ngx_event_t *ev)
{
    ngx_http_upstream_healthcheck_conf_t  *conf;

    conf = ev->data;
    if (conf->stopping || ngx_exiting) {
        conf->stopping = 1;
        return;
    }

    ngx_http_upstream_healthcheck_scan(conf);
    ngx_http_upstream_healthcheck_schedule(conf);

    if (!conf->stopping && !ngx_exiting) {
        ngx_add_timer(&conf->timer, conf->interval);
    }
}

static void
ngx_http_upstream_healthcheck_scan(
    ngx_http_upstream_healthcheck_conf_t *conf)
{
    ngx_http_upstream_healthcheck_state_t  *state;
    ngx_uint_t                            generation;
    ngx_http_upstream_rr_peers_t         *group;

    state = conf->state;
    ngx_shmtx_lock(&state->shpool->mutex);
    generation = ++state->scan_generation;
    if (generation == 0) {
        generation = ++state->scan_generation;
        for (ngx_http_upstream_healthcheck_peer_t *peer_state = state->peers;
             peer_state != NULL; peer_state = peer_state->next)
        {
            peer_state->seen_generation = 0;
        }
    }
    ngx_shmtx_unlock(&state->shpool->mutex);

    group = conf->peers;
    ngx_http_upstream_healthcheck_scan_group(conf, group, 0, generation);
    if (group->next != NULL) {
        ngx_http_upstream_healthcheck_scan_group(conf, group->next, 1,
                                                generation);
    }

    ngx_shmtx_lock(&state->shpool->mutex);
    state->last_scan = ngx_time();
    state->worker_pid = ngx_pid;
    ngx_shmtx_unlock(&state->shpool->mutex);
}

static void
ngx_http_upstream_healthcheck_scan_group(
    ngx_http_upstream_healthcheck_conf_t *conf,
    ngx_http_upstream_rr_peers_t *group, ngx_uint_t backup,
    ngx_uint_t generation)
{
    ngx_http_upstream_healthcheck_state_t  *state;
    ngx_http_upstream_healthcheck_peer_t   *peer_state, *next;
    ngx_http_upstream_rr_peer_t          *peer;

    state = conf->state;
    ngx_http_upstream_rr_peers_wlock(group);
    ngx_shmtx_lock(&state->shpool->mutex);

    for (peer = group->peer; peer != NULL; peer = peer->next) {
        peer_state = ngx_http_upstream_healthcheck_find_peer(state, group, peer);
        if (peer_state == NULL) {
            peer_state = ngx_http_upstream_healthcheck_add_peer_locked(
                             conf, group, peer, backup);
            if (peer_state == NULL) {
                state->errors_total++;
                state->memory_failures_total++;
                if (!ngx_http_upstream_healthcheck_admin_down(conf, peer,
                                                             backup))
                {
                    ngx_http_upstream_healthcheck_set_down(group, peer);
                }
                ngx_http_upstream_healthcheck_log_memory_failure(conf, state);
                continue;
            }
        }

        peer_state->seen_generation = generation;
        if (peer_state->admin_down) {
            ngx_http_upstream_healthcheck_set_down(group, peer);
            continue;
        }

        if (peer_state->ready) {
            ngx_http_upstream_healthcheck_set_up(group, peer);
        } else {
            ngx_http_upstream_healthcheck_set_down(group, peer);
        }
    }

    peer_state = state->peers;
    while (peer_state != NULL) {
        next = peer_state->next;
        if (peer_state->group == group
            && peer_state->seen_generation != generation)
        {
            peer_state->removed = 1;
            if (!peer_state->busy) {
                ngx_http_upstream_healthcheck_remove_peer_locked(state,
                                                                peer_state);
            }
        }
        peer_state = next;
    }

    ngx_shmtx_unlock(&state->shpool->mutex);
    ngx_http_upstream_rr_peers_unlock(group);
}

static void
ngx_http_upstream_healthcheck_schedule(
    ngx_http_upstream_healthcheck_conf_t *conf)
{
    ngx_http_upstream_healthcheck_state_t  *state;
    ngx_http_upstream_healthcheck_peer_t   *peer_state;
    ngx_http_upstream_rr_peers_t         *group;
    ngx_http_upstream_rr_peer_t          *peer;
    ngx_msec_t                            now;
    ngx_uint_t                            after_id, started, capped;
    time_t                                log_time;

    state = conf->state;
    now = ngx_current_msec;
    after_id = state->schedule_cursor;
    started = 0;
    capped = 0;

    while (started < conf->concurrency) {
        ngx_shmtx_lock(&state->shpool->mutex);
        if (state->active_probes >= conf->concurrency) {
            ngx_shmtx_unlock(&state->shpool->mutex);
            capped = 1;
            break;
        }
        peer_state = ngx_http_upstream_healthcheck_next_due(state, now,
                                                           after_id);
        ngx_shmtx_unlock(&state->shpool->mutex);
        if (peer_state == NULL) {
            break;
        }

        group = peer_state->group;
        peer = peer_state->peer;
        ngx_http_upstream_rr_peers_wlock(group);
        ngx_shmtx_lock(&state->shpool->mutex);

        if (peer_state->removed || peer_state->busy || peer_state->admin_down
            || now < peer_state->next_due || peer->zombie)
        {
            after_id = peer_state->id;
            if (peer->zombie && !peer_state->busy) {
                peer_state->removed = 1;
                ngx_http_upstream_healthcheck_remove_peer_locked(state,
                                                                peer_state);
            }
            ngx_shmtx_unlock(&state->shpool->mutex);
            ngx_http_upstream_rr_peers_unlock(group);
            continue;
        }

        if (state->active_probes >= conf->concurrency) {
            ngx_shmtx_unlock(&state->shpool->mutex);
            ngx_http_upstream_rr_peers_unlock(group);
            capped = 1;
            break;
        }

        peer_state->busy = 1;
        peer_state->next_due = now + conf->interval;
        state->active_probes++;
        state->schedule_cursor = peer_state->id;
        after_id = peer_state->id;

        ngx_shmtx_unlock(&state->shpool->mutex);
        ngx_http_upstream_rr_peers_unlock(group);

        started++;
        if (ngx_http_upstream_healthcheck_start(conf, peer_state) != NGX_OK) {
            ngx_http_upstream_healthcheck_start_failed(conf, peer_state, 1);
        }
    }

    if (started == conf->concurrency || capped) {
        ngx_shmtx_lock(&state->shpool->mutex);
        if (ngx_http_upstream_healthcheck_next_due(state, now,
                                                  state->schedule_cursor) != NULL)
        {
            state->concurrency_limited_total++;
            log_time = ngx_time();
            if (state->last_limit_log == 0
                || log_time - state->last_limit_log >= 60)
            {
                state->last_limit_log = log_time;
                ngx_log_error(NGX_LOG_WARN, ngx_cycle->log, 0,
                              "healthcheck concurrency limit reached for upstream \"%V\" with limit %ui",
                              &conf->uscf->host, conf->concurrency);
            }
        }
        ngx_shmtx_unlock(&state->shpool->mutex);
    }
}

static ngx_http_upstream_healthcheck_peer_t *
ngx_http_upstream_healthcheck_next_due(
    ngx_http_upstream_healthcheck_state_t *state, ngx_msec_t now,
    ngx_uint_t after_id)
{
    ngx_http_upstream_healthcheck_peer_t  *peer_state;
    ngx_http_upstream_healthcheck_peer_t  *after, *wrapped;

    after = NULL;
    wrapped = NULL;

    for (peer_state = state->peers; peer_state != NULL;
         peer_state = peer_state->next)
    {
        if (peer_state->removed || peer_state->busy || peer_state->admin_down
            || now < peer_state->next_due)
        {
            continue;
        }

        if (peer_state->id > after_id) {
            if (after == NULL || peer_state->id < after->id) {
                after = peer_state;
            }
        } else if (wrapped == NULL || peer_state->id < wrapped->id) {
            wrapped = peer_state;
        }
    }

    return after != NULL ? after : wrapped;
}

static ngx_http_upstream_healthcheck_peer_t *
ngx_http_upstream_healthcheck_find_peer(
    ngx_http_upstream_healthcheck_state_t *state,
    ngx_http_upstream_rr_peers_t *group,
    ngx_http_upstream_rr_peer_t *peer)
{
    ngx_http_upstream_healthcheck_peer_t  *peer_state;

    for (peer_state = state->peers; peer_state != NULL;
         peer_state = peer_state->next)
    {
        if (peer_state->group == group && peer_state->peer == peer) {
            return peer_state;
        }
    }

    return NULL;
}

static ngx_http_upstream_healthcheck_peer_t *
ngx_http_upstream_healthcheck_add_peer_locked(
    ngx_http_upstream_healthcheck_conf_t *conf,
    ngx_http_upstream_rr_peers_t *group,
    ngx_http_upstream_rr_peer_t *peer, ngx_uint_t backup)
{
    ngx_http_upstream_healthcheck_state_t  *state;
    ngx_http_upstream_healthcheck_peer_t   *peer_state;

    state = conf->state;
    peer_state = ngx_slab_calloc_locked(state->shpool,
                                        sizeof(ngx_http_upstream_healthcheck_peer_t));
    if (peer_state == NULL) {
        return NULL;
    }

    peer_state->group = group;
    peer_state->peer = peer;
    peer_state->id = ++state->next_peer_id;
    if (peer_state->id == 0) {
        peer_state->id = ++state->next_peer_id;
    }
    peer_state->backup = backup;
    peer_state->admin_down = ngx_http_upstream_healthcheck_admin_down(
                                 conf, peer, backup);
    peer_state->last_result = (ngx_uint_t) -1;
    peer_state->last_change = ngx_time();

    ngx_http_upstream_rr_peer_ref(group, peer);

    if (state->tail == NULL) {
        state->peers = peer_state;
    } else {
        state->tail->next = peer_state;
    }
    state->tail = peer_state;
    state->peer_count++;

    if (!peer_state->admin_down) {
        ngx_http_upstream_healthcheck_set_down(group, peer);
    }

    return peer_state;
}

static void
ngx_http_upstream_healthcheck_remove_peer_locked(
    ngx_http_upstream_healthcheck_state_t *state,
    ngx_http_upstream_healthcheck_peer_t *peer_state)
{
    ngx_http_upstream_healthcheck_peer_t  **cursor;
    ngx_http_upstream_rr_peer_t          *peer;
    ngx_http_upstream_rr_peers_t         *group;

    cursor = &state->peers;
    while (*cursor != NULL && *cursor != peer_state) {
        cursor = &(*cursor)->next;
    }
    if (*cursor == NULL) {
        return;
    }

    *cursor = peer_state->next;
    if (state->tail == peer_state) {
        state->tail = NULL;
        if (state->peers != NULL) {
            for (ngx_http_upstream_healthcheck_peer_t *tail = state->peers;
                 tail->next != NULL; tail = tail->next)
            {
                state->tail = tail->next;
            }
            if (state->tail == NULL) {
                state->tail = state->peers;
            }
        }
    }
    state->peer_count--;
    peer = peer_state->peer;
    group = peer_state->group;
    ngx_http_upstream_rr_peer_unref(group, peer);
    ngx_slab_free_locked(state->shpool, peer_state);
}

static ngx_uint_t
ngx_http_upstream_healthcheck_admin_down(
    ngx_http_upstream_healthcheck_conf_t *conf,
    ngx_http_upstream_rr_peer_t *peer, ngx_uint_t backup)
{
    ngx_http_upstream_server_t  *servers;
    ngx_str_t                   *server_name;
    ngx_http_upstream_healthcheck_template_t *template;
    ngx_uint_t                   i;

    if (peer->host != NULL && peer->host->peer != NULL) {
        for (template = conf->templates; template != NULL;
             template = template->next)
        {
            if (template->peer == peer->host->peer) {
                return template->admin_down;
            }
        }
    }

    server_name = ngx_http_upstream_healthcheck_server_name(peer);
    servers = conf->uscf->servers->elts;

    for (i = 0; i < conf->uscf->servers->nelts; i++) {
        if (servers[i].backup == backup
            && ngx_http_upstream_healthcheck_same_string(&servers[i].name,
                                                        server_name))
        {
            return servers[i].down != 0;
        }
    }

    return 0;
}

static ngx_str_t *
ngx_http_upstream_healthcheck_server_name(ngx_http_upstream_rr_peer_t *peer)
{
    if (peer->host != NULL && peer->host->peer != NULL) {
        return &peer->host->peer->server;
    }

    return &peer->server;
}

static void
ngx_http_upstream_healthcheck_set_down(
    ngx_http_upstream_rr_peers_t *group,
    ngx_http_upstream_rr_peer_t *peer)
{
    if (!peer->down) {
        peer->down = 1;
        if (group->tries != 0) {
            group->tries--;
        }
    }
}

static void
ngx_http_upstream_healthcheck_set_up(
    ngx_http_upstream_rr_peers_t *group,
    ngx_http_upstream_rr_peer_t *peer)
{
    if (peer->down) {
        peer->down = 0;
        group->tries++;
    }
}

static void
ngx_http_upstream_healthcheck_log_memory_failure(
    ngx_http_upstream_healthcheck_conf_t *conf,
    ngx_http_upstream_healthcheck_state_t *state)
{
    time_t  now;

    now = ngx_time();
    if (state->last_memory_log == 0 || now - state->last_memory_log >= 60) {
        state->last_memory_log = now;
        ngx_log_error(NGX_LOG_ERR, ngx_cycle->log, 0,
                      "healthcheck shared memory is full for upstream \"%V\"; new peers remain down",
                      &conf->uscf->host);
    }
}

static ngx_int_t
ngx_http_upstream_healthcheck_start(
    ngx_http_upstream_healthcheck_conf_t *conf,
    ngx_http_upstream_healthcheck_peer_t *peer_state)
{
    ngx_http_upstream_healthcheck_probe_t  *probe;
    ngx_pool_t                            *pool;
    ngx_peer_connection_t                 pc;
    ngx_connection_t                     *c;
    ngx_str_t                            *host;
    ngx_int_t                             rc;
    size_t                                len;
    u_char                               *p;

    pool = ngx_create_pool(4096, ngx_cycle->log);
    if (pool == NULL) {
        return NGX_ERROR;
    }

    probe = ngx_pcalloc(pool, sizeof(ngx_http_upstream_healthcheck_probe_t));
    if (probe == NULL) {
        ngx_destroy_pool(pool);
        return NGX_ERROR;
    }

    probe->pool = pool;
    probe->conf = conf;
    probe->peer_state = peer_state;
    probe->started = ngx_current_msec;
    probe->stage = NGX_HTTP_UPSTREAM_HEALTHCHECK_CONNECT;
    probe->deadline.data = probe;
    probe->deadline.handler = ngx_http_upstream_healthcheck_probe_deadline;
    probe->deadline.log = ngx_cycle->log;
    probe->deadline.cancelable = 1;

    if (conf->type != NGX_HTTP_UPSTREAM_HEALTHCHECK_TCP
        && conf->type != NGX_HTTP_UPSTREAM_HEALTHCHECK_TLS_ONLY)
    {
        host = conf->host.len != 0 ? &conf->host
                                  : ngx_http_upstream_healthcheck_server_name(
                                        peer_state->peer);
        len = sizeof("GET ") - 1 + conf->uri.len
              + sizeof(" HTTP/1.0\r\nHost: ") - 1 + host->len
              + sizeof("\r\nConnection: close\r\n\r\n") - 1;
        probe->request = ngx_pnalloc(probe->pool, len);
        if (probe->request == NULL) {
            ngx_destroy_pool(probe->pool);
            return NGX_ERROR;
        }

        p = ngx_cpymem(probe->request, "GET ", sizeof("GET ") - 1);
        p = ngx_cpymem(p, conf->uri.data, conf->uri.len);
        p = ngx_cpymem(p, " HTTP/1.0\r\nHost: ",
                       sizeof(" HTTP/1.0\r\nHost: ") - 1);
        p = ngx_cpymem(p, host->data, host->len);
        p = ngx_cpymem(p, "\r\nConnection: close\r\n\r\n",
                       sizeof("\r\nConnection: close\r\n\r\n") - 1);
        probe->request_len = p - probe->request;
    }

    ngx_memzero(&pc, sizeof(ngx_peer_connection_t));
    pc.sockaddr = peer_state->peer->sockaddr;
    pc.socklen = peer_state->peer->socklen;
    pc.name = &peer_state->peer->name;
    pc.get = ngx_event_get_peer;
    pc.log = ngx_cycle->log;
    pc.log_error = NGX_ERROR_ERR;

    ngx_http_upstream_healthcheck_probe_link(probe);
    ngx_add_timer(&probe->deadline, conf->timeout);
    rc = ngx_event_connect_peer(&pc);
    probe->connection = pc.connection;

    if (rc != NGX_OK && rc != NGX_AGAIN) {
        if (probe->connection != NULL) {
            ngx_close_connection(probe->connection);
            probe->connection = NULL;
        }
        ngx_http_upstream_healthcheck_probe_finish(probe, 0, 0, 1,
                                                  rc == NGX_ERROR);
        return NGX_OK;
    }

    c = probe->connection;
    c->data = probe;
    c->pool = probe->pool;
    c->read->handler = ngx_http_upstream_healthcheck_probe_event;
    c->write->handler = ngx_http_upstream_healthcheck_probe_event;

    if (rc == NGX_OK) {
        ngx_http_upstream_healthcheck_probe_connected(probe);
    }

    return NGX_OK;
}

static void
ngx_http_upstream_healthcheck_probe_event(ngx_event_t *ev)
{
    ngx_connection_t                    *c;
    ngx_http_upstream_healthcheck_probe_t *probe;

    c = ev->data;
    probe = c->data;
    if (probe == NULL || probe->done) {
        return;
    }

    if (ev->timedout || ev->error || c->error) {
        ngx_http_upstream_healthcheck_probe_finish(probe, 0, 0, 1, 0);
        return;
    }

    if (probe->stage == NGX_HTTP_UPSTREAM_HEALTHCHECK_CONNECT) {
        if (!ev->write) {
            return;
        }
        if (ngx_http_upstream_healthcheck_test_connect(c) != NGX_OK) {
            ngx_http_upstream_healthcheck_probe_finish(probe, 0, 0, 1, 0);
            return;
        }
        ngx_http_upstream_healthcheck_probe_connected(probe);
        return;
    }

    if (probe->stage == NGX_HTTP_UPSTREAM_HEALTHCHECK_SENDING) {
        ngx_http_upstream_healthcheck_probe_send(probe);
        return;
    }

    if (probe->stage == NGX_HTTP_UPSTREAM_HEALTHCHECK_READING) {
        ngx_http_upstream_healthcheck_probe_read(probe);
    }
}

static void
ngx_http_upstream_healthcheck_probe_link(
    ngx_http_upstream_healthcheck_probe_t *probe)
{
    probe->next = probe->conf->probes;
    probe->prev = &probe->conf->probes;
    if (probe->next != NULL) {
        probe->next->prev = &probe->next;
    }
    probe->conf->probes = probe;
}

static void
ngx_http_upstream_healthcheck_probe_unlink(
    ngx_http_upstream_healthcheck_probe_t *probe)
{
    if (probe->prev == NULL) {
        return;
    }
    *probe->prev = probe->next;
    if (probe->next != NULL) {
        probe->next->prev = probe->prev;
    }
    probe->prev = NULL;
    probe->next = NULL;
}

static void
ngx_http_upstream_healthcheck_probe_deadline(ngx_event_t *ev)
{
    ngx_http_upstream_healthcheck_probe_t  *probe;

    probe = ev->data;
    if (probe == NULL || probe->done) {
        return;
    }

    ngx_http_upstream_healthcheck_probe_finish(probe, 0, 0, 1, 0);
}

static void
ngx_http_upstream_healthcheck_probe_connected(
    ngx_http_upstream_healthcheck_probe_t *probe)
{
    ngx_connection_t  *c;

    c = probe->connection;
    if (probe->conf->type == NGX_HTTP_UPSTREAM_HEALTHCHECK_TCP) {
        ngx_http_upstream_healthcheck_probe_finish(probe, 1, 0, 1, 0);
        return;
    }

    if (probe->conf->type == NGX_HTTP_UPSTREAM_HEALTHCHECK_HTTPS
        || probe->conf->type == NGX_HTTP_UPSTREAM_HEALTHCHECK_TLS_ONLY)
    {
        ngx_http_upstream_healthcheck_probe_start_tls(probe);
        return;
    }

    probe->stage = NGX_HTTP_UPSTREAM_HEALTHCHECK_SENDING;
    c->read->handler = ngx_http_upstream_healthcheck_probe_event;
    c->write->handler = ngx_http_upstream_healthcheck_probe_event;
    ngx_http_upstream_healthcheck_probe_send(probe);
}

static void
ngx_http_upstream_healthcheck_probe_start_tls(
    ngx_http_upstream_healthcheck_probe_t *probe)
{
#if (NGX_HTTP_SSL)
    ngx_connection_t  *c;
    ngx_int_t          rc;
    ngx_uint_t         force_down;

    c = probe->connection;
    probe->stage = NGX_HTTP_UPSTREAM_HEALTHCHECK_TLS;
    rc = ngx_upstream_healthcheck_tls_start(
             c, probe->pool, &probe->conf->ssl, &probe->conf->tls_name,
             ngx_http_upstream_healthcheck_probe_tls_done, &force_down);
    if (rc == NGX_AGAIN) {
        return;
    }
    if (rc != NGX_OK) {
        ngx_http_upstream_healthcheck_probe_finish(probe, 0, 0, 1, force_down);
        return;
    }
    ngx_http_upstream_healthcheck_probe_tls_done(c);
#else
    ngx_http_upstream_healthcheck_probe_finish(probe, 0, 0, 1, 1);
#endif
}

static void
ngx_http_upstream_healthcheck_probe_tls_done(ngx_connection_t *c)
{
#if (NGX_HTTP_SSL)
    ngx_http_upstream_healthcheck_probe_t  *probe;

    probe = c->data;
    if (probe == NULL || probe->done) {
        return;
    }

    if (ngx_upstream_healthcheck_tls_verified(
            c, &probe->conf->tls_name) != NGX_OK)
    {
        ngx_log_error(NGX_LOG_ERR, c->log, 0,
                      "healthcheck TLS certificate verification failed for \"%V\"",
                      &probe->conf->tls_name);
        ngx_http_upstream_healthcheck_probe_finish(probe, 0, 0, 1, 0);
        return;
    }

    if (probe->conf->type == NGX_HTTP_UPSTREAM_HEALTHCHECK_TLS_ONLY) {
        ngx_http_upstream_healthcheck_probe_finish(probe, 1, 0, 1, 0);
        return;
    }

    probe->stage = NGX_HTTP_UPSTREAM_HEALTHCHECK_SENDING;
    c->read->handler = ngx_http_upstream_healthcheck_probe_event;
    c->write->handler = ngx_http_upstream_healthcheck_probe_event;
    ngx_http_upstream_healthcheck_probe_send(probe);
#endif
}

static void
ngx_http_upstream_healthcheck_probe_send(
    ngx_http_upstream_healthcheck_probe_t *probe)
{
    ngx_connection_t  *c;
    ssize_t            n;

    c = probe->connection;
    while (probe->request_sent < probe->request_len) {
        n = c->send(c, probe->request + probe->request_sent,
                    probe->request_len - probe->request_sent);
        if (n == NGX_AGAIN) {
            if (ngx_handle_read_event(c->read, 0) != NGX_OK
                || ngx_handle_write_event(c->write, 0) != NGX_OK)
            {
                ngx_http_upstream_healthcheck_probe_finish(probe, 0, 0, 1, 0);
            }
            return;
        }
        if (n <= 0) {
            ngx_http_upstream_healthcheck_probe_finish(probe, 0, 0, 1, 0);
            return;
        }
        probe->request_sent += n;
    }

    probe->stage = NGX_HTTP_UPSTREAM_HEALTHCHECK_READING;
    c->read->handler = ngx_http_upstream_healthcheck_probe_event;
    c->write->handler = ngx_http_upstream_healthcheck_probe_event;
    ngx_http_upstream_healthcheck_probe_read(probe);
}

static void
ngx_http_upstream_healthcheck_probe_read(
    ngx_http_upstream_healthcheck_probe_t *probe)
{
    ngx_connection_t  *c;
    ssize_t            n;
    size_t             i;
    ngx_uint_t         status;

    c = probe->connection;
    for (;;) {
        if (probe->response_len == sizeof(probe->response)) {
            ngx_http_upstream_healthcheck_probe_finish(probe, 0, 0, 1, 0);
            return;
        }

        n = c->recv(c, probe->response + probe->response_len,
                    sizeof(probe->response) - probe->response_len);
        if (n == NGX_AGAIN) {
            if (ngx_handle_read_event(c->read, 0) != NGX_OK
                || ngx_handle_write_event(c->write, 0) != NGX_OK)
            {
                ngx_http_upstream_healthcheck_probe_finish(probe, 0, 0, 1, 0);
            }
            return;
        }
        if (n <= 0) {
            ngx_http_upstream_healthcheck_probe_finish(probe, 0, 0, 1, 0);
            return;
        }

        probe->response_len += n;
        for (i = 1; i < probe->response_len; i++) {
            if (probe->response[i - 1] == '\r'
                && probe->response[i] == '\n')
            {
                if (ngx_http_upstream_healthcheck_parse_status(
                        probe->response, i - 1, &status) != NGX_OK)
                {
                    ngx_http_upstream_healthcheck_probe_finish(probe, 0, 0,
                                                             1, 0);
                    return;
                }
                ngx_http_upstream_healthcheck_probe_finish(
                    probe,
                    ngx_http_upstream_healthcheck_status_allowed(probe->conf,
                                                                status),
                    status, 1, 0);
                return;
            }
        }
    }
}

static ngx_int_t
ngx_http_upstream_healthcheck_parse_status(u_char *line, size_t len,
    ngx_uint_t *status)
{
    if (len < sizeof("HTTP/1.1 200 ") - 1
        || (ngx_strncmp(line, "HTTP/1.0 ", sizeof("HTTP/1.0 ") - 1) != 0
            && ngx_strncmp(line, "HTTP/1.1 ", sizeof("HTTP/1.1 ") - 1) != 0)
        || line[9] < '0' || line[9] > '9'
        || line[10] < '0' || line[10] > '9'
        || line[11] < '0' || line[11] > '9'
        || line[12] != ' ')
    {
        return NGX_ERROR;
    }

    *status = (line[9] - '0') * 100 + (line[10] - '0') * 10
              + (line[11] - '0');
    if (*status < 200 || *status > 599) {
        return NGX_ERROR;
    }

    return NGX_OK;
}

static ngx_int_t
ngx_http_upstream_healthcheck_status_allowed(
    ngx_http_upstream_healthcheck_conf_t *conf, ngx_uint_t status)
{
    ngx_uint_t  i, *statuses;

    statuses = conf->statuses->elts;
    for (i = 0; i < conf->statuses->nelts; i++) {
        if (statuses[i] == status) {
            return 1;
        }
    }

    return 0;
}

static void
ngx_http_upstream_healthcheck_probe_finish(
    ngx_http_upstream_healthcheck_probe_t *probe, ngx_uint_t success,
    ngx_uint_t status, ngx_uint_t attempted, ngx_uint_t force_down)
{
    ngx_connection_t  *c;

    if (probe == NULL || probe->done) {
        return;
    }

    probe->done = 1;
    if (probe->deadline.timer_set) {
        ngx_del_timer(&probe->deadline);
    }
    ngx_http_upstream_healthcheck_probe_unlink(probe);

    c = probe->connection;
    if (c != NULL) {
        probe->connection = NULL;
#if (NGX_HTTP_SSL)
        ngx_upstream_healthcheck_tls_close(c);
#endif
        ngx_close_connection(c);
    }

    ngx_http_upstream_healthcheck_complete(probe->conf, probe->peer_state,
                                         success, status, attempted, force_down,
                                         ngx_current_msec - probe->started);
    ngx_destroy_pool(probe->pool);
}

static void
ngx_http_upstream_healthcheck_start_failed(
    ngx_http_upstream_healthcheck_conf_t *conf,
    ngx_http_upstream_healthcheck_peer_t *peer_state, ngx_uint_t force_down)
{
    ngx_http_upstream_healthcheck_state_t  *state;
    ngx_http_upstream_rr_peers_t         *group;
    ngx_http_upstream_rr_peer_t          *peer;
    state = conf->state;
    group = peer_state->group;
    peer = peer_state->peer;
    ngx_http_upstream_rr_peers_wlock(group);
    ngx_shmtx_lock(&state->shpool->mutex);

    if (peer_state->busy) {
        peer_state->busy = 0;
        if (state->active_probes != 0) {
            state->active_probes--;
        }
        peer_state->next_due = ngx_current_msec + conf->interval;

        if (peer->zombie || peer_state->removed) {
            peer_state->removed = 1;
            ngx_http_upstream_healthcheck_remove_peer_locked(state,
                                                            peer_state);
        } else if (force_down && !peer_state->admin_down) {
            state->errors_total++;
            state->memory_failures_total++;
            peer_state->rise_streak = 0;
            if (peer_state->fall_streak < NGX_HTTP_UPSTREAM_HEALTHCHECK_MAX_STREAK) {
                peer_state->fall_streak++;
            }
            if (peer_state->ready) {
                peer_state->ready = 0;
                peer_state->check_up_down_total++;
                peer_state->last_change = ngx_time();
            }
            ngx_http_upstream_healthcheck_set_down(group, peer);
            ngx_http_upstream_healthcheck_log_memory_failure(conf, state);
        }
    }

    ngx_shmtx_unlock(&state->shpool->mutex);
    ngx_http_upstream_rr_peers_unlock(group);
}

static void
ngx_http_upstream_healthcheck_complete(
    ngx_http_upstream_healthcheck_conf_t *conf,
    ngx_http_upstream_healthcheck_peer_t *peer_state, ngx_uint_t success,
    ngx_uint_t status, ngx_uint_t attempted, ngx_uint_t force_down,
    ngx_msec_t duration)
{
    ngx_http_upstream_healthcheck_state_t  *state;
    ngx_http_upstream_rr_peers_t         *group;
    ngx_http_upstream_rr_peer_t          *peer;
    time_t                                now;

    state = conf->state;
    group = peer_state->group;
    peer = peer_state->peer;
    now = ngx_time();

    ngx_http_upstream_rr_peers_wlock(group);
    ngx_shmtx_lock(&state->shpool->mutex);

    if (peer_state->busy) {
        peer_state->busy = 0;
        if (state->active_probes != 0) {
            state->active_probes--;
        }

        if (peer->zombie || peer_state->removed) {
            peer_state->removed = 1;
            ngx_http_upstream_healthcheck_remove_peer_locked(state,
                                                            peer_state);
        } else {
            peer_state->next_due = ngx_current_msec + conf->interval;
            if (attempted) {
                peer_state->has_result = 1;
                peer_state->last_result = success ? 1 : 0;
                peer_state->status_code = status;
                peer_state->last_duration = duration;
                peer_state->last_check = now;
                peer_state->checks_total++;

                if (success) {
                    peer_state->fall_streak = 0;
                    if (peer_state->rise_streak
                        < NGX_HTTP_UPSTREAM_HEALTHCHECK_MAX_STREAK)
                    {
                        peer_state->rise_streak++;
                    }
                } else {
                    peer_state->check_failures_total++;
                    peer_state->rise_streak = 0;
                    if (peer_state->fall_streak
                        < NGX_HTTP_UPSTREAM_HEALTHCHECK_MAX_STREAK)
                    {
                        peer_state->fall_streak++;
                    }
                }

                if (success && !peer_state->ready
                    && peer_state->rise_streak >= conf->rise)
                {
                    peer_state->ready = 1;
                    peer_state->last_change = now;
                    ngx_http_upstream_healthcheck_set_up(group, peer);
                } else if ((!success
                            && (force_down
                                || peer_state->fall_streak >= conf->fall))
                           && peer_state->ready)
                {
                    peer_state->ready = 0;
                    peer_state->check_up_down_total++;
                    peer_state->last_change = now;
                    ngx_http_upstream_healthcheck_set_down(group, peer);
                }
            }

            if (force_down && !peer_state->admin_down) {
                state->errors_total++;
                if (peer_state->ready) {
                    peer_state->ready = 0;
                    peer_state->check_up_down_total++;
                    peer_state->last_change = now;
                }
                ngx_http_upstream_healthcheck_set_down(group, peer);
            }
        }
    }

    ngx_shmtx_unlock(&state->shpool->mutex);
    ngx_http_upstream_rr_peers_unlock(group);
}

static ngx_int_t
ngx_http_upstream_healthcheck_test_connect(ngx_connection_t *c)
{
    int        err;
    socklen_t  len;

#if (NGX_HAVE_KQUEUE)
    if (ngx_event_flags & NGX_USE_KQUEUE_EVENT) {
        if (c->write->pending_eof || c->read->pending_eof) {
            err = c->write->pending_eof ? c->write->kq_errno
                                        : c->read->kq_errno;
            (void) ngx_connection_error(c, err,
                                        "healthcheck connect() failed");
            return NGX_ERROR;
        }
        return NGX_OK;
    }
#endif

    err = 0;
    len = sizeof(int);
    if (getsockopt(c->fd, SOL_SOCKET, SO_ERROR, (void *) &err, &len) == -1) {
        err = ngx_socket_errno;
    }
    if (err != 0) {
        (void) ngx_connection_error(c, err, "healthcheck connect() failed");
        return NGX_ERROR;
    }

    return NGX_OK;
}

static void *
ngx_stream_upstream_healthcheck_create_main_conf(ngx_conf_t *cf)
{
    ngx_stream_upstream_healthcheck_main_conf_t  *conf;

    conf = ngx_pcalloc(cf->pool,
                       sizeof(ngx_stream_upstream_healthcheck_main_conf_t));
    if (conf == NULL) {
        return NULL;
    }

    if (ngx_array_init(&conf->upstreams, cf->pool, 4,
                       sizeof(ngx_stream_upstream_healthcheck_conf_t *))
        != NGX_OK)
    {
        return NULL;
    }

    return conf;
}

static void *
ngx_stream_upstream_healthcheck_create_srv_conf(ngx_conf_t *cf)
{
    ngx_stream_upstream_healthcheck_conf_t  *conf;

    conf = ngx_pcalloc(cf->pool, sizeof(ngx_stream_upstream_healthcheck_conf_t));
    if (conf == NULL) {
        return NULL;
    }

    conf->interval = 5000;
    conf->timeout = 1000;
    conf->shm_size = 1024 * 1024;
    conf->max_response = 4096;
    conf->fall = 3;
    conf->rise = 2;
    conf->concurrency = 16;
    conf->type = NGX_STREAM_UPSTREAM_HEALTHCHECK_TCP;

    return conf;
}

static char *
ngx_stream_upstream_healthcheck_directive(ngx_conf_t *cf, ngx_command_t *cmd,
    void *conf)
{
    ngx_stream_upstream_healthcheck_conf_t  *hc = conf;
    ngx_str_t                             *value, key, val;
    u_char                                *equal;
    ngx_uint_t                             seen, i, flag, number;
    ngx_msec_t                             msec;
    ssize_t                                size;

    if (hc->enabled) {
        ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                           "duplicate healthcheck_tcp directive");
        return NGX_CONF_ERROR;
    }

    value = cf->args->elts;
    seen = 0;

    for (i = 1; i < cf->args->nelts; i++) {
        equal = ngx_strlchr(value[i].data, value[i].data + value[i].len, '=');
        if (equal == NULL || equal == value[i].data
            || equal == value[i].data + value[i].len - 1)
        {
            ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                               "healthcheck_tcp parameter must be key=value");
            return NGX_CONF_ERROR;
        }

        key.data = value[i].data;
        key.len = equal - value[i].data;
        val.data = equal + 1;
        val.len = value[i].len - key.len - 1;
        flag = 0;

        if (key.len == sizeof("type") - 1
            && ngx_strncmp(key.data, "type", key.len) == 0)
        {
            flag = 1;
            if (val.len == sizeof("tcp") - 1
                && ngx_strncmp(val.data, "tcp", val.len) == 0)
            {
                hc->type = NGX_STREAM_UPSTREAM_HEALTHCHECK_TCP;
            } else if (val.len == sizeof("tls") - 1
                       && ngx_strncmp(val.data, "tls", val.len) == 0)
            {
                hc->type = NGX_STREAM_UPSTREAM_HEALTHCHECK_TLS;
            } else {
                flag = 0;
            }
        } else if (key.len == sizeof("interval") - 1
                   && ngx_strncmp(key.data, "interval", key.len) == 0)
        {
            flag = 2;
            if (ngx_http_upstream_healthcheck_parse_msec(&val, &msec) != NGX_OK) {
                flag = 0;
            } else {
                hc->interval = msec;
            }
        } else if (key.len == sizeof("timeout") - 1
                   && ngx_strncmp(key.data, "timeout", key.len) == 0)
        {
            flag = 4;
            if (ngx_http_upstream_healthcheck_parse_msec(&val, &msec) != NGX_OK) {
                flag = 0;
            } else {
                hc->timeout = msec;
            }
        } else if (key.len == sizeof("fall") - 1
                   && ngx_strncmp(key.data, "fall", key.len) == 0)
        {
            flag = 8;
            if (ngx_http_upstream_healthcheck_parse_uint(
                    &val, 1, NGX_HTTP_UPSTREAM_HEALTHCHECK_MAX_STREAK,
                    &number) != NGX_OK)
            {
                flag = 0;
            } else {
                hc->fall = number;
            }
        } else if (key.len == sizeof("rise") - 1
                   && ngx_strncmp(key.data, "rise", key.len) == 0)
        {
            flag = 16;
            if (ngx_http_upstream_healthcheck_parse_uint(
                    &val, 1, NGX_HTTP_UPSTREAM_HEALTHCHECK_MAX_STREAK,
                    &number) != NGX_OK)
            {
                flag = 0;
            } else {
                hc->rise = number;
            }
        } else if (key.len == sizeof("concurrency") - 1
                   && ngx_strncmp(key.data, "concurrency", key.len) == 0)
        {
            flag = 32;
            if (ngx_http_upstream_healthcheck_parse_uint(
                    &val, 1, NGX_HTTP_UPSTREAM_HEALTHCHECK_MAX_CONCURRENCY,
                    &number) != NGX_OK)
            {
                flag = 0;
            } else {
                hc->concurrency = number;
            }
        } else if (key.len == sizeof("shm_size") - 1
                   && ngx_strncmp(key.data, "shm_size", key.len) == 0)
        {
            flag = 64;
            size = ngx_parse_size(&val);
            if (size < (ssize_t) (8 * ngx_pagesize)) {
                flag = 0;
            } else {
                hc->shm_size = size;
            }
        } else if (key.len == sizeof("max_response") - 1
                   && ngx_strncmp(key.data, "max_response", key.len) == 0)
        {
            flag = 128;
            size = ngx_parse_size(&val);
            if (size < 1 || size > 65536) {
                flag = 0;
            } else {
                hc->max_response = (size_t) size;
            }
        } else if (key.len == sizeof("min_recv") - 1
                   && ngx_strncmp(key.data, "min_recv", key.len) == 0)
        {
            flag = 256;
            if (ngx_http_upstream_healthcheck_parse_uint(
                    &val, 1, 65536, &number) != NGX_OK)
            {
                flag = 0;
            } else {
                hc->min_recv = number;
                hc->min_recv_set = 1;
            }
        } else {
            ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                               "unknown healthcheck_tcp parameter \"%V\"",
                               &key);
            return NGX_CONF_ERROR;
        }

        if (flag == 0) {
            ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                               "invalid healthcheck_tcp parameter \"%V\"",
                               &key);
            return NGX_CONF_ERROR;
        }
        if (seen & flag) {
            ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                               "duplicate healthcheck_tcp parameter \"%V\"",
                               &key);
            return NGX_CONF_ERROR;
        }
        seen |= flag;
    }

    if (hc->timeout == 0 || hc->interval == 0
        || hc->min_recv > hc->max_response)
    {
        ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                           "invalid healthcheck_tcp interval, timeout, or min_recv");
        return NGX_CONF_ERROR;
    }

    hc->enabled = 1;
    hc->uscf = ngx_stream_conf_get_module_srv_conf(cf,
                                                    ngx_stream_upstream_module);
    return NGX_CONF_OK;
}

static ngx_int_t
ngx_stream_upstream_healthcheck_set_payload(ngx_conf_t *cf, ngx_str_t *target,
    ngx_uint_t *set, ngx_str_t *value, ngx_uint_t hex)
{
    ngx_str_t  decoded;
    ngx_uint_t i;
    u_char     c, high, low;

    if (*set) {
        ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                           "duplicate stream healthcheck payload directive");
        return NGX_ERROR;
    }

    if (!hex) {
        if (ngx_http_upstream_healthcheck_copy_string(cf->pool, target, value)
            != NGX_OK)
        {
            return NGX_ERROR;
        }
        *set = 1;
        return NGX_OK;
    }

    if (value->len == 0 || (value->len & 1) != 0) {
        ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                           "hex stream healthcheck payload must contain pairs of digits");
        return NGX_ERROR;
    }

    decoded.len = value->len / 2;
    decoded.data = ngx_pnalloc(cf->pool, decoded.len);
    if (decoded.data == NULL) {
        return NGX_ERROR;
    }

    for (i = 0; i < decoded.len; i++) {
        c = value->data[i * 2];
        if (c >= '0' && c <= '9') {
            high = c - '0';
        } else if (c >= 'a' && c <= 'f') {
            high = c - 'a' + 10;
        } else if (c >= 'A' && c <= 'F') {
            high = c - 'A' + 10;
        } else {
            return NGX_ERROR;
        }

        c = value->data[i * 2 + 1];
        if (c >= '0' && c <= '9') {
            low = c - '0';
        } else if (c >= 'a' && c <= 'f') {
            low = c - 'a' + 10;
        } else if (c >= 'A' && c <= 'F') {
            low = c - 'A' + 10;
        } else {
            return NGX_ERROR;
        }

        decoded.data[i] = (high << 4) | low;
    }

    *target = decoded;
    *set = 1;
    return NGX_OK;
}

static char *
ngx_stream_upstream_healthcheck_payload(ngx_conf_t *cf, ngx_command_t *cmd,
    void *conf)
{
    ngx_stream_upstream_healthcheck_conf_t  *hc = conf;
    ngx_str_t                             *value;

    value = cf->args->elts;
    if (ngx_stream_upstream_healthcheck_set_payload(cf, &hc->send_data,
                                                   &hc->send_set, &value[1], 0)
        != NGX_OK)
    {
        return NGX_CONF_ERROR;
    }

    return NGX_CONF_OK;
}

static char *
ngx_stream_upstream_healthcheck_payload_hex(ngx_conf_t *cf, ngx_command_t *cmd,
    void *conf)
{
    ngx_stream_upstream_healthcheck_conf_t  *hc = conf;
    ngx_str_t                             *value;

    value = cf->args->elts;
    if (ngx_stream_upstream_healthcheck_set_payload(cf, &hc->send_data,
                                                   &hc->send_set, &value[1], 1)
        != NGX_OK)
    {
        return NGX_CONF_ERROR;
    }

    return NGX_CONF_OK;
}

static char *
ngx_stream_upstream_healthcheck_expect(ngx_conf_t *cf, ngx_command_t *cmd,
    void *conf)
{
    ngx_stream_upstream_healthcheck_conf_t  *hc = conf;
    ngx_str_t                             *value, key, val;
    u_char                                *equal;
    ngx_uint_t                             i, min_recv_seen, number;

    if (hc->expect_set) {
        ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                           "duplicate stream healthcheck expectation");
        return NGX_CONF_ERROR;
    }

    value = cf->args->elts;
    if (value[1].len == 0
        || ngx_stream_upstream_healthcheck_set_payload(
               cf, &hc->expect_data, &hc->expect_set, &value[1], 0) != NGX_OK)
    {
        return NGX_CONF_ERROR;
    }

    min_recv_seen = 0;
    for (i = 2; i < cf->args->nelts; i++) {
        equal = ngx_strlchr(value[i].data, value[i].data + value[i].len, '=');
        if (equal == NULL || equal == value[i].data
            || equal == value[i].data + value[i].len - 1)
        {
            return NGX_CONF_ERROR;
        }
        key.data = value[i].data;
        key.len = equal - value[i].data;
        val.data = equal + 1;
        val.len = value[i].len - key.len - 1;
        if (key.len != sizeof("min_recv") - 1
            || ngx_strncmp(key.data, "min_recv", key.len) != 0
            || min_recv_seen
            || hc->min_recv_set
            || ngx_http_upstream_healthcheck_parse_uint(&val, 1, 65536,
                                                       &number) != NGX_OK)
        {
            ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                               "invalid healthcheck_expect parameter \"%V\"",
                               &key);
            return NGX_CONF_ERROR;
        }
        hc->min_recv = number;
        hc->min_recv_set = 1;
        min_recv_seen = 1;
    }

    return NGX_CONF_OK;
}

static char *
ngx_stream_upstream_healthcheck_expect_hex(ngx_conf_t *cf, ngx_command_t *cmd,
    void *conf)
{
    ngx_stream_upstream_healthcheck_conf_t  *hc = conf;
    ngx_str_t                             *value, key, val;
    u_char                                *equal;
    ngx_uint_t                             i, min_recv_seen, number;

    if (hc->expect_set) {
        ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                           "duplicate stream healthcheck expectation");
        return NGX_CONF_ERROR;
    }

    value = cf->args->elts;
    if (ngx_stream_upstream_healthcheck_set_payload(
            cf, &hc->expect_data, &hc->expect_set, &value[1], 1) != NGX_OK)
    {
        return NGX_CONF_ERROR;
    }

    min_recv_seen = 0;
    for (i = 2; i < cf->args->nelts; i++) {
        equal = ngx_strlchr(value[i].data, value[i].data + value[i].len, '=');
        if (equal == NULL || equal == value[i].data
            || equal == value[i].data + value[i].len - 1)
        {
            return NGX_CONF_ERROR;
        }
        key.data = value[i].data;
        key.len = equal - value[i].data;
        val.data = equal + 1;
        val.len = value[i].len - key.len - 1;
        if (key.len != sizeof("min_recv") - 1
            || ngx_strncmp(key.data, "min_recv", key.len) != 0
            || min_recv_seen
            || hc->min_recv_set
            || ngx_http_upstream_healthcheck_parse_uint(&val, 1, 65536,
                                                       &number) != NGX_OK)
        {
            ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                               "invalid healthcheck_expect_hex parameter \"%V\"",
                               &key);
            return NGX_CONF_ERROR;
        }
        hc->min_recv = number;
        hc->min_recv_set = 1;
        min_recv_seen = 1;
    }

    return NGX_CONF_OK;
}

static char *
ngx_stream_upstream_healthcheck_tls_name(ngx_conf_t *cf, ngx_command_t *cmd,
    void *conf)
{
    ngx_stream_upstream_healthcheck_conf_t  *hc = conf;
    ngx_str_t                             *value;

    if (hc->tls_name_set) {
        ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                           "duplicate healthcheck_tls_name directive");
        return NGX_CONF_ERROR;
    }
    value = cf->args->elts;
    if (value[1].len == 0
        || ngx_http_upstream_healthcheck_has_crlf(&value[1]) != NGX_OK
        || ngx_http_upstream_healthcheck_copy_string(cf->pool, &hc->tls_name,
                                                    &value[1]) != NGX_OK)
    {
        return NGX_CONF_ERROR;
    }
    hc->tls_name_set = 1;
    return NGX_CONF_OK;
}

static char *
ngx_stream_upstream_healthcheck_trusted_certificate(ngx_conf_t *cf,
    ngx_command_t *cmd, void *conf)
{
    ngx_stream_upstream_healthcheck_conf_t  *hc = conf;
    ngx_str_t                             *value;

    if (hc->trusted_certificate_set) {
        ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                           "duplicate healthcheck_trusted_certificate directive");
        return NGX_CONF_ERROR;
    }
    value = cf->args->elts;
    if (value[1].len == 0
        || ngx_http_upstream_healthcheck_copy_string(
               cf->pool, &hc->trusted_certificate, &value[1]) != NGX_OK)
    {
        return NGX_CONF_ERROR;
    }
    hc->trusted_certificate_set = 1;
    return NGX_CONF_OK;
}

static ngx_int_t
ngx_stream_upstream_healthcheck_postconfiguration(ngx_conf_t *cf)
{
    ngx_stream_upstream_healthcheck_main_conf_t  *mcf;
    ngx_stream_upstream_main_conf_t            *umcf;
    ngx_stream_upstream_srv_conf_t            **uscfp;
    ngx_stream_upstream_healthcheck_conf_t        *hc, **slot;
    ngx_stream_upstream_server_t               *servers;
    ngx_str_t                                    zone_name;
    u_char                                      *p;
    ngx_uint_t                                   i, j, k;

    mcf = ngx_stream_conf_get_module_main_conf(
              cf, ngx_stream_upstream_healthcheck_module);
    umcf = ngx_stream_conf_get_module_main_conf(cf, ngx_stream_upstream_module);
    uscfp = umcf->upstreams.elts;

    for (i = 0; i < umcf->upstreams.nelts; i++) {
        hc = uscfp[i]->srv_conf[ngx_stream_upstream_healthcheck_module.ctx_index];
        if (hc == NULL || !hc->enabled) {
            if (hc != NULL && (hc->send_set || hc->expect_set
                               || hc->tls_name_set
                               || hc->trusted_certificate_set))
            {
                ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                                   "stream healthcheck settings require healthcheck_tcp");
                return NGX_ERROR;
            }
            continue;
        }

        hc->uscf = uscfp[i];
        hc->config_pool = cf->pool;
        if (uscfp[i]->shm_zone == NULL) {
            ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                               "healthcheck_tcp requires an upstream zone in \"%V\"",
                               &uscfp[i]->host);
            return NGX_ERROR;
        }

        if (hc->min_recv_set && hc->min_recv > hc->max_response) {
            ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                               "healthcheck min_recv exceeds max_response in \"%V\"",
                               &uscfp[i]->host);
            return NGX_ERROR;
        }
        if (hc->expect_set && !hc->min_recv_set) {
            hc->min_recv = hc->expect_data.len;
        }
        if (hc->expect_set && hc->expect_data.len > hc->max_response) {
            ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                               "healthcheck expectation exceeds max_response in \"%V\"",
                               &uscfp[i]->host);
            return NGX_ERROR;
        }

        if (hc->type == NGX_STREAM_UPSTREAM_HEALTHCHECK_TLS) {
#if (NGX_STREAM_SSL)
            if (!hc->tls_name_set || !hc->trusted_certificate_set) {
                ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                                   "TLS healthcheck requires healthcheck_tls_name and healthcheck_trusted_certificate for \"%V\"",
                                   &uscfp[i]->host);
                return NGX_ERROR;
            }

            if (ngx_upstream_healthcheck_ssl_context(
                    cf, &hc->ssl, &hc->trusted_certificate,
                    &uscfp[i]->host) != NGX_OK)
            {
                return NGX_ERROR;
            }
#else
            ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                               "TLS healthcheck requires nginx stream SSL support");
            return NGX_ERROR;
#endif
        } else if (hc->tls_name_set || hc->trusted_certificate_set) {
            ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                               "TLS settings require type=tls in \"%V\"",
                               &uscfp[i]->host);
            return NGX_ERROR;
        }

        servers = uscfp[i]->servers->elts;
        for (j = 0; j < uscfp[i]->servers->nelts; j++) {
            for (k = 0; k < j; k++) {
                if (servers[j].backup == servers[k].backup
                    && ngx_http_upstream_healthcheck_same_string(
                           &servers[j].name, &servers[k].name))
                {
                    ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                                       "duplicate server address \"%V\" in healthchecked upstream \"%V\"",
                                       &servers[j].name, &uscfp[i]->host);
                    return NGX_ERROR;
                }
            }
        }

        zone_name.len = sizeof("nginx_stream_healthcheck_") - 1
                        + uscfp[i]->host.len;
        if (zone_name.len > NGX_MAX_PATH) {
            return NGX_ERROR;
        }
        zone_name.data = ngx_pnalloc(cf->pool, zone_name.len);
        if (zone_name.data == NULL) {
            return NGX_ERROR;
        }
        p = ngx_cpymem(zone_name.data, "nginx_stream_healthcheck_",
                       sizeof("nginx_stream_healthcheck_") - 1);
        ngx_memcpy(p, uscfp[i]->host.data, uscfp[i]->host.len);
        hc->zone_name = zone_name;

        hc->shm_zone = ngx_shared_memory_add(
                           cf, &hc->zone_name, hc->shm_size,
                           &ngx_stream_upstream_healthcheck_module);
        if (hc->shm_zone == NULL) {
            return NGX_ERROR;
        }
        hc->shm_zone->init = ngx_stream_upstream_healthcheck_init_zone;
        hc->shm_zone->data = hc;
        hc->shm_zone->noreuse = 1;

        slot = ngx_array_push(&mcf->upstreams);
        if (slot == NULL) {
            return NGX_ERROR;
        }
        *slot = hc;
    }

    return NGX_OK;
}

static ngx_int_t
ngx_stream_upstream_healthcheck_init_zone(ngx_shm_zone_t *shm_zone, void *data)
{
    ngx_stream_upstream_healthcheck_conf_t   *conf;
    ngx_stream_upstream_healthcheck_state_t  *state;
    ngx_slab_pool_t                        *shpool;

    conf = shm_zone->data;
    shpool = (ngx_slab_pool_t *) shm_zone->shm.addr;
    conf->peers = conf->uscf->peer.data;

    if (conf->peers == NULL || conf->peers->shpool == NULL
        || conf->peers->shpool
           != (ngx_slab_pool_t *) conf->uscf->shm_zone->shm.addr)
    {
        ngx_log_error(NGX_LOG_EMERG, shm_zone->shm.log, 0,
                      "healthcheck_tcp requires a built-in round-robin-compatible balancer and shared upstream zone in \"%V\"",
                      &conf->uscf->host);
        return NGX_ERROR;
    }

    if (shm_zone->shm.exists) {
        conf->state = shpool->data;
        return NGX_OK;
    }

    state = ngx_slab_calloc(shpool,
                             sizeof(ngx_stream_upstream_healthcheck_state_t));
    if (state == NULL) {
        ngx_log_error(NGX_LOG_EMERG, shm_zone->shm.log, 0,
                      "cannot allocate stream healthcheck state in zone \"%V\"",
                      &shm_zone->shm.name);
        return NGX_ERROR;
    }
    state->shpool = shpool;
    shpool->data = state;
    conf->state = state;

    return ngx_stream_upstream_healthcheck_prepare_zone(conf);
}

static ngx_int_t
ngx_stream_upstream_healthcheck_prepare_zone(
    ngx_stream_upstream_healthcheck_conf_t *conf)
{
    ngx_stream_upstream_rr_peers_t          *group;
    ngx_stream_upstream_rr_peer_t           *peer;
    ngx_stream_upstream_healthcheck_state_t   *state;
    ngx_stream_upstream_healthcheck_template_t **templatep, *template;
    ngx_uint_t                               backup;

    state = conf->state;
    group = conf->peers;
    backup = 0;
    conf->templates = NULL;

    do {
        ngx_stream_upstream_rr_peers_rlock(group);
        for (peer = group->resolve; peer != NULL; peer = peer->next) {
            template = ngx_pcalloc(conf->config_pool,
                                   sizeof(ngx_stream_upstream_healthcheck_template_t));
            if (template == NULL) {
                ngx_stream_upstream_rr_peers_unlock(group);
                return NGX_ERROR;
            }
            template->peer = peer;
            template->admin_down = ngx_stream_upstream_healthcheck_admin_down(
                                      conf, peer, backup);
            templatep = &conf->templates;
            while (*templatep != NULL) {
                templatep = &(*templatep)->next;
            }
            *templatep = template;
        }
        ngx_stream_upstream_rr_peers_unlock(group);
        group = group->next;
        backup = 1;
    } while (group != NULL);

    group = conf->peers;
    backup = 0;
    do {
        ngx_stream_upstream_rr_peers_wlock(group);
        ngx_shmtx_lock(&state->shpool->mutex);
        for (peer = group->peer; peer != NULL; peer = peer->next) {
            if (ngx_stream_upstream_healthcheck_add_peer_locked(
                    conf, group, peer, backup) == NULL)
            {
                state->errors_total++;
                state->memory_failures_total++;
                if (!ngx_stream_upstream_healthcheck_admin_down(conf, peer,
                                                               backup))
                {
                    ngx_stream_upstream_healthcheck_set_down(group, peer);
                }
                ngx_stream_upstream_healthcheck_log_memory_failure(conf, state);
            } else if (!ngx_stream_upstream_healthcheck_admin_down(conf, peer,
                                                                  backup))
            {
                ngx_stream_upstream_healthcheck_set_down(group, peer);
            }
        }
        for (peer = group->resolve; peer != NULL; peer = peer->next) {
            if (!ngx_stream_upstream_healthcheck_admin_down(conf, peer, backup)) {
                peer->down = 1;
            }
        }
        ngx_shmtx_unlock(&state->shpool->mutex);
        ngx_stream_upstream_rr_peers_unlock(group);
        group = group->next;
        backup = 1;
    } while (group != NULL);

    return NGX_OK;
}

static ngx_int_t
ngx_stream_upstream_healthcheck_init_process(ngx_cycle_t *cycle)
{
    ngx_stream_upstream_healthcheck_main_conf_t  *mcf;
    ngx_stream_upstream_healthcheck_conf_t      **hcs, *hc;
    ngx_stream_upstream_healthcheck_state_t      *state;
    ngx_stream_upstream_healthcheck_peer_t       *peer_state;
    ngx_uint_t                                  i;

    if (ngx_process != NGX_PROCESS_WORKER && ngx_process != NGX_PROCESS_SINGLE) {
        return NGX_OK;
    }
    if (ngx_worker != 0) {
        return NGX_OK;
    }

    mcf = ngx_stream_cycle_get_module_main_conf(
              cycle, ngx_stream_upstream_healthcheck_module);
    if (mcf == NULL) {
        return NGX_OK;
    }

    hcs = mcf->upstreams.elts;
    for (i = 0; i < mcf->upstreams.nelts; i++) {
        hc = hcs[i];
        state = hc->state;
        if (state == NULL) {
            ngx_log_error(NGX_LOG_ALERT, cycle->log, 0,
                          "stream healthcheck state is unavailable for upstream \"%V\"",
                          &hc->uscf->host);
            return NGX_ERROR;
        }
        ngx_shmtx_lock(&state->shpool->mutex);
        if (state->worker_pid != ngx_pid) {
            state->active_probes = 0;
            for (peer_state = state->peers; peer_state != NULL;
                 peer_state = peer_state->next)
            {
                if (peer_state->busy) {
                    peer_state->busy = 0;
                    peer_state->next_due = 0;
                }
            }
            state->worker_pid = ngx_pid;
        }
        ngx_shmtx_unlock(&state->shpool->mutex);

        hc->stopping = 0;
        ngx_memzero(&hc->timer, sizeof(ngx_event_t));
        hc->timer.data = hc;
        hc->timer.handler = ngx_stream_upstream_healthcheck_timer;
        hc->timer.log = cycle->log;
        hc->timer.cancelable = 1;
        ngx_stream_upstream_healthcheck_timer(&hc->timer);
    }

    return NGX_OK;
}

static void
ngx_stream_upstream_healthcheck_exit_process(ngx_cycle_t *cycle)
{
    ngx_stream_upstream_healthcheck_main_conf_t  *mcf;
    ngx_stream_upstream_healthcheck_conf_t      **hcs;
    ngx_stream_upstream_healthcheck_probe_t       *probe;
    ngx_uint_t                                  i;

    if (ngx_worker != 0) {
        return;
    }
    mcf = ngx_stream_cycle_get_module_main_conf(
              cycle, ngx_stream_upstream_healthcheck_module);
    if (mcf == NULL) {
        return;
    }
    hcs = mcf->upstreams.elts;
    for (i = 0; i < mcf->upstreams.nelts; i++) {
        hcs[i]->stopping = 1;
        if (hcs[i]->timer.timer_set) {
            ngx_del_timer(&hcs[i]->timer);
        }
        while (hcs[i]->probes != NULL) {
            probe = hcs[i]->probes;
            ngx_stream_upstream_healthcheck_probe_finish(probe, 0, 0, 0);
        }
    }
}

static void
ngx_stream_upstream_healthcheck_timer(ngx_event_t *ev)
{
    ngx_stream_upstream_healthcheck_conf_t  *conf;

    conf = ev->data;
    if (conf->stopping || ngx_exiting) {
        conf->stopping = 1;
        return;
    }
    ngx_stream_upstream_healthcheck_scan(conf);
    ngx_stream_upstream_healthcheck_schedule(conf);
    if (!conf->stopping && !ngx_exiting) {
        ngx_add_timer(&conf->timer, conf->interval);
    }
}

static void
ngx_stream_upstream_healthcheck_scan(ngx_stream_upstream_healthcheck_conf_t *conf)
{
    ngx_stream_upstream_healthcheck_state_t  *state;
    ngx_stream_upstream_rr_peers_t         *group;
    ngx_uint_t                              generation;
    ngx_stream_upstream_healthcheck_peer_t   *peer_state;

    state = conf->state;
    ngx_shmtx_lock(&state->shpool->mutex);
    generation = ++state->scan_generation;
    if (generation == 0) {
        generation = ++state->scan_generation;
        for (peer_state = state->peers; peer_state != NULL;
             peer_state = peer_state->next)
        {
            peer_state->seen_generation = 0;
        }
    }
    ngx_shmtx_unlock(&state->shpool->mutex);

    group = conf->peers;
    ngx_stream_upstream_healthcheck_scan_group(conf, group, 0, generation);
    if (group->next != NULL) {
        ngx_stream_upstream_healthcheck_scan_group(conf, group->next, 1,
                                                 generation);
    }

    ngx_shmtx_lock(&state->shpool->mutex);
    state->last_scan = ngx_time();
    state->worker_pid = ngx_pid;
    ngx_shmtx_unlock(&state->shpool->mutex);
}

static void
ngx_stream_upstream_healthcheck_scan_group(
    ngx_stream_upstream_healthcheck_conf_t *conf,
    ngx_stream_upstream_rr_peers_t *group, ngx_uint_t backup,
    ngx_uint_t generation)
{
    ngx_stream_upstream_healthcheck_state_t  *state;
    ngx_stream_upstream_healthcheck_peer_t   *peer_state, *next;
    ngx_stream_upstream_rr_peer_t          *peer;

    state = conf->state;
    ngx_stream_upstream_rr_peers_wlock(group);
    ngx_shmtx_lock(&state->shpool->mutex);
    for (peer = group->peer; peer != NULL; peer = peer->next) {
        peer_state = ngx_stream_upstream_healthcheck_find_peer(state, group,
                                                              peer);
        if (peer_state == NULL) {
            peer_state = ngx_stream_upstream_healthcheck_add_peer_locked(
                             conf, group, peer, backup);
            if (peer_state == NULL) {
                state->errors_total++;
                state->memory_failures_total++;
                if (!ngx_stream_upstream_healthcheck_admin_down(conf, peer,
                                                               backup))
                {
                    ngx_stream_upstream_healthcheck_set_down(group, peer);
                }
                ngx_stream_upstream_healthcheck_log_memory_failure(conf, state);
                continue;
            }
        }

        peer_state->seen_generation = generation;
        if (peer_state->admin_down || !peer_state->ready) {
            ngx_stream_upstream_healthcheck_set_down(group, peer);
        } else {
            ngx_stream_upstream_healthcheck_set_up(group, peer);
        }
    }

    peer_state = state->peers;
    while (peer_state != NULL) {
        next = peer_state->next;
        if (peer_state->group == group
            && peer_state->seen_generation != generation)
        {
            peer_state->removed = 1;
            if (!peer_state->busy) {
                ngx_stream_upstream_healthcheck_remove_peer_locked(state,
                                                                  peer_state);
            }
        }
        peer_state = next;
    }
    ngx_shmtx_unlock(&state->shpool->mutex);
    ngx_stream_upstream_rr_peers_unlock(group);
}

static void
ngx_stream_upstream_healthcheck_schedule(
    ngx_stream_upstream_healthcheck_conf_t *conf)
{
    ngx_stream_upstream_healthcheck_state_t  *state;
    ngx_stream_upstream_healthcheck_peer_t   *peer_state;
    ngx_stream_upstream_rr_peers_t         *group;
    ngx_stream_upstream_rr_peer_t          *peer;
    ngx_msec_t                              now;
    ngx_uint_t                              after_id, started, capped;
    time_t                                  log_time;

    state = conf->state;
    now = ngx_current_msec;
    after_id = state->schedule_cursor;
    started = 0;
    capped = 0;

    while (started < conf->concurrency) {
        ngx_shmtx_lock(&state->shpool->mutex);
        if (state->active_probes >= conf->concurrency) {
            ngx_shmtx_unlock(&state->shpool->mutex);
            capped = 1;
            break;
        }
        peer_state = ngx_stream_upstream_healthcheck_next_due(state, now,
                                                             after_id);
        ngx_shmtx_unlock(&state->shpool->mutex);
        if (peer_state == NULL) {
            break;
        }

        group = peer_state->group;
        peer = peer_state->peer;
        ngx_stream_upstream_rr_peers_wlock(group);
        ngx_shmtx_lock(&state->shpool->mutex);
        if (peer_state->removed || peer_state->busy || peer_state->admin_down
            || now < peer_state->next_due || peer->zombie)
        {
            after_id = peer_state->id;
            if (peer->zombie && !peer_state->busy) {
                peer_state->removed = 1;
                ngx_stream_upstream_healthcheck_remove_peer_locked(state,
                                                                  peer_state);
            }
            ngx_shmtx_unlock(&state->shpool->mutex);
            ngx_stream_upstream_rr_peers_unlock(group);
            continue;
        }
        if (state->active_probes >= conf->concurrency) {
            ngx_shmtx_unlock(&state->shpool->mutex);
            ngx_stream_upstream_rr_peers_unlock(group);
            capped = 1;
            break;
        }

        peer_state->busy = 1;
        peer_state->next_due = now + conf->interval;
        state->active_probes++;
        state->schedule_cursor = peer_state->id;
        after_id = peer_state->id;
        ngx_shmtx_unlock(&state->shpool->mutex);
        ngx_stream_upstream_rr_peers_unlock(group);

        started++;
        if (ngx_stream_upstream_healthcheck_start(conf, peer_state) != NGX_OK) {
            ngx_stream_upstream_healthcheck_start_failed(conf, peer_state, 1);
        }
    }

    if (started == conf->concurrency || capped) {
        ngx_shmtx_lock(&state->shpool->mutex);
        if (ngx_stream_upstream_healthcheck_next_due(state, now,
                                                    state->schedule_cursor)
            != NULL)
        {
            state->concurrency_limited_total++;
            log_time = ngx_time();
            if (state->last_limit_log == 0
                || log_time - state->last_limit_log >= 60)
            {
                state->last_limit_log = log_time;
                ngx_log_error(NGX_LOG_WARN, ngx_cycle->log, 0,
                              "stream healthcheck concurrency limit reached for upstream \"%V\" with limit %ui",
                              &conf->uscf->host, conf->concurrency);
            }
        }
        ngx_shmtx_unlock(&state->shpool->mutex);
    }
}

static ngx_stream_upstream_healthcheck_peer_t *
ngx_stream_upstream_healthcheck_next_due(
    ngx_stream_upstream_healthcheck_state_t *state, ngx_msec_t now,
    ngx_uint_t after_id)
{
    ngx_stream_upstream_healthcheck_peer_t  *peer_state, *after, *wrapped;

    after = NULL;
    wrapped = NULL;
    for (peer_state = state->peers; peer_state != NULL;
         peer_state = peer_state->next)
    {
        if (peer_state->removed || peer_state->busy || peer_state->admin_down
            || now < peer_state->next_due)
        {
            continue;
        }
        if (peer_state->id > after_id) {
            if (after == NULL || peer_state->id < after->id) {
                after = peer_state;
            }
        } else if (wrapped == NULL || peer_state->id < wrapped->id) {
            wrapped = peer_state;
        }
    }
    return after != NULL ? after : wrapped;
}

static ngx_stream_upstream_healthcheck_peer_t *
ngx_stream_upstream_healthcheck_find_peer(
    ngx_stream_upstream_healthcheck_state_t *state,
    ngx_stream_upstream_rr_peers_t *group,
    ngx_stream_upstream_rr_peer_t *peer)
{
    ngx_stream_upstream_healthcheck_peer_t  *peer_state;

    for (peer_state = state->peers; peer_state != NULL;
         peer_state = peer_state->next)
    {
        if (peer_state->group == group && peer_state->peer == peer) {
            return peer_state;
        }
    }
    return NULL;
}

static ngx_stream_upstream_healthcheck_peer_t *
ngx_stream_upstream_healthcheck_add_peer_locked(
    ngx_stream_upstream_healthcheck_conf_t *conf,
    ngx_stream_upstream_rr_peers_t *group,
    ngx_stream_upstream_rr_peer_t *peer, ngx_uint_t backup)
{
    ngx_stream_upstream_healthcheck_state_t  *state;
    ngx_stream_upstream_healthcheck_peer_t   *peer_state;

    state = conf->state;
    peer_state = ngx_slab_calloc_locked(state->shpool,
                                        sizeof(ngx_stream_upstream_healthcheck_peer_t));
    if (peer_state == NULL) {
        return NULL;
    }
    peer_state->group = group;
    peer_state->peer = peer;
    peer_state->id = ++state->next_peer_id;
    if (peer_state->id == 0) {
        peer_state->id = ++state->next_peer_id;
    }
    peer_state->backup = backup;
    peer_state->admin_down = ngx_stream_upstream_healthcheck_admin_down(
                                 conf, peer, backup);
    peer_state->last_change = ngx_time();
    ngx_stream_upstream_rr_peer_ref(group, peer);

    if (state->tail == NULL) {
        state->peers = peer_state;
    } else {
        state->tail->next = peer_state;
    }
    state->tail = peer_state;
    state->peer_count++;
    if (!peer_state->admin_down) {
        ngx_stream_upstream_healthcheck_set_down(group, peer);
    }
    return peer_state;
}

static void
ngx_stream_upstream_healthcheck_remove_peer_locked(
    ngx_stream_upstream_healthcheck_state_t *state,
    ngx_stream_upstream_healthcheck_peer_t *peer_state)
{
    ngx_stream_upstream_healthcheck_peer_t  **cursor, *tail;

    cursor = &state->peers;
    while (*cursor != NULL && *cursor != peer_state) {
        cursor = &(*cursor)->next;
    }
    if (*cursor == NULL) {
        return;
    }
    *cursor = peer_state->next;
    if (state->tail == peer_state) {
        state->tail = NULL;
        for (tail = state->peers; tail != NULL; tail = tail->next) {
            state->tail = tail;
        }
    }
    state->peer_count--;
    ngx_stream_upstream_rr_peer_unref(peer_state->group, peer_state->peer);
    ngx_slab_free_locked(state->shpool, peer_state);
}

static ngx_uint_t
ngx_stream_upstream_healthcheck_admin_down(
    ngx_stream_upstream_healthcheck_conf_t *conf,
    ngx_stream_upstream_rr_peer_t *peer, ngx_uint_t backup)
{
    ngx_stream_upstream_server_t               *servers;
    ngx_stream_upstream_healthcheck_template_t   *template;
    ngx_str_t                                  *server_name;
    ngx_uint_t                                  i;

    if (peer->host != NULL && peer->host->peer != NULL) {
        for (template = conf->templates; template != NULL;
             template = template->next)
        {
            if (template->peer == peer->host->peer) {
                return template->admin_down;
            }
        }
    }
    server_name = ngx_stream_upstream_healthcheck_server_name(peer);
    servers = conf->uscf->servers->elts;
    for (i = 0; i < conf->uscf->servers->nelts; i++) {
        if (servers[i].backup == backup
            && ngx_http_upstream_healthcheck_same_string(&servers[i].name,
                                                        server_name))
        {
            return servers[i].down != 0;
        }
    }
    return 0;
}

static ngx_str_t *
ngx_stream_upstream_healthcheck_server_name(ngx_stream_upstream_rr_peer_t *peer)
{
    if (peer->host != NULL && peer->host->peer != NULL) {
        return &peer->host->peer->server;
    }
    return &peer->server;
}

static void
ngx_stream_upstream_healthcheck_set_down(
    ngx_stream_upstream_rr_peers_t *group,
    ngx_stream_upstream_rr_peer_t *peer)
{
    if (!peer->down) {
        peer->down = 1;
        if (group->tries != 0) {
            group->tries--;
        }
    }
}

static void
ngx_stream_upstream_healthcheck_set_up(
    ngx_stream_upstream_rr_peers_t *group,
    ngx_stream_upstream_rr_peer_t *peer)
{
    if (peer->down) {
        peer->down = 0;
        group->tries++;
    }
}

static void
ngx_stream_upstream_healthcheck_log_memory_failure(
    ngx_stream_upstream_healthcheck_conf_t *conf,
    ngx_stream_upstream_healthcheck_state_t *state)
{
    time_t  now;

    now = ngx_time();
    if (state->last_memory_log == 0 || now - state->last_memory_log >= 60) {
        state->last_memory_log = now;
        ngx_log_error(NGX_LOG_ERR, ngx_cycle->log, 0,
                      "healthcheck shared memory is full for stream upstream \"%V\"; new peers remain down",
                      &conf->uscf->host);
    }
}

static ngx_int_t
ngx_stream_upstream_healthcheck_start(
    ngx_stream_upstream_healthcheck_conf_t *conf,
    ngx_stream_upstream_healthcheck_peer_t *peer_state)
{
    ngx_stream_upstream_healthcheck_probe_t  *probe;
    ngx_peer_connection_t                   pc;
    ngx_connection_t                       *c;
    ngx_pool_t                             *pool;
    ngx_int_t                               rc;

    pool = ngx_create_pool(4096, ngx_cycle->log);
    if (pool == NULL) {
        return NGX_ERROR;
    }
    probe = ngx_pcalloc(pool, sizeof(ngx_stream_upstream_healthcheck_probe_t));
    if (probe == NULL) {
        ngx_destroy_pool(pool);
        return NGX_ERROR;
    }
    probe->pool = pool;
    probe->conf = conf;
    probe->peer_state = peer_state;
    probe->started = ngx_current_msec;
    probe->stage = NGX_HTTP_UPSTREAM_HEALTHCHECK_CONNECT;
    probe->deadline.data = probe;
    probe->deadline.handler = ngx_stream_upstream_healthcheck_probe_deadline;
    probe->deadline.log = ngx_cycle->log;
    probe->deadline.cancelable = 1;

    if (conf->expect_set || conf->min_recv_set) {
        probe->response = ngx_pnalloc(pool, conf->max_response);
        if (probe->response == NULL) {
            ngx_destroy_pool(pool);
            return NGX_ERROR;
        }
    }

    ngx_memzero(&pc, sizeof(ngx_peer_connection_t));
    pc.sockaddr = peer_state->peer->sockaddr;
    pc.socklen = peer_state->peer->socklen;
    pc.name = &peer_state->peer->name;
    pc.get = ngx_event_get_peer;
    pc.log = ngx_cycle->log;
    pc.log_error = NGX_ERROR_ERR;

    ngx_stream_upstream_healthcheck_probe_link(probe);
    ngx_add_timer(&probe->deadline, conf->timeout);
    rc = ngx_event_connect_peer(&pc);
    probe->connection = pc.connection;
    if (rc != NGX_OK && rc != NGX_AGAIN) {
        if (probe->connection != NULL) {
            ngx_close_connection(probe->connection);
            probe->connection = NULL;
        }
        ngx_stream_upstream_healthcheck_probe_finish(probe, 0, 1, 0);
        return NGX_OK;
    }

    c = probe->connection;
    c->data = probe;
    c->pool = probe->pool;
    c->read->handler = ngx_stream_upstream_healthcheck_probe_event;
    c->write->handler = ngx_stream_upstream_healthcheck_probe_event;
    if (rc == NGX_OK) {
        ngx_stream_upstream_healthcheck_probe_event(c->write);
    }
    return NGX_OK;
}

static void
ngx_stream_upstream_healthcheck_probe_event(ngx_event_t *ev)
{
    ngx_connection_t                     *c;
    ngx_stream_upstream_healthcheck_probe_t *probe;

    c = ev->data;
    probe = c->data;
    if (probe == NULL || probe->done) {
        return;
    }
    if (ev->timedout || ev->error || c->error) {
        ngx_stream_upstream_healthcheck_probe_finish(probe, 0, 1, 0);
        return;
    }
    if (probe->stage == NGX_HTTP_UPSTREAM_HEALTHCHECK_CONNECT) {
        if (!ev->write) {
            return;
        }
        if (ngx_http_upstream_healthcheck_test_connect(c) != NGX_OK) {
            ngx_stream_upstream_healthcheck_probe_finish(probe, 0, 1, 0);
            return;
        }
        if (probe->conf->type == NGX_STREAM_UPSTREAM_HEALTHCHECK_TLS) {
            ngx_stream_upstream_healthcheck_probe_start_tls(probe);
        } else {
            ngx_stream_upstream_healthcheck_probe_send(probe);
        }
        return;
    }
    if (probe->stage == NGX_HTTP_UPSTREAM_HEALTHCHECK_SENDING) {
        ngx_stream_upstream_healthcheck_probe_send(probe);
        return;
    }
    if (probe->stage == NGX_HTTP_UPSTREAM_HEALTHCHECK_READING) {
        ngx_stream_upstream_healthcheck_probe_read(probe);
    }
}

static void
ngx_stream_upstream_healthcheck_probe_link(
    ngx_stream_upstream_healthcheck_probe_t *probe)
{
    probe->next = probe->conf->probes;
    probe->prev = &probe->conf->probes;
    if (probe->next != NULL) {
        probe->next->prev = &probe->next;
    }
    probe->conf->probes = probe;
}

static void
ngx_stream_upstream_healthcheck_probe_unlink(
    ngx_stream_upstream_healthcheck_probe_t *probe)
{
    if (probe->prev == NULL) {
        return;
    }
    *probe->prev = probe->next;
    if (probe->next != NULL) {
        probe->next->prev = probe->prev;
    }
    probe->prev = NULL;
    probe->next = NULL;
}

static void
ngx_stream_upstream_healthcheck_probe_deadline(ngx_event_t *ev)
{
    ngx_stream_upstream_healthcheck_probe_t  *probe;

    probe = ev->data;
    if (probe != NULL && !probe->done) {
        ngx_stream_upstream_healthcheck_probe_finish(probe, 0, 1, 0);
    }
}

static void
ngx_stream_upstream_healthcheck_probe_start_tls(
    ngx_stream_upstream_healthcheck_probe_t *probe)
{
#if (NGX_STREAM_SSL)
    ngx_connection_t  *c;
    ngx_int_t          rc;
    ngx_uint_t         force_down;

    c = probe->connection;
    probe->stage = NGX_HTTP_UPSTREAM_HEALTHCHECK_TLS;
    rc = ngx_upstream_healthcheck_tls_start(
             c, probe->pool, &probe->conf->ssl, &probe->conf->tls_name,
             ngx_stream_upstream_healthcheck_probe_tls_done, &force_down);
    if (rc == NGX_AGAIN) {
        return;
    }
    if (rc != NGX_OK) {
        ngx_stream_upstream_healthcheck_probe_finish(probe, 0, 1, force_down);
        return;
    }
    ngx_stream_upstream_healthcheck_probe_tls_done(c);
#else
    ngx_stream_upstream_healthcheck_probe_finish(probe, 0, 1, 1);
#endif
}

static void
ngx_stream_upstream_healthcheck_probe_tls_done(ngx_connection_t *c)
{
#if (NGX_STREAM_SSL)
    ngx_stream_upstream_healthcheck_probe_t  *probe;

    probe = c->data;
    if (probe == NULL || probe->done) {
        return;
    }
    if (ngx_upstream_healthcheck_tls_verified(
            c, &probe->conf->tls_name) != NGX_OK)
    {
        ngx_log_error(NGX_LOG_ERR, c->log, 0,
                      "stream healthcheck TLS certificate verification failed for \"%V\"",
                      &probe->conf->tls_name);
        ngx_stream_upstream_healthcheck_probe_finish(probe, 0, 1, 0);
        return;
    }
    c->read->handler = ngx_stream_upstream_healthcheck_probe_event;
    c->write->handler = ngx_stream_upstream_healthcheck_probe_event;
    ngx_stream_upstream_healthcheck_probe_send(probe);
#endif
}

static void
ngx_stream_upstream_healthcheck_probe_send(
    ngx_stream_upstream_healthcheck_probe_t *probe)
{
    ngx_connection_t  *c;
    ssize_t            n;

    c = probe->connection;
    probe->stage = NGX_HTTP_UPSTREAM_HEALTHCHECK_SENDING;
    while (probe->request_sent < probe->conf->send_data.len) {
        n = c->send(c, probe->conf->send_data.data + probe->request_sent,
                    probe->conf->send_data.len - probe->request_sent);
        if (n == NGX_AGAIN) {
            if (ngx_handle_read_event(c->read, 0) != NGX_OK
                || ngx_handle_write_event(c->write, 0) != NGX_OK)
            {
                ngx_stream_upstream_healthcheck_probe_finish(probe, 0, 1, 0);
            }
            return;
        }
        if (n <= 0) {
            ngx_stream_upstream_healthcheck_probe_finish(probe, 0, 1, 0);
            return;
        }
        probe->request_sent += n;
    }

    if (!probe->conf->expect_set && !probe->conf->min_recv_set) {
        ngx_stream_upstream_healthcheck_probe_finish(probe, 1, 1, 0);
        return;
    }
    probe->stage = NGX_HTTP_UPSTREAM_HEALTHCHECK_READING;
    c->read->handler = ngx_stream_upstream_healthcheck_probe_event;
    c->write->handler = ngx_stream_upstream_healthcheck_probe_event;
    ngx_stream_upstream_healthcheck_probe_read(probe);
}

static ngx_uint_t
ngx_stream_upstream_healthcheck_response_matches(
    ngx_stream_upstream_healthcheck_probe_t *probe)
{
    ngx_str_t  *expect;
    size_t      i;

    if (probe->response_len < probe->conf->min_recv) {
        return 0;
    }
    if (!probe->conf->expect_set) {
        return 1;
    }
    expect = &probe->conf->expect_data;
    if (expect->len == 0 || expect->len > probe->response_len) {
        return 0;
    }
    for (i = 0; i <= probe->response_len - expect->len; i++) {
        if (ngx_memcmp(probe->response + i, expect->data, expect->len) == 0) {
            return 1;
        }
    }
    return 0;
}

static void
ngx_stream_upstream_healthcheck_probe_read(
    ngx_stream_upstream_healthcheck_probe_t *probe)
{
    ngx_connection_t  *c;
    ssize_t            n;

    c = probe->connection;
    for (;;) {
        if (ngx_stream_upstream_healthcheck_response_matches(probe)) {
            ngx_stream_upstream_healthcheck_probe_finish(probe, 1, 1, 0);
            return;
        }
        if (probe->response_len == probe->conf->max_response) {
            ngx_stream_upstream_healthcheck_probe_finish(probe, 0, 1, 0);
            return;
        }
        n = c->recv(c, probe->response + probe->response_len,
                    probe->conf->max_response - probe->response_len);
        if (n == NGX_AGAIN) {
            if (ngx_handle_read_event(c->read, 0) != NGX_OK
                || ngx_handle_write_event(c->write, 0) != NGX_OK)
            {
                ngx_stream_upstream_healthcheck_probe_finish(probe, 0, 1, 0);
            }
            return;
        }
        if (n <= 0) {
            ngx_stream_upstream_healthcheck_probe_finish(probe, 0, 1, 0);
            return;
        }
        probe->response_len += n;
    }
}

static void
ngx_stream_upstream_healthcheck_probe_finish(
    ngx_stream_upstream_healthcheck_probe_t *probe, ngx_uint_t success,
    ngx_uint_t attempted, ngx_uint_t force_down)
{
    ngx_connection_t  *c;

    if (probe == NULL || probe->done) {
        return;
    }
    probe->done = 1;
    if (probe->deadline.timer_set) {
        ngx_del_timer(&probe->deadline);
    }
    ngx_stream_upstream_healthcheck_probe_unlink(probe);
    c = probe->connection;
    if (c != NULL) {
        probe->connection = NULL;
#if (NGX_STREAM_SSL)
        ngx_upstream_healthcheck_tls_close(c);
#endif
        ngx_close_connection(c);
    }
    ngx_stream_upstream_healthcheck_complete(
        probe->conf, probe->peer_state, success, attempted, force_down,
        ngx_current_msec - probe->started);
    ngx_destroy_pool(probe->pool);
}

static void
ngx_stream_upstream_healthcheck_start_failed(
    ngx_stream_upstream_healthcheck_conf_t *conf,
    ngx_stream_upstream_healthcheck_peer_t *peer_state, ngx_uint_t memory)
{
    ngx_stream_upstream_healthcheck_state_t  *state;
    ngx_stream_upstream_rr_peers_t         *group;
    ngx_stream_upstream_rr_peer_t          *peer;

    state = conf->state;
    group = peer_state->group;
    peer = peer_state->peer;
    ngx_stream_upstream_rr_peers_wlock(group);
    ngx_shmtx_lock(&state->shpool->mutex);
    if (peer_state->busy) {
        peer_state->busy = 0;
        if (state->active_probes != 0) {
            state->active_probes--;
        }
        peer_state->next_due = ngx_current_msec + conf->interval;
        if (peer->zombie || peer_state->removed) {
            peer_state->removed = 1;
            ngx_stream_upstream_healthcheck_remove_peer_locked(state,
                                                              peer_state);
        } else if (!peer_state->admin_down) {
            state->errors_total++;
            if (memory) {
                state->memory_failures_total++;
            }
            peer_state->rise_streak = 0;
            if (peer_state->fall_streak
                < NGX_HTTP_UPSTREAM_HEALTHCHECK_MAX_STREAK)
            {
                peer_state->fall_streak++;
            }
            if (peer_state->ready) {
                peer_state->ready = 0;
                peer_state->check_up_down_total++;
                peer_state->last_change = ngx_time();
            }
            ngx_stream_upstream_healthcheck_set_down(group, peer);
            if (memory) {
                ngx_stream_upstream_healthcheck_log_memory_failure(conf, state);
            }
        }
    }
    ngx_shmtx_unlock(&state->shpool->mutex);
    ngx_stream_upstream_rr_peers_unlock(group);
}

static void
ngx_stream_upstream_healthcheck_complete(
    ngx_stream_upstream_healthcheck_conf_t *conf,
    ngx_stream_upstream_healthcheck_peer_t *peer_state, ngx_uint_t success,
    ngx_uint_t attempted, ngx_uint_t force_down, ngx_msec_t duration)
{
    ngx_stream_upstream_healthcheck_state_t  *state;
    ngx_stream_upstream_rr_peers_t         *group;
    ngx_stream_upstream_rr_peer_t          *peer;
    time_t                                  now;

    state = conf->state;
    group = peer_state->group;
    peer = peer_state->peer;
    now = ngx_time();
    ngx_stream_upstream_rr_peers_wlock(group);
    ngx_shmtx_lock(&state->shpool->mutex);
    if (peer_state->busy) {
        peer_state->busy = 0;
        if (state->active_probes != 0) {
            state->active_probes--;
        }
        if (peer->zombie || peer_state->removed) {
            peer_state->removed = 1;
            ngx_stream_upstream_healthcheck_remove_peer_locked(state,
                                                              peer_state);
        } else {
            peer_state->next_due = ngx_current_msec + conf->interval;
            if (attempted) {
                peer_state->has_result = 1;
                peer_state->last_result = success;
                peer_state->status_code = success;
                peer_state->last_duration = duration;
                peer_state->last_check = now;
                peer_state->checks_total++;
                if (success) {
                    peer_state->fall_streak = 0;
                    if (peer_state->rise_streak
                        < NGX_HTTP_UPSTREAM_HEALTHCHECK_MAX_STREAK)
                    {
                        peer_state->rise_streak++;
                    }
                } else {
                    peer_state->check_failures_total++;
                    peer_state->rise_streak = 0;
                    if (peer_state->fall_streak
                        < NGX_HTTP_UPSTREAM_HEALTHCHECK_MAX_STREAK)
                    {
                        peer_state->fall_streak++;
                    }
                }
                if (success && !peer_state->ready
                    && peer_state->rise_streak >= conf->rise)
                {
                    peer_state->ready = 1;
                    peer_state->last_change = now;
                    ngx_stream_upstream_healthcheck_set_up(group, peer);
                } else if (!success
                           && (force_down
                               || peer_state->fall_streak >= conf->fall)
                           && peer_state->ready)
                {
                    peer_state->ready = 0;
                    peer_state->check_up_down_total++;
                    peer_state->last_change = now;
                    ngx_stream_upstream_healthcheck_set_down(group, peer);
                }
            }
            if (force_down && !peer_state->admin_down) {
                state->errors_total++;
                if (peer_state->ready) {
                    peer_state->ready = 0;
                    peer_state->check_up_down_total++;
                    peer_state->last_change = now;
                }
                ngx_stream_upstream_healthcheck_set_down(group, peer);
            }
        }
    }
    ngx_shmtx_unlock(&state->shpool->mutex);
    ngx_stream_upstream_rr_peers_unlock(group);
}

static ngx_int_t
ngx_stream_upstream_healthcheck_collect_samples(
    ngx_http_request_t *r, ngx_stream_upstream_healthcheck_conf_t *conf,
    ngx_stream_upstream_rr_peers_t *group, ngx_uint_t backup,
    ngx_http_upstream_healthcheck_sample_t **samples, ngx_uint_t *count)
{
    ngx_stream_upstream_healthcheck_state_t  *state;
    ngx_stream_upstream_healthcheck_peer_t   *peer_state;
    ngx_http_upstream_healthcheck_sample_t   *sample;
    ngx_str_t                              *server_name;
    ngx_uint_t                              n;

    state = conf->state;
    if (state == NULL) {
        return NGX_ERROR;
    }
    n = 0;
    ngx_stream_upstream_rr_peers_rlock(group);
    ngx_shmtx_lock(&state->shpool->mutex);
    for (peer_state = state->peers; peer_state != NULL;
         peer_state = peer_state->next)
    {
        if (peer_state->group == group && !peer_state->removed
            && !peer_state->peer->zombie)
        {
            n++;
        }
    }
    ngx_shmtx_unlock(&state->shpool->mutex);
    if (n == 0) {
        ngx_stream_upstream_rr_peers_unlock(group);
        *samples = NULL;
        *count = 0;
        return NGX_OK;
    }

    sample = ngx_pcalloc(r->pool,
                         n * sizeof(ngx_http_upstream_healthcheck_sample_t));
    if (sample == NULL) {
        ngx_stream_upstream_rr_peers_unlock(group);
        return NGX_ERROR;
    }
    ngx_shmtx_lock(&state->shpool->mutex);
    n = 0;
    for (peer_state = state->peers; peer_state != NULL;
         peer_state = peer_state->next)
    {
        if (peer_state->group != group || peer_state->removed
            || peer_state->peer->zombie)
        {
            continue;
        }
        server_name = ngx_stream_upstream_healthcheck_server_name(
                          peer_state->peer);
        if (ngx_http_upstream_healthcheck_copy_string(
                r->pool, &sample[n].upstream, &conf->uscf->host) != NGX_OK
            || ngx_http_upstream_healthcheck_copy_string(
                r->pool, &sample[n].server, server_name) != NGX_OK
            || ngx_http_upstream_healthcheck_copy_string(
                r->pool, &sample[n].peer, &peer_state->peer->name) != NGX_OK)
        {
            ngx_shmtx_unlock(&state->shpool->mutex);
            ngx_stream_upstream_rr_peers_unlock(group);
            return NGX_ERROR;
        }
        sample[n].backup = backup;
        sample[n].admin_down = peer_state->admin_down;
        sample[n].ready = peer_state->admin_down ? 0 : peer_state->ready;
        sample[n].last_result = peer_state->has_result
                                ? (ngx_int_t) peer_state->last_result : -1;
        sample[n].status_code = 0;
        sample[n].rise_streak = peer_state->rise_streak;
        sample[n].fall_streak = peer_state->fall_streak;
        sample[n].checks_total = peer_state->checks_total;
        sample[n].check_failures_total = peer_state->check_failures_total;
        sample[n].check_up_down_total = peer_state->check_up_down_total;
        sample[n].last_duration = peer_state->last_duration;
        sample[n].last_check = peer_state->last_check;
        sample[n].last_change = peer_state->last_change;
        n++;
    }
    ngx_shmtx_unlock(&state->shpool->mutex);
    ngx_stream_upstream_rr_peers_unlock(group);
    *samples = sample;
    *count = n;
    return NGX_OK;
}

static ngx_int_t
ngx_stream_upstream_healthcheck_render_group(
    ngx_http_upstream_healthcheck_output_t *out,
    ngx_stream_upstream_healthcheck_conf_t *conf)
{
    ngx_stream_upstream_healthcheck_state_t  *state;
    ngx_uint_t                              active, errors, limited, memory;
    ngx_pid_t                               pid;
    time_t                                  last_scan;

    state = conf->state;
    if (state == NULL) {
        return NGX_ERROR;
    }
    ngx_shmtx_lock(&state->shpool->mutex);
    active = state->active_probes;
    errors = state->errors_total;
    limited = state->concurrency_limited_total;
    memory = state->memory_failures_total;
    pid = state->worker_pid;
    last_scan = state->last_scan;
    ngx_shmtx_unlock(&state->shpool->mutex);

    if (ngx_http_upstream_healthcheck_output_group_time(
            out, "nginx_stream_healthcheck_last_scan_timestamp_seconds",
            &conf->uscf->host, last_scan) != NGX_OK
        || ngx_http_upstream_healthcheck_output_group_uint(
            out, "nginx_stream_healthcheck_errors_total",
            &conf->uscf->host, errors) != NGX_OK
        || ngx_http_upstream_healthcheck_output_group_uint(
            out, "nginx_stream_healthcheck_active_probes",
            &conf->uscf->host, active) != NGX_OK
        || ngx_http_upstream_healthcheck_output_group_uint(
            out, "nginx_stream_healthcheck_worker_pid",
            &conf->uscf->host, (ngx_uint_t) pid) != NGX_OK
        || ngx_http_upstream_healthcheck_output_group_uint(
            out, "nginx_stream_healthcheck_concurrency",
            &conf->uscf->host, conf->concurrency) != NGX_OK
        || ngx_http_upstream_healthcheck_output_group_uint(
            out, "nginx_stream_healthcheck_concurrency_limited_total",
            &conf->uscf->host, limited) != NGX_OK
        || ngx_http_upstream_healthcheck_output_group_uint(
            out, "nginx_stream_healthcheck_memory_failures_total",
            &conf->uscf->host, memory) != NGX_OK)
    {
        return NGX_ERROR;
    }
    return NGX_OK;
}

#if (NGX_HTTP_SSL || NGX_STREAM_SSL)
static ngx_int_t
ngx_upstream_healthcheck_ssl_context(ngx_conf_t *cf, ngx_ssl_t *ssl,
    ngx_str_t *trusted_certificate, ngx_str_t *upstream)
{
    ngx_pool_cleanup_t  *cln;

    ssl->log = cf->log;
    if (ngx_ssl_create(ssl, NGX_SSL_DEFAULT_PROTOCOLS, NULL) != NGX_OK) {
        return NGX_ERROR;
    }
    cln = ngx_pool_cleanup_add(cf->pool, 0);
    if (cln == NULL) {
        ngx_ssl_cleanup_ctx(ssl);
        return NGX_ERROR;
    }
    cln->handler = ngx_ssl_cleanup_ctx;
    cln->data = ssl;

#if (OPENSSL_VERSION_NUMBER >= 0x10100000L)
    if (SSL_CTX_set_min_proto_version(ssl->ctx, TLS1_2_VERSION) != 1) {
        ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                           "cannot require TLS 1.2 for healthcheck in \"%V\"",
                           upstream);
        return NGX_ERROR;
    }
#else
    ngx_conf_log_error(NGX_LOG_EMERG, cf, 0,
                       "healthcheck TLS requires OpenSSL 1.1.0 or newer");
    return NGX_ERROR;
#endif

    SSL_CTX_set_verify(ssl->ctx, SSL_VERIFY_PEER, NULL);
    if (ngx_ssl_trusted_certificate(cf, ssl, trusted_certificate, 10)
        != NGX_OK)
    {
        return NGX_ERROR;
    }
    return NGX_OK;
}

static ngx_int_t
ngx_upstream_healthcheck_tls_start(ngx_connection_t *c, ngx_pool_t *pool,
    ngx_ssl_t *ssl, ngx_str_t *tls_name,
    ngx_connection_handler_pt handler, ngx_uint_t *force_down)
{
    ngx_int_t  rc;
    u_char    *name;

    *force_down = 0;
    c->pool = pool;
    if (ssl->ctx == NULL
        || ngx_ssl_create_connection(ssl, c, NGX_SSL_CLIENT) != NGX_OK)
    {
        *force_down = 1;
        return NGX_ERROR;
    }
    name = ngx_pnalloc(pool, tls_name->len + 1);
    if (name == NULL) {
        *force_down = 1;
        return NGX_ERROR;
    }
    ngx_memcpy(name, tls_name->data, tls_name->len);
    name[tls_name->len] = '\0';
    if (SSL_set_tlsext_host_name(c->ssl->connection, (char *) name) != 1) {
        ngx_ssl_error(NGX_LOG_ERR, c->log, 0,
                      "SSL_set_tlsext_host_name() failed for active healthcheck");
        return NGX_ERROR;
    }
    c->ssl->handler = handler;
    rc = ngx_ssl_handshake(c);
    return rc;
}

static ngx_int_t
ngx_upstream_healthcheck_tls_verified(ngx_connection_t *c, ngx_str_t *tls_name)
{
    if (c->ssl == NULL || !c->ssl->handshaked
        || SSL_get_verify_result(c->ssl->connection) != X509_V_OK
        || ngx_ssl_check_host(c, tls_name) != NGX_OK)
    {
        return NGX_ERROR;
    }
    return NGX_OK;
}

static void
ngx_upstream_healthcheck_tls_close(ngx_connection_t *c)
{
    ngx_int_t  rc;

    if (c->ssl == NULL) {
        return;
    }
    c->ssl->no_wait_shutdown = 1;
    c->ssl->no_send_shutdown = 1;
    rc = ngx_ssl_shutdown(c);
    if (rc == NGX_AGAIN && c->ssl != NULL) {
        SSL_free(c->ssl->connection);
        c->ssl = NULL;
    }
}
#endif

static ngx_int_t
ngx_http_upstream_healthcheck_metrics(ngx_http_request_t *r)
{
    ngx_http_upstream_healthcheck_main_conf_t  *mcf;
    ngx_http_upstream_healthcheck_conf_t      **hcs;
    ngx_stream_upstream_healthcheck_main_conf_t *smcf;
    ngx_stream_upstream_healthcheck_conf_t     **shcs;
    ngx_http_upstream_healthcheck_sample_t     *group_samples, *sample;
    ngx_http_upstream_healthcheck_output_t      out;
    ngx_array_t                               samples, stream_samples;
    ngx_buf_t                                *buffer;
    ngx_chain_t                               chain;
    ngx_uint_t                                i, j, count;
    ngx_uint_t                                family_index;
    ngx_int_t                                 rc;

    if (!(r->method & (NGX_HTTP_GET|NGX_HTTP_HEAD))) {
        return NGX_HTTP_NOT_ALLOWED;
    }

    rc = ngx_http_discard_request_body(r);
    if (rc != NGX_OK) {
        return rc;
    }

    mcf = ngx_http_get_module_main_conf(r,
                                        ngx_http_upstream_healthcheck_module);
    if (mcf == NULL
        || ngx_array_init(&samples, r->pool, 16,
                          sizeof(ngx_http_upstream_healthcheck_sample_t))
           != NGX_OK)
    {
        return NGX_HTTP_INTERNAL_SERVER_ERROR;
    }

    hcs = mcf->upstreams.elts;
    for (i = 0; i < mcf->upstreams.nelts; i++) {
        for (j = 0; j < 2; j++) {
            ngx_http_upstream_rr_peers_t *group;

            group = j == 0 ? hcs[i]->peers : hcs[i]->peers->next;
            if (group == NULL) {
                continue;
            }

            if (ngx_http_upstream_healthcheck_collect_samples(
                    r, hcs[i], group, j, &group_samples, &count)
                != NGX_OK)
            {
                return NGX_HTTP_INTERNAL_SERVER_ERROR;
            }

            if (count == 0) {
                continue;
            }

            sample = ngx_array_push_n(&samples, count);
            if (sample == NULL) {
                return NGX_HTTP_INTERNAL_SERVER_ERROR;
            }
            ngx_memcpy(sample, group_samples,
                       count * sizeof(ngx_http_upstream_healthcheck_sample_t));
        }
    }

    if (ngx_array_init(&stream_samples, r->pool, 16,
                       sizeof(ngx_http_upstream_healthcheck_sample_t))
        != NGX_OK)
    {
        return NGX_HTTP_INTERNAL_SERVER_ERROR;
    }
    smcf = ngx_stream_cycle_get_module_main_conf(
               ngx_cycle, ngx_stream_upstream_healthcheck_module);
    if (smcf != NULL) {
        shcs = smcf->upstreams.elts;
        for (i = 0; i < smcf->upstreams.nelts; i++) {
            for (j = 0; j < 2; j++) {
                ngx_stream_upstream_rr_peers_t *group;

                group = j == 0 ? shcs[i]->peers : shcs[i]->peers->next;
                if (group == NULL) {
                    continue;
                }
                if (ngx_stream_upstream_healthcheck_collect_samples(
                        r, shcs[i], group, j, &group_samples, &count)
                    != NGX_OK)
                {
                    return NGX_HTTP_INTERNAL_SERVER_ERROR;
                }
                if (count == 0) {
                    continue;
                }
                sample = ngx_array_push_n(&stream_samples, count);
                if (sample == NULL) {
                    return NGX_HTTP_INTERNAL_SERVER_ERROR;
                }
                ngx_memcpy(sample, group_samples,
                           count * sizeof(ngx_http_upstream_healthcheck_sample_t));
            }
        }
    }

    ngx_memzero(&out, sizeof(ngx_http_upstream_healthcheck_output_t));
    out.pool = r->pool;

    if (ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_healthcheck_peer_up",
            "Whether the active checker currently enables the peer.",
            "gauge") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_healthcheck_admin_down",
            "Whether the peer was configured with the down parameter.",
            "gauge") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_healthcheck_checks_total",
            "Completed active health checks.", "counter") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_healthcheck_check_failures_total",
            "Completed active health checks that failed.",
            "counter") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_healthcheck_check_up_down_total",
            "Failed checks and resource failures that changed a ready peer to down.",
            "counter") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_healthcheck_check_status",
            "Latest check result: -1 before a result, 0 failure, or 1 success.",
            "gauge") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_healthcheck_check_code",
            "Latest parsed HTTP status code, or 0 when not available.",
            "gauge") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_healthcheck_check_duration_seconds",
            "Duration of the latest completed active check.",
            "gauge") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_healthcheck_check_last_change_seconds",
            "Seconds since the peer health state last changed or was initialized.",
            "gauge") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_healthcheck_last_check_timestamp_seconds",
            "Unix timestamp of the latest completed check, or 0 before a result.",
            "gauge") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_healthcheck_rise_streak",
            "Consecutive successful checks for the peer.", "gauge") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_healthcheck_fall_streak",
            "Consecutive failed checks for the peer.", "gauge") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_healthcheck_last_scan_timestamp_seconds",
            "Unix timestamp of the latest peer-list scan.", "gauge")
           != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_healthcheck_errors_total",
            "Checker resource and state errors for the upstream.",
            "counter") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_healthcheck_active_probes",
            "Active asynchronous probes for the upstream.", "gauge")
           != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_healthcheck_worker_pid",
            "Operating-system process ID of the worker that owns the checker.",
            "gauge") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_healthcheck_concurrency",
            "Maximum simultaneous probes configured for the upstream.",
            "gauge") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_healthcheck_concurrency_limited_total",
            "Scheduler scans that reached the configured probe concurrency.",
            "counter") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_healthcheck_memory_failures_total",
            "Checker state or probe memory allocation failures.",
            "counter") != NGX_OK)
    {
        return NGX_HTTP_INTERNAL_SERVER_ERROR;
    }

    if (ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_stream_healthcheck_peer_up",
            "Whether the stream active checker currently enables the peer.",
            "gauge") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_stream_healthcheck_admin_down",
            "Whether the stream peer was configured with the down parameter.",
            "gauge") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_stream_healthcheck_checks_total",
            "Completed stream active health checks.", "counter") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_stream_healthcheck_check_failures_total",
            "Completed stream active health checks that failed.",
            "counter") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_stream_healthcheck_check_up_down_total",
            "Failed checks and resource failures that changed a ready peer to down.",
            "counter") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_stream_healthcheck_check_status",
            "Latest stream check result: -1 before a result, 0 failure, or 1 success.",
            "gauge") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_stream_healthcheck_check_duration_seconds",
            "Duration of the latest completed stream active check.",
            "gauge") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_stream_healthcheck_check_last_change_seconds",
            "Seconds since the stream peer health state last changed or was initialized.",
            "gauge") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_stream_healthcheck_last_check_timestamp_seconds",
            "Unix timestamp of the latest completed stream check, or 0 before a result.",
            "gauge") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_stream_healthcheck_rise_streak",
            "Consecutive successful stream checks for the peer.", "gauge")
           != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_stream_healthcheck_fall_streak",
            "Consecutive failed stream checks for the peer.", "gauge")
           != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_stream_healthcheck_last_scan_timestamp_seconds",
            "Unix timestamp of the latest stream peer-list scan.", "gauge")
           != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_stream_healthcheck_errors_total",
            "Stream checker resource and state errors for the upstream.",
            "counter") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_stream_healthcheck_active_probes",
            "Active asynchronous stream probes for the upstream.", "gauge")
           != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_stream_healthcheck_worker_pid",
            "Operating-system process ID of the stream checker worker.",
            "gauge") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_stream_healthcheck_concurrency",
            "Maximum simultaneous stream probes configured for the upstream.",
            "gauge") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_stream_healthcheck_concurrency_limited_total",
            "Stream scheduler scans that reached the configured probe concurrency.",
            "counter") != NGX_OK
        || ngx_http_upstream_healthcheck_output_help(
            &out, "nginx_stream_healthcheck_memory_failures_total",
            "Stream checker state or probe memory allocation failures.",
            "counter") != NGX_OK)
    {
        return NGX_HTTP_INTERNAL_SERVER_ERROR;
    }

    count = samples.nelts;
    sample = samples.elts;
    for (i = 0; i < 12; i++) {
        static const char *families[] = {
            "nginx_healthcheck_peer_up",
            "nginx_healthcheck_admin_down",
            "nginx_healthcheck_checks_total",
            "nginx_healthcheck_check_failures_total",
            "nginx_healthcheck_check_up_down_total",
            "nginx_healthcheck_check_status",
            "nginx_healthcheck_check_code",
            "nginx_healthcheck_check_duration_seconds",
            "nginx_healthcheck_check_last_change_seconds",
            "nginx_healthcheck_last_check_timestamp_seconds",
            "nginx_healthcheck_rise_streak",
            "nginx_healthcheck_fall_streak"
        };

        if (ngx_http_upstream_healthcheck_render_peer_family(
                &out, families[i], (ngx_array_t *) &samples, i, ngx_time())
            != NGX_OK)
        {
            return NGX_HTTP_INTERNAL_SERVER_ERROR;
        }
    }

    hcs = mcf->upstreams.elts;
    for (i = 0; i < mcf->upstreams.nelts; i++) {
        if (ngx_http_upstream_healthcheck_render_group(&out, hcs[i]) != NGX_OK) {
            return NGX_HTTP_INTERNAL_SERVER_ERROR;
        }
    }

    for (i = 0; i < 11; i++) {
        static const char *families[] = {
            "nginx_stream_healthcheck_peer_up",
            "nginx_stream_healthcheck_admin_down",
            "nginx_stream_healthcheck_checks_total",
            "nginx_stream_healthcheck_check_failures_total",
            "nginx_stream_healthcheck_check_up_down_total",
            "nginx_stream_healthcheck_check_status",
            "nginx_stream_healthcheck_check_duration_seconds",
            "nginx_stream_healthcheck_check_last_change_seconds",
            "nginx_stream_healthcheck_last_check_timestamp_seconds",
            "nginx_stream_healthcheck_rise_streak",
            "nginx_stream_healthcheck_fall_streak"
        };
        static const ngx_uint_t indexes[] = { 0, 1, 2, 3, 4, 5, 7, 8, 9, 10, 11 };

        family_index = indexes[i];
        if (ngx_http_upstream_healthcheck_render_peer_family(
                &out, families[i], &stream_samples, family_index, ngx_time())
            != NGX_OK)
        {
            return NGX_HTTP_INTERNAL_SERVER_ERROR;
        }
    }
    if (smcf != NULL) {
        shcs = smcf->upstreams.elts;
        for (i = 0; i < smcf->upstreams.nelts; i++) {
            if (ngx_stream_upstream_healthcheck_render_group(&out, shcs[i])
                != NGX_OK)
            {
                return NGX_HTTP_INTERNAL_SERVER_ERROR;
            }
        }
    }

    r->headers_out.status = NGX_HTTP_OK;
    r->headers_out.content_length_n = out.len;
    r->headers_out.content_type_len = sizeof("text/plain; version=0.0.4; charset=utf-8") - 1;
    ngx_str_set(&r->headers_out.content_type,
                "text/plain; version=0.0.4; charset=utf-8");
    r->headers_out.content_type_lowcase = NULL;

    buffer = ngx_calloc_buf(r->pool);
    if (buffer == NULL) {
        return NGX_HTTP_INTERNAL_SERVER_ERROR;
    }

    buffer->pos = out.data;
    buffer->last = out.data + out.len;
    buffer->memory = 1;
    buffer->last_buf = (r == r->main);
    buffer->last_in_chain = 1;
    chain.buf = buffer;
    chain.next = NULL;

    rc = ngx_http_send_header(r);
    if (rc == NGX_ERROR || rc > NGX_OK || r->header_only) {
        return rc;
    }

    return ngx_http_output_filter(r, &chain);
}

static ngx_int_t
ngx_http_upstream_healthcheck_collect_samples(
    ngx_http_request_t *r, ngx_http_upstream_healthcheck_conf_t *conf,
    ngx_http_upstream_rr_peers_t *group, ngx_uint_t backup,
    ngx_http_upstream_healthcheck_sample_t **samples, ngx_uint_t *count)
{
    ngx_http_upstream_healthcheck_state_t  *state;
    ngx_http_upstream_healthcheck_peer_t   *peer_state;
    ngx_http_upstream_healthcheck_sample_t *sample;
    ngx_str_t                            *server_name;
    ngx_uint_t                            n;

    state = conf->state;
    n = 0;
    ngx_http_upstream_rr_peers_rlock(group);
    ngx_shmtx_lock(&state->shpool->mutex);

    for (peer_state = state->peers; peer_state != NULL;
         peer_state = peer_state->next)
    {
        if (peer_state->group == group && !peer_state->removed
            && !peer_state->peer->zombie)
        {
            n++;
        }
    }

    ngx_shmtx_unlock(&state->shpool->mutex);

    if (n == 0) {
        ngx_http_upstream_rr_peers_unlock(group);
        *samples = NULL;
        *count = 0;
        return NGX_OK;
    }

    sample = ngx_pcalloc(r->pool,
                         n * sizeof(ngx_http_upstream_healthcheck_sample_t));
    if (sample == NULL) {
        ngx_http_upstream_rr_peers_unlock(group);
        return NGX_ERROR;
    }

    ngx_shmtx_lock(&state->shpool->mutex);
    n = 0;
    for (peer_state = state->peers; peer_state != NULL;
         peer_state = peer_state->next)
    {
        if (peer_state->group != group || peer_state->removed
            || peer_state->peer->zombie)
        {
            continue;
        }

        server_name = ngx_http_upstream_healthcheck_server_name(peer_state->peer);
        if (ngx_http_upstream_healthcheck_copy_string(
                r->pool, &sample[n].upstream, &conf->uscf->host) != NGX_OK
            || ngx_http_upstream_healthcheck_copy_string(
                r->pool, &sample[n].server, server_name) != NGX_OK
            || ngx_http_upstream_healthcheck_copy_string(
                r->pool, &sample[n].peer, &peer_state->peer->name) != NGX_OK)
        {
            ngx_shmtx_unlock(&state->shpool->mutex);
            ngx_http_upstream_rr_peers_unlock(group);
            return NGX_ERROR;
        }

        sample[n].backup = backup;
        sample[n].admin_down = peer_state->admin_down;
        sample[n].ready = peer_state->admin_down ? 0 : peer_state->ready;
        sample[n].last_result = peer_state->has_result
                                ? (ngx_int_t) peer_state->last_result : -1;
        sample[n].status_code = peer_state->status_code;
        sample[n].rise_streak = peer_state->rise_streak;
        sample[n].fall_streak = peer_state->fall_streak;
        sample[n].checks_total = peer_state->checks_total;
        sample[n].check_failures_total = peer_state->check_failures_total;
        sample[n].check_up_down_total = peer_state->check_up_down_total;
        sample[n].last_duration = peer_state->last_duration;
        sample[n].last_check = peer_state->last_check;
        sample[n].last_change = peer_state->last_change;
        n++;
    }

    ngx_shmtx_unlock(&state->shpool->mutex);
    ngx_http_upstream_rr_peers_unlock(group);
    *samples = sample;
    *count = n;
    return NGX_OK;
}

static ngx_int_t
ngx_http_upstream_healthcheck_render_peer_family(
    ngx_http_upstream_healthcheck_output_t *out, const char *name,
    ngx_array_t *samples, ngx_uint_t family, time_t now)
{
    ngx_http_upstream_healthcheck_sample_t  *sample;
    ngx_uint_t                             i;

    sample = samples->elts;
    for (i = 0; i < samples->nelts; i++) {
        switch (family) {
        case 0:
            if (ngx_http_upstream_healthcheck_output_sample_uint(
                    out, name, &sample[i], sample[i].ready) != NGX_OK)
            {
                return NGX_ERROR;
            }
            break;

        case 1:
            if (ngx_http_upstream_healthcheck_output_sample_uint(
                    out, name, &sample[i], sample[i].admin_down) != NGX_OK)
            {
                return NGX_ERROR;
            }
            break;

        case 2:
            if (ngx_http_upstream_healthcheck_output_sample_uint(
                    out, name, &sample[i], sample[i].checks_total) != NGX_OK)
            {
                return NGX_ERROR;
            }
            break;

        case 3:
            if (ngx_http_upstream_healthcheck_output_sample_uint(
                    out, name, &sample[i],
                    sample[i].check_failures_total) != NGX_OK)
            {
                return NGX_ERROR;
            }
            break;

        case 4:
            if (ngx_http_upstream_healthcheck_output_sample_uint(
                    out, name, &sample[i],
                    sample[i].check_up_down_total) != NGX_OK)
            {
                return NGX_ERROR;
            }
            break;

        case 5:
            if (ngx_http_upstream_healthcheck_output_sample_int(
                    out, name, &sample[i], sample[i].last_result) != NGX_OK)
            {
                return NGX_ERROR;
            }
            break;

        case 6:
            if (ngx_http_upstream_healthcheck_output_sample_uint(
                    out, name, &sample[i], sample[i].status_code) != NGX_OK)
            {
                return NGX_ERROR;
            }
            break;

        case 7:
            if (ngx_http_upstream_healthcheck_output_text(out, name) != NGX_OK
                || ngx_http_upstream_healthcheck_output_labels(
                       out, &sample[i]) != NGX_OK
                || ngx_http_upstream_healthcheck_output_text(out, " ")
                   != NGX_OK
                || ngx_http_upstream_healthcheck_output_seconds(
                       out, sample[i].last_duration) != NGX_OK
                || ngx_http_upstream_healthcheck_output_text(out, "\n")
                   != NGX_OK)
            {
                return NGX_ERROR;
            }
            break;

        case 8:
            if (ngx_http_upstream_healthcheck_output_sample_uint(
                    out, name, &sample[i],
                    now >= sample[i].last_change
                    ? now - sample[i].last_change : 0) != NGX_OK)
            {
                return NGX_ERROR;
            }
            break;

        case 9:
            if (ngx_http_upstream_healthcheck_output_sample_uint(
                    out, name, &sample[i], sample[i].last_check) != NGX_OK)
            {
                return NGX_ERROR;
            }
            break;

        case 10:
            if (ngx_http_upstream_healthcheck_output_sample_uint(
                    out, name, &sample[i], sample[i].rise_streak) != NGX_OK)
            {
                return NGX_ERROR;
            }
            break;

        case 11:
            if (ngx_http_upstream_healthcheck_output_sample_uint(
                    out, name, &sample[i], sample[i].fall_streak) != NGX_OK)
            {
                return NGX_ERROR;
            }
            break;

        default:
            return NGX_ERROR;
        }

    }

    return NGX_OK;
}

static ngx_int_t
ngx_http_upstream_healthcheck_render_group(
    ngx_http_upstream_healthcheck_output_t *out,
    ngx_http_upstream_healthcheck_conf_t *conf)
{
    ngx_http_upstream_healthcheck_state_t  *state;
    ngx_uint_t                            active, errors, limited, memory;
    ngx_pid_t                              pid;
    time_t                                last_scan;

    state = conf->state;
    ngx_shmtx_lock(&state->shpool->mutex);
    active = state->active_probes;
    errors = state->errors_total;
    limited = state->concurrency_limited_total;
    memory = state->memory_failures_total;
    pid = state->worker_pid;
    last_scan = state->last_scan;
    ngx_shmtx_unlock(&state->shpool->mutex);

    if (ngx_http_upstream_healthcheck_output_group_time(
            out, "nginx_healthcheck_last_scan_timestamp_seconds",
            &conf->uscf->host, last_scan) != NGX_OK
        || ngx_http_upstream_healthcheck_output_group_uint(
            out, "nginx_healthcheck_errors_total", &conf->uscf->host,
            errors) != NGX_OK
        || ngx_http_upstream_healthcheck_output_group_uint(
            out, "nginx_healthcheck_active_probes", &conf->uscf->host,
            active) != NGX_OK
        || ngx_http_upstream_healthcheck_output_group_uint(
            out, "nginx_healthcheck_worker_pid", &conf->uscf->host,
            (ngx_uint_t) pid) != NGX_OK
        || ngx_http_upstream_healthcheck_output_group_uint(
            out, "nginx_healthcheck_concurrency", &conf->uscf->host,
            conf->concurrency) != NGX_OK
        || ngx_http_upstream_healthcheck_output_group_uint(
            out, "nginx_healthcheck_concurrency_limited_total",
            &conf->uscf->host, limited) != NGX_OK
        || ngx_http_upstream_healthcheck_output_group_uint(
            out, "nginx_healthcheck_memory_failures_total",
            &conf->uscf->host, memory) != NGX_OK)
    {
        return NGX_ERROR;
    }

    return NGX_OK;
}

static ngx_int_t
ngx_http_upstream_healthcheck_output_reserve(
    ngx_http_upstream_healthcheck_output_t *out, size_t extra)
{
    size_t   capacity, needed;
    u_char  *data;

    if (extra > (size_t) -1 - out->len) {
        return NGX_ERROR;
    }

    needed = out->len + extra;
    if (needed <= out->capacity) {
        return NGX_OK;
    }

    capacity = out->capacity == 0 ? 1024 : out->capacity;
    while (capacity < needed) {
        if (capacity > (size_t) -1 / 2) {
            capacity = needed;
            break;
        }
        capacity *= 2;
    }

    data = ngx_pnalloc(out->pool, capacity);
    if (data == NULL) {
        return NGX_ERROR;
    }

    if (out->len != 0) {
        ngx_memcpy(data, out->data, out->len);
    }

    out->data = data;
    out->capacity = capacity;
    return NGX_OK;
}

static ngx_int_t
ngx_http_upstream_healthcheck_output_append(
    ngx_http_upstream_healthcheck_output_t *out, const u_char *data,
    size_t len)
{
    if (ngx_http_upstream_healthcheck_output_reserve(out, len) != NGX_OK) {
        return NGX_ERROR;
    }

    if (len != 0) {
        ngx_memcpy(out->data + out->len, data, len);
        out->len += len;
    }

    return NGX_OK;
}

static ngx_int_t
ngx_http_upstream_healthcheck_output_text(
    ngx_http_upstream_healthcheck_output_t *out, const char *text)
{
    return ngx_http_upstream_healthcheck_output_append(
               out, (const u_char *) text, ngx_strlen((u_char *) text));
}

static ngx_int_t
ngx_http_upstream_healthcheck_output_uint(
    ngx_http_upstream_healthcheck_output_t *out, ngx_uint_t value)
{
    u_char   buffer[NGX_INT_T_LEN + 1];
    u_char  *last;

    last = ngx_sprintf(buffer, "%ui", value);
    return ngx_http_upstream_healthcheck_output_append(out, buffer,
                                                      last - buffer);
}

static ngx_int_t
ngx_http_upstream_healthcheck_output_int(
    ngx_http_upstream_healthcheck_output_t *out, ngx_int_t value)
{
    u_char   buffer[NGX_INT_T_LEN + 2];
    u_char  *last;

    last = ngx_sprintf(buffer, "%i", value);
    return ngx_http_upstream_healthcheck_output_append(out, buffer,
                                                      last - buffer);
}

static ngx_int_t
ngx_http_upstream_healthcheck_output_seconds(
    ngx_http_upstream_healthcheck_output_t *out, ngx_msec_t value)
{
    u_char   buffer[NGX_INT_T_LEN + 5];
    u_char  *last;

    last = ngx_sprintf(buffer, "%ui.%03ui", value / 1000, value % 1000);
    return ngx_http_upstream_healthcheck_output_append(out, buffer,
                                                      last - buffer);
}

static ngx_int_t
ngx_http_upstream_healthcheck_output_label(
    ngx_http_upstream_healthcheck_output_t *out, const ngx_str_t *value)
{
    size_t   i;
    u_char   ch;

    if (ngx_http_upstream_healthcheck_output_text(out, "\"") != NGX_OK) {
        return NGX_ERROR;
    }

    for (i = 0; i < value->len; i++) {
        ch = value->data[i];
        if (ch == '\\' || ch == '"') {
            if (ngx_http_upstream_healthcheck_output_append(out, (u_char *) "\\", 1)
                != NGX_OK
                || ngx_http_upstream_healthcheck_output_append(out, &ch, 1)
                   != NGX_OK)
            {
                return NGX_ERROR;
            }
        } else if (ch == '\n') {
            if (ngx_http_upstream_healthcheck_output_text(out, "\\n") != NGX_OK) {
                return NGX_ERROR;
            }
        } else if (ngx_http_upstream_healthcheck_output_append(out, &ch, 1)
                   != NGX_OK)
        {
            return NGX_ERROR;
        }
    }

    return ngx_http_upstream_healthcheck_output_text(out, "\"");
}

static ngx_int_t
ngx_http_upstream_healthcheck_output_labels(
    ngx_http_upstream_healthcheck_output_t *out,
    const ngx_http_upstream_healthcheck_sample_t *sample)
{
    if (ngx_http_upstream_healthcheck_output_text(out, "{upstream=") != NGX_OK
        || ngx_http_upstream_healthcheck_output_label(out, &sample->upstream)
           != NGX_OK
        || ngx_http_upstream_healthcheck_output_text(out, ",server=") != NGX_OK
        || ngx_http_upstream_healthcheck_output_label(out, &sample->server)
           != NGX_OK
        || ngx_http_upstream_healthcheck_output_text(out, ",peer=") != NGX_OK
        || ngx_http_upstream_healthcheck_output_label(out, &sample->peer)
           != NGX_OK
        || ngx_http_upstream_healthcheck_output_text(out, ",backup=\"")
           != NGX_OK
        || ngx_http_upstream_healthcheck_output_text(
               out, sample->backup ? "true" : "false") != NGX_OK
        || ngx_http_upstream_healthcheck_output_text(out, "\"}") != NGX_OK)
    {
        return NGX_ERROR;
    }

    return NGX_OK;
}

static ngx_int_t
ngx_http_upstream_healthcheck_output_group_label(
    ngx_http_upstream_healthcheck_output_t *out, const ngx_str_t *upstream)
{
    if (ngx_http_upstream_healthcheck_output_text(out, "{upstream=") != NGX_OK
        || ngx_http_upstream_healthcheck_output_label(out, upstream) != NGX_OK
        || ngx_http_upstream_healthcheck_output_text(out, "}") != NGX_OK)
    {
        return NGX_ERROR;
    }

    return NGX_OK;
}

static ngx_int_t
ngx_http_upstream_healthcheck_output_help(
    ngx_http_upstream_healthcheck_output_t *out, const char *name,
    const char *help, const char *type)
{
    return ngx_http_upstream_healthcheck_output_text(out, "# HELP ")
           == NGX_OK
           && ngx_http_upstream_healthcheck_output_text(out, name) == NGX_OK
           && ngx_http_upstream_healthcheck_output_text(out, " ") == NGX_OK
           && ngx_http_upstream_healthcheck_output_text(out, help) == NGX_OK
           && ngx_http_upstream_healthcheck_output_text(out, "\n# TYPE ")
              == NGX_OK
           && ngx_http_upstream_healthcheck_output_text(out, name) == NGX_OK
           && ngx_http_upstream_healthcheck_output_text(out, " ") == NGX_OK
           && ngx_http_upstream_healthcheck_output_text(out, type) == NGX_OK
           && ngx_http_upstream_healthcheck_output_text(out, "\n") == NGX_OK
           ? NGX_OK : NGX_ERROR;
}

static ngx_int_t
ngx_http_upstream_healthcheck_output_sample(
    ngx_http_upstream_healthcheck_output_t *out, const char *name,
    const ngx_http_upstream_healthcheck_sample_t *sample, const char *value)
{
    if (ngx_http_upstream_healthcheck_output_text(out, name) != NGX_OK
        || ngx_http_upstream_healthcheck_output_labels(out, sample) != NGX_OK
        || ngx_http_upstream_healthcheck_output_text(out, " ") != NGX_OK
        || ngx_http_upstream_healthcheck_output_text(out, value) != NGX_OK
        || ngx_http_upstream_healthcheck_output_text(out, "\n") != NGX_OK)
    {
        return NGX_ERROR;
    }

    return NGX_OK;
}

static ngx_int_t
ngx_http_upstream_healthcheck_output_sample_uint(
    ngx_http_upstream_healthcheck_output_t *out, const char *name,
    const ngx_http_upstream_healthcheck_sample_t *sample, ngx_uint_t value)
{
    u_char   buffer[NGX_INT_T_LEN + 1];
    u_char  *last;

    last = ngx_sprintf(buffer, "%ui", value);
    *last = '\0';
    return ngx_http_upstream_healthcheck_output_sample(out, name, sample,
                                                       (char *) buffer);
}

static ngx_int_t
ngx_http_upstream_healthcheck_output_sample_int(
    ngx_http_upstream_healthcheck_output_t *out, const char *name,
    const ngx_http_upstream_healthcheck_sample_t *sample, ngx_int_t value)
{
    u_char   buffer[NGX_INT_T_LEN + 2];
    u_char  *last;

    last = ngx_sprintf(buffer, "%i", value);
    *last = '\0';
    return ngx_http_upstream_healthcheck_output_sample(out, name, sample,
                                                       (char *) buffer);
}

static ngx_int_t
ngx_http_upstream_healthcheck_output_group_uint(
    ngx_http_upstream_healthcheck_output_t *out, const char *name,
    const ngx_str_t *upstream, ngx_uint_t value)
{
    if (ngx_http_upstream_healthcheck_output_text(out, name) != NGX_OK
        || ngx_http_upstream_healthcheck_output_group_label(out, upstream)
           != NGX_OK
        || ngx_http_upstream_healthcheck_output_text(out, " ") != NGX_OK
        || ngx_http_upstream_healthcheck_output_uint(out, value) != NGX_OK
        || ngx_http_upstream_healthcheck_output_text(out, "\n") != NGX_OK)
    {
        return NGX_ERROR;
    }

    return NGX_OK;
}

static ngx_int_t
ngx_http_upstream_healthcheck_output_group_time(
    ngx_http_upstream_healthcheck_output_t *out, const char *name,
    const ngx_str_t *upstream, time_t value)
{
    if (ngx_http_upstream_healthcheck_output_text(out, name) != NGX_OK
        || ngx_http_upstream_healthcheck_output_group_label(out, upstream)
           != NGX_OK
        || ngx_http_upstream_healthcheck_output_text(out, " ") != NGX_OK
        || ngx_http_upstream_healthcheck_output_int(out, value) != NGX_OK
        || ngx_http_upstream_healthcheck_output_text(out, "\n") != NGX_OK)
    {
        return NGX_ERROR;
    }

    return NGX_OK;
}

static ngx_int_t
ngx_http_upstream_healthcheck_copy_string(ngx_pool_t *pool, ngx_str_t *dst,
    const ngx_str_t *src)
{
    dst->len = src->len;
    if (src->len == 0) {
        dst->data = NULL;
        return NGX_OK;
    }

    dst->data = ngx_pnalloc(pool, src->len + 1);
    if (dst->data == NULL) {
        return NGX_ERROR;
    }

    ngx_memcpy(dst->data, src->data, src->len);
    dst->data[src->len] = '\0';
    return NGX_OK;
}

static ngx_int_t
ngx_http_upstream_healthcheck_parse_uint(ngx_str_t *value, ngx_uint_t min,
    ngx_uint_t max, ngx_uint_t *result)
{
    ngx_uint_t  i, digit, number;

    if (value->len == 0) {
        return NGX_ERROR;
    }

    number = 0;
    for (i = 0; i < value->len; i++) {
        if (value->data[i] < '0' || value->data[i] > '9') {
            return NGX_ERROR;
        }
        digit = value->data[i] - '0';
        if (digit > max || number > (max - digit) / 10) {
            return NGX_ERROR;
        }
        number = number * 10 + digit;
    }

    if (number < min || number > max) {
        return NGX_ERROR;
    }

    *result = number;
    return NGX_OK;
}

static ngx_int_t
ngx_http_upstream_healthcheck_parse_msec(ngx_str_t *value,
    ngx_msec_t *result)
{
    time_t  parsed;

    parsed = ngx_parse_time(value, 0);
    if (parsed == NGX_ERROR || parsed <= 0
        || parsed > (time_t) NGX_MAX_INT_T_VALUE)
    {
        return NGX_ERROR;
    }

    *result = (ngx_msec_t) parsed;
    return NGX_OK;
}

static ngx_int_t
ngx_http_upstream_healthcheck_has_crlf(const ngx_str_t *value)
{
    size_t  i;

    for (i = 0; i < value->len; i++) {
        if (value->data[i] == '\r' || value->data[i] == '\n') {
            return NGX_ERROR;
        }
    }

    return NGX_OK;
}

static ngx_int_t
ngx_http_upstream_healthcheck_same_string(const ngx_str_t *a,
    const ngx_str_t *b)
{
    return a->len == b->len && (a->len == 0
           || ngx_memcmp(a->data, b->data, a->len) == 0);
}

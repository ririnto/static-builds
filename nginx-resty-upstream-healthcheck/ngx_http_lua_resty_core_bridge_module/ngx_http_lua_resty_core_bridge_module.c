#include <nginx.h>
#include <ngx_config.h>
#include <ngx_core.h>
#include <ngx_http.h>

#include <lua.h>

#define NGX_HTTP_LUA_RESTY_CORE_BRIDGE_REGEX_COMPILE_ONCE 0x01
#define NGX_HTTP_LUA_RESTY_CORE_BRIDGE_REGEX_JIT 0x04

#if (NGX_PCRE2)
#define NGX_HTTP_LUA_RESTY_CORE_BRIDGE_REGEX_CASELESS 0x00000008
#else
#define NGX_HTTP_LUA_RESTY_CORE_BRIDGE_REGEX_CASELESS 0x00000001
#endif

typedef struct {
  void *pool;
  unsigned char *name_table;
  int name_count;
  int name_entry_size;
  int ncaptures;
  int *captures;
  void *regex;
  void *regex_sd;
  void *replace;
  const unsigned char *pattern;
} ngx_http_lua_resty_core_bridge_regex_state_t;

typedef struct ngx_http_lua_regex_s ngx_http_lua_regex_t;

extern ngx_int_t ngx_http_lua_add_package_preload(ngx_conf_t *cf,
                                                  const char *package,
                                                  lua_CFunction func);

extern int ngx_http_lua_ffi_shdict_get(void *zone, const unsigned char *key,
                                       size_t key_len, int *value_type,
                                       unsigned char **str_value_buf,
                                       size_t *str_value_len, double *num_value,
                                       int *user_flags, int get_stale,
                                       int *is_stale, char **errmsg);
extern int ngx_http_lua_ffi_shdict_incr(void *zone, const unsigned char *key,
                                        size_t key_len, double *value,
                                        char **err, int has_init, double init,
                                        long init_ttl, int *forcible);
extern int ngx_http_lua_ffi_shdict_store(void *zone, int op,
                                         const unsigned char *key,
                                         size_t key_len, int value_type,
                                         const unsigned char *str_value_buf,
                                         size_t str_value_len, double num_value,
                                         long exptime, int user_flags,
                                         char **errmsg, int *forcible);
extern int ngx_http_lua_ffi_shdict_flush_all(void *zone);
extern long ngx_http_lua_ffi_shdict_get_ttl(void *zone,
                                            const unsigned char *key,
                                            size_t key_len);
extern int ngx_http_lua_ffi_shdict_set_expire(void *zone,
                                              const unsigned char *key,
                                              size_t key_len, long exptime);
extern size_t ngx_http_lua_ffi_shdict_capacity(void *zone);
extern size_t ngx_http_lua_ffi_shdict_free_space(void *zone);
extern void *ngx_http_lua_ffi_shdict_udata_to_zone(void *zone_udata);
extern ngx_http_lua_regex_t *
ngx_http_lua_ffi_compile_regex(const unsigned char *pat, size_t pat_len,
                               int flags, int pcre_opts, unsigned char *errstr,
                               size_t errstr_size);
extern int ngx_http_lua_ffi_exec_regex(ngx_http_lua_regex_t *re, int flags,
                                       const unsigned char *s, size_t len,
                                       int pos);
extern void ngx_http_lua_ffi_destroy_regex(ngx_http_lua_regex_t *re);

typedef struct {
  int (*shdict_get)(void *zone, const unsigned char *key, size_t key_len,
                    int *value_type, unsigned char **str_value_buf,
                    size_t *str_value_len, double *num_value, int *user_flags,
                    int get_stale, int *is_stale, char **errmsg);
  int (*shdict_incr)(void *zone, const unsigned char *key, size_t key_len,
                     double *value, char **err, int has_init, double init,
                     long init_ttl, int *forcible);
  int (*shdict_store)(void *zone, int op, const unsigned char *key,
                      size_t key_len, int value_type,
                      const unsigned char *str_value_buf, size_t str_value_len,
                      double num_value, long exptime, int user_flags,
                      char **errmsg, int *forcible);
  int (*shdict_flush_all)(void *zone);
  long (*shdict_get_ttl)(void *zone, const unsigned char *key, size_t key_len);
  int (*shdict_set_expire)(void *zone, const unsigned char *key, size_t key_len,
                           long exptime);
  size_t (*shdict_capacity)(void *zone);
  size_t (*shdict_free_space)(void *zone);
  void *(*shdict_udata_to_zone)(void *zone_udata);
} ngx_http_lua_resty_core_bridge_shdict_t;

typedef struct {
  void *(*compile)(const unsigned char *pat, size_t pat_len,
                   const unsigned char *opts, size_t opts_len, int *flags,
                   unsigned char *errstr, size_t errstr_size);
  int (*exec)(void *re, int flags, const unsigned char *s, size_t len, int pos);
  void (*destroy)(void *re);
  int (*capture)(void *re, int capture_idx, int *from, int *to);
} ngx_http_lua_resty_core_bridge_regex_t;

static ngx_int_t ngx_http_lua_resty_core_bridge_init(ngx_conf_t *cf);
static int ngx_http_lua_resty_core_bridge_preload(lua_State *L);
static void *ngx_http_lua_resty_core_bridge_regex_compile(
    const unsigned char *pat, size_t pat_len, const unsigned char *opts,
    size_t opts_len, int *flags, unsigned char *errstr, size_t errstr_size);
static int ngx_http_lua_resty_core_bridge_regex_exec(void *re, int flags,
                                                     const unsigned char *s,
                                                     size_t len, int pos);
static void ngx_http_lua_resty_core_bridge_regex_destroy(void *re);
static int ngx_http_lua_resty_core_bridge_regex_capture(void *re,
                                                        int capture_idx,
                                                        int *from, int *to);
static ngx_int_t ngx_http_lua_resty_core_bridge_regex_parse_opts(
    const unsigned char *opts, size_t opts_len, int *flags, int *pcre_opts,
    unsigned char *errstr, size_t errstr_size);

static ngx_http_lua_resty_core_bridge_shdict_t
    ngx_http_lua_resty_core_bridge_shdict = {
        ngx_http_lua_ffi_shdict_get,
        ngx_http_lua_ffi_shdict_incr,
        ngx_http_lua_ffi_shdict_store,
        ngx_http_lua_ffi_shdict_flush_all,
        ngx_http_lua_ffi_shdict_get_ttl,
        ngx_http_lua_ffi_shdict_set_expire,
        ngx_http_lua_ffi_shdict_capacity,
        ngx_http_lua_ffi_shdict_free_space,
        ngx_http_lua_ffi_shdict_udata_to_zone};

static ngx_http_lua_resty_core_bridge_regex_t
    ngx_http_lua_resty_core_bridge_regex = {
        ngx_http_lua_resty_core_bridge_regex_compile,
        ngx_http_lua_resty_core_bridge_regex_exec,
        ngx_http_lua_resty_core_bridge_regex_destroy,
        ngx_http_lua_resty_core_bridge_regex_capture};

static ngx_http_module_t ngx_http_lua_resty_core_bridge_module_ctx = {
    NULL, ngx_http_lua_resty_core_bridge_init, NULL, NULL, NULL, NULL, NULL,
    NULL};

ngx_module_t ngx_http_lua_resty_core_bridge_module = {
    NGX_MODULE_V1, &ngx_http_lua_resty_core_bridge_module_ctx,
    NULL,          NGX_HTTP_MODULE,
    NULL,          NULL,
    NULL,          NULL,
    NULL,          NULL,
    NULL,          NGX_MODULE_V1_PADDING};

static ngx_int_t ngx_http_lua_resty_core_bridge_init(ngx_conf_t *cf) {
  if (ngx_http_lua_add_package_preload(
          cf, "ngx.resty_core_bridge",
          ngx_http_lua_resty_core_bridge_preload) != NGX_OK) {
    return NGX_ERROR;
  }

  return NGX_OK;
}

static int ngx_http_lua_resty_core_bridge_preload(lua_State *L) {
  lua_createtable(L, 0, 2);
  lua_pushlightuserdata(L, &ngx_http_lua_resty_core_bridge_shdict);
  lua_setfield(L, -2, "shdict");
  lua_pushlightuserdata(L, &ngx_http_lua_resty_core_bridge_regex);
  lua_setfield(L, -2, "regex");
  return 1;
}

static ngx_int_t ngx_http_lua_resty_core_bridge_regex_parse_opts(
    const unsigned char *opts, size_t opts_len, int *flags, int *pcre_opts,
    unsigned char *errstr, size_t errstr_size) {
  size_t i;

  *flags = 0;
  *pcre_opts = 0;

  for (i = 0; i < opts_len; i++) {
    switch (opts[i]) {
    case 'j':
      *flags |= NGX_HTTP_LUA_RESTY_CORE_BRIDGE_REGEX_JIT;
      break;

    case 'o':
      *flags |= NGX_HTTP_LUA_RESTY_CORE_BRIDGE_REGEX_COMPILE_ONCE;
      break;

    case 'i':
      *pcre_opts |= NGX_HTTP_LUA_RESTY_CORE_BRIDGE_REGEX_CASELESS;
      break;

    default:
      if (errstr != NULL && 0 < errstr_size) {
        ngx_snprintf(errstr, errstr_size, "unsupported regex options%Z");
      }

      return NGX_ERROR;
    }
  }

  return NGX_OK;
}

static void *ngx_http_lua_resty_core_bridge_regex_compile(
    const unsigned char *pat, size_t pat_len, const unsigned char *opts,
    size_t opts_len, int *flags, unsigned char *errstr, size_t errstr_size) {
  int pcre_opts;

  if (ngx_http_lua_resty_core_bridge_regex_parse_opts(
          opts, opts_len, flags, &pcre_opts, errstr, errstr_size) != NGX_OK) {
    return NULL;
  }

  return ngx_http_lua_ffi_compile_regex(pat, pat_len, *flags, pcre_opts, errstr,
                                        errstr_size);
}

static int ngx_http_lua_resty_core_bridge_regex_exec(void *re, int flags,
                                                     const unsigned char *s,
                                                     size_t len, int pos) {
  return ngx_http_lua_ffi_exec_regex((ngx_http_lua_regex_t *)re, flags, s, len,
                                     pos);
}

static void ngx_http_lua_resty_core_bridge_regex_destroy(void *re) {
  ngx_http_lua_ffi_destroy_regex((ngx_http_lua_regex_t *)re);
}

static int ngx_http_lua_resty_core_bridge_regex_capture(void *re,
                                                        int capture_idx,
                                                        int *from, int *to) {
  ngx_http_lua_resty_core_bridge_regex_state_t *state;
  int offset;

  if (capture_idx < 0 || 1 < capture_idx) {
    return NGX_DECLINED;
  }

  state = (ngx_http_lua_resty_core_bridge_regex_state_t *)re;
  if (state == NULL || state->captures == NULL ||
      state->ncaptures < capture_idx) {
    return NGX_DECLINED;
  }

  offset = capture_idx * 2;
  if (state->captures[offset] < 0 || state->captures[offset + 1] < 0) {
    return NGX_DECLINED;
  }

  *from = state->captures[offset];
  *to = state->captures[offset + 1];

  return NGX_OK;
}

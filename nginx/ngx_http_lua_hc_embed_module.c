#include <ngx_config.h>
#include <ngx_core.h>
#include <ngx_http.h>

#include <lua.h>
#include <lauxlib.h>

#include "ngx_http_lua_api.h"

/** Embedded LuaJIT bytecode symbol generated from resty/upstream/healthcheck.lua. */
extern const char luaJIT_BC_resty_upstream_healthcheck[];

static ngx_int_t ngx_http_lua_hc_embed_init(ngx_conf_t *cf);
static int ngx_http_lua_hc_embed_loader(lua_State *L);

static ngx_http_module_t ngx_http_lua_hc_embed_module_ctx = {
    ngx_http_lua_hc_embed_init,
    NULL,
    NULL,
    NULL,
    NULL,
    NULL,
    NULL,
    NULL
};

/** NGINX module definition for embedded lua-resty-upstream-healthcheck preload support. */
ngx_module_t ngx_http_lua_hc_embed_module = {
    NGX_MODULE_V1,
    &ngx_http_lua_hc_embed_module_ctx,
    NULL,
    NGX_HTTP_MODULE,
    NULL,
    NULL,
    NULL,
    NULL,
    NULL,
    NULL,
    NULL,
    NGX_MODULE_V1_PADDING
};

static ngx_int_t
ngx_http_lua_hc_embed_init(ngx_conf_t *cf)
{
    return ngx_http_lua_add_package_preload(cf, "resty.upstream.healthcheck",
                                            ngx_http_lua_hc_embed_loader);
}

static int
ngx_http_lua_hc_embed_loader(lua_State *L)
{
    if (luaL_loadbuffer(L,
                        luaJIT_BC_resty_upstream_healthcheck,
                        ~(size_t) 0,
                        "resty.upstream.healthcheck") != 0)
    {
        return lua_error(L);
    }

    return 1;
}

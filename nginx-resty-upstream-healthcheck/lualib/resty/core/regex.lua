local ffi = require "ffi"
local bit = require "bit"
local base = require "resty.core.base"
local bridge = require "ngx.resty_core_bridge"

local ngx = ngx
local ffi_cast = ffi.cast
local ffi_gc = ffi.gc
local ffi_new = ffi.new
local ffi_str = ffi.string
local band = bit.band
local tostring = tostring
local type = type

local MAX_ERR_MSG_LEN = 256
local FLAG_COMPILE_ONCE = 0x01
local PCRE_ERROR_NOMATCH = -1
local STATUS_LINE_REGEX = [[^HTTP/\d+\.\d+\s+(\d+)]]
local UPSTREAM_NAME_REGEX = [[^(.*):\d+$]]

local _M = {
    version = base.version,
}

if type(ngx) ~= "table" then
    return _M
end

ffi.cdef[[
typedef struct {
    void *(*compile)(const unsigned char *pat, size_t pat_len,
        const unsigned char *opts, size_t opts_len, int *flags,
        unsigned char *errstr, size_t errstr_size);
    int (*exec)(void *re, int flags, const unsigned char *s,
        size_t len, int pos);
    void (*destroy)(void *re);
    int (*capture)(void *re, int capture_idx, int *from, int *to);
} ngx_http_lua_resty_core_bridge_regex_t;
]]

local bridge_ptr = bridge and bridge.regex
if bridge_ptr == nil then
    error("resty core regex bridge unavailable")
end

local regex_bridge = ffi_cast("ngx_http_lua_resty_core_bridge_regex_t*", bridge_ptr)
local regex_cache = {}
local capture_from = ffi_new("int[1]")
local capture_to = ffi_new("int[1]")
local compile_flags = ffi_new("int[1]")

local supported_patterns = {
    [STATUS_LINE_REGEX] = {
        opts = "joi",
    },
    [UPSTREAM_NAME_REGEX] = {
        opts = "jo",
    },
}

local function get_supported_entry(regex, opts)
    local entry = supported_patterns[regex]
    if entry == nil then
        return nil, "unsupported regex"
    end

    if opts ~= entry.opts then
        return nil, "unsupported regex options"
    end

    return entry
end

local function get_compiled_regex(regex, opts)
    local cache_key = regex .. "\0" .. opts
    local cached = regex_cache[cache_key]
    if cached ~= nil then
        return cached
    end

    local errbuf = base.get_string_buf(MAX_ERR_MSG_LEN)
    local compiled = regex_bridge.compile(regex, #regex, opts, #opts, compile_flags, errbuf, MAX_ERR_MSG_LEN)
    if compiled == nil then
        return nil, ffi_str(errbuf)
    end

    cached = {
        compiled = ffi_gc(compiled, regex_bridge.destroy),
        flags = tonumber(compile_flags[0]),
    }

    if band(cached.flags, FLAG_COMPILE_ONCE) ~= 0 then
        regex_cache[cache_key] = cached
    end

    return cached
end

local function patch_re_find()
    local re = ngx.re
    if type(re) ~= "table" then
        re = {}
        ngx.re = re
    end

    if type(re.find) == "function" then
        return
    end

    re.find = function(subject, regex, opts, _, nth)
        if type(subject) ~= "string" then
            return nil, nil, "subject is not a string"
        end

        if nth ~= nil and nth ~= 1 then
            return nil, nil, "unsupported capture index"
        end

        local _, support_err = get_supported_entry(regex, opts)
        if support_err then
            return nil, nil, support_err
        end

        local compiled, compile_err = get_compiled_regex(regex, opts)
        if compiled == nil then
            return nil, nil, compile_err
        end

        local rc = regex_bridge.exec(compiled.compiled, compiled.flags, subject, #subject, 0)
        if rc == PCRE_ERROR_NOMATCH then
            return nil
        end

        if rc < 0 then
            return nil, nil, "pcre_exec() failed: " .. rc
        end

        if rc == 0 then
            return nil, nil, "capture size too small"
        end

        local capture_idx = nth or 0
        if rc <= capture_idx then
            return nil, nil
        end

        local cap_rc = regex_bridge.capture(compiled.compiled, capture_idx, capture_from, capture_to)
        if cap_rc ~= 0 then
            return nil, nil
        end

        return tonumber(capture_from[0]) + 1, tonumber(capture_to[0])
    end
end

patch_re_find()

return _M

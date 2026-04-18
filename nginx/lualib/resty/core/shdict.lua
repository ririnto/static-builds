local ffi = require "ffi"
local base = require "resty.core.base"
local bridge = require "ngx.resty_core_bridge"

local error = error
local ffi_new = ffi.new
local ffi_str = ffi.string
local next = next
local tonumber = tonumber
local tostring = tostring
local type = type
local getmetatable = getmetatable
local ngx_shared = ngx.shared

local _M = {
    version = base.version,
}

if not ngx_shared then
    return _M
end

ffi.cdef[[
typedef struct {
    int (*shdict_get)(void *zone, const unsigned char *key, size_t key_len,
        int *value_type, unsigned char **str_value_buf,
        size_t *str_value_len, double *num_value, int *user_flags,
        int get_stale, int *is_stale, char **errmsg);
    int (*shdict_incr)(void *zone, const unsigned char *key, size_t key_len,
        double *value, char **err, int has_init, double init,
        long init_ttl, int *forcible);
    int (*shdict_store)(void *zone, int op, const unsigned char *key,
        size_t key_len, int value_type, const unsigned char *str_value_buf,
        size_t str_value_len, double num_value, long exptime,
        int user_flags, char **errmsg, int *forcible);
    int (*shdict_flush_all)(void *zone);
    long (*shdict_get_ttl)(void *zone, const unsigned char *key,
        size_t key_len);
    int (*shdict_set_expire)(void *zone, const unsigned char *key,
        size_t key_len, long exptime);
    size_t (*shdict_capacity)(void *zone);
    size_t (*shdict_free_space)(void *zone);
    void *(*shdict_udata_to_zone)(void *zone_udata);
} ngx_http_lua_resty_core_bridge_shdict_t;

void free(void *ptr);
]]

local bridge_ptr = bridge and bridge.shdict
if bridge_ptr == nil then
    error("resty core shdict bridge unavailable")
end

local shdict_bridge = ffi.cast("ngx_http_lua_resty_core_bridge_shdict_t*", bridge_ptr)

local value_type = ffi_new("int[1]")
local user_flags = ffi_new("int[1]")
local num_value = ffi_new("double[1]")
local is_stale = ffi_new("int[1]")
local forcible = ffi_new("int[1]")
local str_value_buf = ffi_new("unsigned char *[1]")
local errmsg = base.get_errmsg_ptr()

local function check_zone(zone)
    if not zone or type(zone) ~= "table" then
        error("bad \"zone\" argument", 3)
    end

    zone = zone[1]
    if type(zone) ~= "userdata" then
        error("bad \"zone\" argument", 3)
    end

    zone = shdict_bridge.shdict_udata_to_zone(zone)
    if zone == nil then
        error("bad \"zone\" argument", 3)
    end

    return zone
end

local function shdict_store(zone, op, key, value, exptime, flags)
    zone = check_zone(zone)

    if not exptime then
        exptime = 0
    elseif exptime < 0 then
        error('bad "exptime" argument', 2)
    end

    if not flags then
        flags = 0
    end

    if key == nil then
        return nil, "nil key"
    end

    if type(key) ~= "string" then
        key = tostring(key)
    end

    local key_len = #key
    if key_len == 0 then
        return nil, "empty key"
    end
    if key_len > 65535 then
        return nil, "key too long"
    end

    local str_val_buf
    local str_val_len = 0
    local num_val = 0
    local valtyp = type(value)

    if valtyp == "string" then
        valtyp = 4
        str_val_buf = value
        str_val_len = #value
    elseif valtyp == "number" then
        valtyp = 3
        num_val = value
    elseif value == nil then
        valtyp = 0
    elseif valtyp == "boolean" then
        valtyp = 1
        num_val = value and 1 or 0
    else
        return nil, "bad value type"
    end

    local rc = shdict_bridge.shdict_store(zone, op, key, key_len,
        valtyp, str_val_buf, str_val_len, num_val, exptime * 1000,
        flags, errmsg, forcible)

    if rc == 0 then
        return true, nil, forcible[0] == 1
    end

    return false, ffi_str(errmsg[0]), forcible[0] == 1
end

local function shdict_set(zone, key, value, exptime, flags)
    return shdict_store(zone, 0, key, value, exptime, flags)
end

local function shdict_add(zone, key, value, exptime, flags)
    return shdict_store(zone, 0x0001, key, value, exptime, flags)
end

local function shdict_delete(zone, key)
    return shdict_set(zone, key, nil)
end

local function shdict_get(zone, key)
    zone = check_zone(zone)

    if key == nil then
        return nil, "nil key"
    end

    if type(key) ~= "string" then
        key = tostring(key)
    end

    local key_len = #key
    if key_len == 0 then
        return nil, "empty key"
    end
    if key_len > 65535 then
        return nil, "key too long"
    end

    local size = base.get_string_buf_size()
    local buf = base.get_string_buf(size)
    str_value_buf[0] = buf
    local value_len = base.get_size_ptr()
    value_len[0] = size

    local rc = shdict_bridge.shdict_get(zone, key, key_len, value_type,
        str_value_buf, value_len, num_value, user_flags, 0, is_stale, errmsg)

    if rc ~= 0 then
        if errmsg[0] ~= nil then
            return nil, ffi_str(errmsg[0])
        end

        error("failed to get the key")
    end

    local typ = value_type[0]
    if typ == 0 then
        return nil
    end

    local flags = tonumber(user_flags[0])
    local val

    if typ == 4 then
        if str_value_buf[0] ~= buf then
            buf = str_value_buf[0]
            val = ffi_str(buf, value_len[0])
            ffi.C.free(buf)
        else
            val = ffi_str(buf, value_len[0])
        end
    elseif typ == 3 then
        val = tonumber(num_value[0])
    elseif typ == 1 then
        val = tonumber(buf[0]) ~= 0
    else
        error("unknown value type: " .. typ)
    end

    if flags ~= 0 then
        return val, flags
    end

    return val
end

local function shdict_incr(zone, key, value)
    zone = check_zone(zone)

    if key == nil then
        return nil, "nil key"
    end

    if type(key) ~= "string" then
        key = tostring(key)
    end

    local key_len = #key
    if key_len == 0 then
        return nil, "empty key"
    end
    if key_len > 65535 then
        return nil, "key too long"
    end

    if type(value) ~= "number" then
        value = tonumber(value)
    end

    num_value[0] = value

    local rc = shdict_bridge.shdict_incr(zone, key, key_len, num_value,
        errmsg, 0, 0, 0, forcible)

    if rc ~= 0 then
        return nil, ffi_str(errmsg[0])
    end

    return tonumber(num_value[0])
end

local function shdict_flush_all(zone)
    zone = check_zone(zone)
    shdict_bridge.shdict_flush_all(zone)
end

local function shdict_ttl(zone, key)
    zone = check_zone(zone)

    if key == nil then
        return nil, "nil key"
    end

    if type(key) ~= "string" then
        key = tostring(key)
    end

    local key_len = #key
    if key_len == 0 then
        return nil, "empty key"
    end

    if key_len > 65535 then
        return nil, "key too long"
    end

    local rc = shdict_bridge.shdict_get_ttl(zone, key, key_len)
    if rc == base.FFI_DECLINED then
        return nil, "not found"
    end

    return tonumber(rc) / 1000
end

local function shdict_expire(zone, key, exptime)
    zone = check_zone(zone)

    if not exptime then
        error('bad "exptime" argument', 2)
    end

    if key == nil then
        return nil, "nil key"
    end

    if type(key) ~= "string" then
        key = tostring(key)
    end

    local key_len = #key
    if key_len == 0 then
        return nil, "empty key"
    end

    if key_len > 65535 then
        return nil, "key too long"
    end

    local rc = shdict_bridge.shdict_set_expire(zone, key, key_len, exptime * 1000)
    if rc == base.FFI_DECLINED then
        return nil, "not found"
    end

    return true
end

local function shdict_capacity(zone)
    zone = check_zone(zone)
    return tonumber(shdict_bridge.shdict_capacity(zone))
end

local function shdict_free_space(zone)
    zone = check_zone(zone)
    return tonumber(shdict_bridge.shdict_free_space(zone))
end

local _, dict = next(ngx_shared, nil)
if dict then
    local mt = getmetatable(dict)
    if mt then
        mt = mt.__index
        if mt then
            mt.get = shdict_get
            mt.incr = shdict_incr
            mt.set = shdict_set
            mt.add = shdict_add
            mt.delete = shdict_delete
            mt.flush_all = shdict_flush_all
            mt.ttl = shdict_ttl
            mt.expire = shdict_expire
            mt.capacity = shdict_capacity
            mt.free_space = shdict_free_space
        end
    end
end

return _M

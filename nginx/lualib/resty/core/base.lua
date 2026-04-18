local ffi = require "ffi"

local ceil = math.ceil
local pcall = pcall

local str_buf_size = 4096
local str_buf
local size_ptr
local errmsg

local ok, new_tab = pcall(require, "table.new")
if not ok then
    new_tab = function()
        return {}
    end
end

local c_buf_type = ffi.typeof("char[?]")

local _M = new_tab(0, 8)

_M.version = "runtime-ffi-bridge"
_M.FFI_DECLINED = -5

function _M.get_errmsg_ptr()
    if not errmsg then
        errmsg = ffi.new("char *[1]")
    end

    return errmsg
end

function _M.get_size_ptr()
    if not size_ptr then
        size_ptr = ffi.new("size_t[1]")
    end

    return size_ptr
end

function _M.get_string_buf_size()
    return str_buf_size
end

function _M.get_string_buf(size, must_alloc)
    if size > str_buf_size or must_alloc then
        return ffi.new(c_buf_type, size)
    end

    if not str_buf then
        str_buf = ffi.new(c_buf_type, str_buf_size)
    end

    return str_buf
end

function _M.set_string_buf_size(size)
    if size <= 0 then
        return
    end

    str_buf = nil
    str_buf_size = ceil(size)
end

return _M

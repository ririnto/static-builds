function parser_error(message) {
  print "Error: invalid metadata schema: " message > "/dev/stderr"
  exit 1
}
function skip_ws(c) {
  c = substr(JSON, POS, 1)
  while (POS <= LEN && (c == " " || c == "\n" || c == "\r" || c == "\t")) {
    POS++
    c = substr(JSON, POS, 1)
  }
}
function expect_char(ch,    c) {
  skip_ws()
  c = substr(JSON, POS, 1)
  if (c != ch) {
    parser_error("expected  ch  at byte " POS)
  }
  POS++
}
function parse_literal(literal) {
  skip_ws()
  if (substr(JSON, POS, length(literal)) != literal) {
    parser_error("expected  literal  at byte " POS)
  }
  POS += length(literal)
}
function parse_number(c) {
  skip_ws()
  c = substr(JSON, POS, 1)
  if (c == "-") {
    POS++
  }
  c = substr(JSON, POS, 1)
  if (c == "0") {
    POS++
  } else {
    if (c !~ /[0-9]/) {
      parser_error("invalid number at byte " POS)
    }
    while (substr(JSON, POS, 1) ~ /[0-9]/) {
      POS++
    }
  }
  if (substr(JSON, POS, 1) == ".") {
    POS++
    if (substr(JSON, POS, 1) !~ /[0-9]/) {
      parser_error("invalid number fraction at byte " POS)
    }
    while (substr(JSON, POS, 1) ~ /[0-9]/) {
      POS++
    }
  }
  c = substr(JSON, POS, 1)
  if (c == "e" || c == "E") {
    POS++
    c = substr(JSON, POS, 1)
    if (c == "+" || c == "-") {
      POS++
    }
    if (substr(JSON, POS, 1) !~ /[0-9]/) {
      parser_error("invalid number exponent at byte " POS)
    }
    while (substr(JSON, POS, 1) ~ /[0-9]/) {
      POS++
    }
  }
}
function parse_string(c, esc, out, hex) {
  skip_ws()
  if (substr(JSON, POS, 1) != "\"") {
    parser_error("expected string at byte " POS)
  }
  POS++
  out = ""
  c = substr(JSON, POS, 1)
  while (POS <= LEN && c != "\"") {
    if (c == "\\") {
      POS++
      if (POS > LEN) {
        parser_error("unterminated escape sequence at byte " POS)
      }
      esc = substr(JSON, POS, 1)
      if (esc == "\"" || esc == "\\" || esc == "/") {
        out = out esc
      } else if (esc == "b") {
        out = out sprintf("%c", 8)
      } else if (esc == "f") {
        out = out sprintf("%c", 12)
      } else if (esc == "n") {
        out = out "\n"
      } else if (esc == "r") {
        out = out "\r"
      } else if (esc == "t") {
        out = out "\t"
      } else if (esc == "u") {
        if (POS + 4 > LEN) {
          parser_error("incomplete unicode escape at byte " POS)
        }
        hex = substr(JSON, POS + 1, 4)
        if (hex !~ /^[0-9A-Fa-f]{4}$/) {
          parser_error("invalid unicode escape at byte " POS)
        }
        parser_error("unsupported unicode escape at byte " POS)
        POS += 4
      } else {
        parser_error("invalid escape sequence at byte " POS)
      }
    } else {
      out = out c
    }
    POS++
    c = substr(JSON, POS, 1)
  }
  if (c == "\"") {
    POS++
    STRING_VALUE = out
  } else {
    parser_error("unterminated string")
  }
}
function parse_value() {
  skip_ws()
  if (POS > LEN) {
    parser_error("unexpected end of input")
  }
  if (substr(JSON, POS, 1) == "{") {
    parse_generic_object()
  } else if (substr(JSON, POS, 1) == "[") {
    parse_generic_array()
  } else if (substr(JSON, POS, 1) == "\"") {
    parse_string()
  } else if (substr(JSON, POS, 4) == "null") {
    parse_literal("null")
  } else if (substr(JSON, POS, 4) == "true") {
    parse_literal("true")
  } else if (substr(JSON, POS, 5) == "false") {
    parse_literal("false")
  } else {
    parse_number()
  }
}
function parse_generic_object(key, c) {
  expect_char("{")
  skip_ws()
  c = substr(JSON, POS, 1)
  while (c != "}") {
    parse_string()
    key = STRING_VALUE
    if (key == "") {
      parser_error("object key must not be empty")
    }
    expect_char(":")
    parse_value()
    skip_ws()
    c = substr(JSON, POS, 1)
    if (c == ",") {
      POS++
    } else if (c != "}") {
      parser_error(sprintf("expected , or %c at byte %d", 125, POS))
    }
  }
  POS++
}
function parse_generic_array(c) {
  expect_char("[")
  skip_ws()
  c = substr(JSON, POS, 1)
  while (c != "]") {
    parse_value()
    skip_ws()
    c = substr(JSON, POS, 1)
    if (c == ",") {
      POS++
    } else if (c != "]") {
      parser_error("expected , or ] at byte " POS)
    }
  }
  POS++
}
function parse_scalar_field(target, field_name) {
  parse_string()
  print "SCALAR" "\t" target "\t" field_name "\t" STRING_VALUE
}
function parse_env(target,    key, c) {
  expect_char("{")
  skip_ws()
  c = substr(JSON, POS, 1)
  while (c != "}") {
    parse_string()
    key = STRING_VALUE
    if (key == "") {
      parser_error("target '" target "' has empty env key")
    }
    expect_char(":")
    parse_string()
    print "ENV" "\t" target "\t" key "\t" STRING_VALUE
    skip_ws()
    c = substr(JSON, POS, 1)
    if (c == ",") {
      POS++
    } else if (c != "}") {
      parser_error("target '" target "' has malformed env object")
    }
  }
  POS++
}
function parse_release_files(target,    c) {
  expect_char("[")
  skip_ws()
  c = substr(JSON, POS, 1)
  while (c != "]") {
    parse_string()
    print "RELEASE_FILE" "\t" target "\t" STRING_VALUE
    skip_ws()
    c = substr(JSON, POS, 1)
    if (c == ",") {
      POS++
    } else if (c != "]") {
      parser_error("target '" target "' has malformed release_files array")
    }
  }
  POS++
}
function parse_download_entry(target, entry_index,    key, c, has_url, has_name, url_value, name_value) {
  has_url = 0
  has_name = 0
  url_value = ""
  name_value = ""
  expect_char("{")
  skip_ws()
  c = substr(JSON, POS, 1)
  if (c == "}") {
    parser_error("target '" target "' has invalid download fields in entry #" entry_index)
  }
  while (c != "}") {
    parse_string()
    key = STRING_VALUE
    expect_char(":")
    if (key == "url") {
      parse_string()
      url_value = STRING_VALUE
      has_url = 1
    } else if (key == "name") {
      parse_string()
      name_value = STRING_VALUE
      has_name = 1
    } else {
      parse_value()
    }
    skip_ws()
    c = substr(JSON, POS, 1)
    if (c == ",") {
      POS++
    } else if (c != "}") {
      parser_error("target '" target "' has invalid download entry #" entry_index)
    }
  }
  POS++
  if (!has_url || !has_name) {
    parser_error("target '" target "' has invalid download fields in entry #" entry_index)
  }
  print "DOWNLOAD" "\t" target "\t" entry_index "\t" url_value "\t" name_value
}
function parse_downloads(target,    c, entry_no) {
  entry_no = 0
  expect_char("[")
  skip_ws()
  c = substr(JSON, POS, 1)
  while (c != "]") {
    entry_no++
    parse_download_entry(target, entry_no)
    skip_ws()
    c = substr(JSON, POS, 1)
    if (c == ",") {
      POS++
    } else if (c != "]") {
      parser_error("target '" target "' has invalid downloads metadata")
    }
  }
  POS++
}
function parse_target(target,    key, c) {
  expect_char("{")
  print "TARGET" "\t" target
  skip_ws()
  c = substr(JSON, POS, 1)
  while (c != "}") {
    parse_string()
    key = STRING_VALUE
    expect_char(":")
    if (key == "tag_prefix") {
      parse_scalar_field(target, "tag_prefix")
    } else if (key == "version_env_var") {
      parse_scalar_field(target, "version_env_var")
    } else if (key == "env") {
      parse_env(target)
    } else if (key == "release_files") {
      parse_release_files(target)
    } else if (key == "downloads") {
      parse_downloads(target)
    } else {
      parse_value()
    }
    skip_ws()
    c = substr(JSON, POS, 1)
    if (c == ",") {
      POS++
    } else if (c != "}") {
      parser_error("target '" target "' must be an object")
    }
  }
  POS++
}
function parse_root(target, c, count) {
  count = 0
  expect_char("{")
  skip_ws()
  c = substr(JSON, POS, 1)
  if (c == "}") {
    parser_error("metadata must be a non-empty object")
  }
  while (c != "}") {
    parse_string()
    target = STRING_VALUE
    if (target == "") {
      parser_error("metadata entries must be objects keyed by target name")
    }
    expect_char(":")
    skip_ws()
    if (substr(JSON, POS, 1) != "{") {
      parser_error("metadata entries must be objects keyed by target name")
    }
    parse_target(target)
    count++
    skip_ws()
    c = substr(JSON, POS, 1)
    if (c == ",") {
      POS++
    } else if (c != "}") {
      parser_error(sprintf("expected , or %c at byte %d", 125, POS))
    }
  }
  POS++
  if (count == 0) {
    parser_error("metadata must be a non-empty object")
  }
}
BEGIN {
  file_path = ARGV[1]
  ARGC = 1
  while ((getline line < file_path) > 0) {
    JSON = JSON line "\n"
  }
  close(file_path)
  LEN = length(JSON)
  POS = 1
  parse_root()
  skip_ws()
  if (POS <= LEN) {
    parser_error("unexpected trailing content at byte " POS)
  }
}

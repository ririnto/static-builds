#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

fail() {
  printf 'Error: %s\n' "$*" >&2
  exit 1
}

usage() {
  script_name=$(basename "$0")
  printf 'Usage: %s <command> [args...]\n' "$script_name" >&2
  printf '\n' >&2
  printf 'Commands:\n' >&2
  printf '  list-targets\n' >&2
  printf '  get-env <target>\n' >&2
  printf '  get-official-version <target>\n' >&2
  printf '  get-tag-prefix <target>\n' >&2
  printf '  get-release-files <target>\n' >&2
  printf '  get-downloads <target>\n' >&2
  exit 1
}

resolve_metadata_path() {
  if [ "${METADATA_PATH_OVERRIDE:-}" ]; then
    printf '%s\n' "$METADATA_PATH_OVERRIDE"
    return 0
  fi
  printf '%s/metadata.json\n' "$ROOT_DIR"
}

parse_metadata_records() {
  metadata_path=$(resolve_metadata_path)
  if [ ! -f "$metadata_path" ]; then
    fail "metadata file not found: $metadata_path"
  fi

  awk '
function parser_error(message) {
  print "Error: invalid metadata schema: " message > "/dev/stderr"
  exit 1
}
function skip_ws(    c) {
  while (POS <= LEN) {
    c = substr(JSON, POS, 1)
    if (c == " " || c == "\n" || c == "\r" || c == "\t") {
      POS++
      continue
    }
    break
  }
}
function expect_char(ch,    c) {
  skip_ws()
  c = substr(JSON, POS, 1)
  if (c != ch) {
    parser_error("expected '" ch "' at byte " POS)
  }
  POS++
}
function parse_literal(literal) {
  skip_ws()
  if (substr(JSON, POS, length(literal)) != literal) {
    parser_error("expected '" literal "' at byte " POS)
  }
  POS += length(literal)
}
function parse_number(    c) {
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
function parse_string(    c, esc, out, hex) {
  skip_ws()
  if (substr(JSON, POS, 1) != "\"") {
    parser_error("expected string at byte " POS)
  }
  POS++
  out = ""
  while (POS <= LEN) {
    c = substr(JSON, POS, 1)
    if (c == "\"") {
      POS++
      STRING_VALUE = out
      return
    }
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
      POS++
      continue
    }
    out = out c
    POS++
  }
  parser_error("unterminated string")
}
function parse_value() {
  skip_ws()
  if (POS > LEN) {
    parser_error("unexpected end of input")
  }
  if (substr(JSON, POS, 1) == "{") {
    parse_generic_object()
    return
  }
  if (substr(JSON, POS, 1) == "[") {
    parse_generic_array()
    return
  }
  if (substr(JSON, POS, 1) == "\"") {
    parse_string()
    return
  }
  if (substr(JSON, POS, 4) == "null") {
    parse_literal("null")
    return
  }
  if (substr(JSON, POS, 4) == "true") {
    parse_literal("true")
    return
  }
  if (substr(JSON, POS, 5) == "false") {
    parse_literal("false")
    return
  }
  parse_number()
}
function parse_generic_object(    key, c) {
  expect_char("{")
  skip_ws()
  if (substr(JSON, POS, 1) == "}") {
    POS++
    return
  }
  while (1) {
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
      continue
    }
    if (c == "}") {
      POS++
      return
    }
    parser_error("expected ',' or '}' at byte " POS)
  }
}
function parse_generic_array(    c) {
  expect_char("[")
  skip_ws()
  if (substr(JSON, POS, 1) == "]") {
    POS++
    return
  }
  while (1) {
    parse_value()
    skip_ws()
    c = substr(JSON, POS, 1)
    if (c == ",") {
      POS++
      continue
    }
    if (c == "]") {
      POS++
      return
    }
    parser_error("expected ',' or ']' at byte " POS)
  }
}
function parse_scalar_field(target, field_name) {
  parse_string()
  print "SCALAR" "\t" target "\t" field_name "\t" STRING_VALUE
}
function parse_env(target,    key, c) {
  expect_char("{")
  skip_ws()
  if (substr(JSON, POS, 1) == "}") {
    POS++
    return
  }
  while (1) {
    parse_string()
    key = STRING_VALUE
    if (key == "") {
      parser_error("target '\''" target "'\'' has empty env key")
    }
    expect_char(":")
    parse_string()
    print "ENV" "\t" target "\t" key "\t" STRING_VALUE
    skip_ws()
    c = substr(JSON, POS, 1)
    if (c == ",") {
      POS++
      continue
    }
    if (c == "}") {
      POS++
      return
    }
    parser_error("target '\''" target "'\'' has malformed env object")
  }
}
function parse_release_files(target,    c) {
  expect_char("[")
  skip_ws()
  if (substr(JSON, POS, 1) == "]") {
    POS++
    return
  }
  while (1) {
    parse_string()
    print "RELEASE_FILE" "\t" target "\t" STRING_VALUE
    skip_ws()
    c = substr(JSON, POS, 1)
    if (c == ",") {
      POS++
      continue
    }
    if (c == "]") {
      POS++
      return
    }
    parser_error("target '\''" target "'\'' has malformed release_files array")
  }
}
function parse_download_entry(target, entry_index,    key, c, has_url, has_name, url_value, name_value) {
  has_url = 0
  has_name = 0
  url_value = ""
  name_value = ""
  expect_char("{")
  skip_ws()
  if (substr(JSON, POS, 1) == "}") {
    parser_error("target '\''" target "'\'' has invalid download fields in entry #" entry_index)
  }
  while (1) {
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
      continue
    }
    if (c == "}") {
      POS++
      break
    }
    parser_error("target '\''" target "'\'' has invalid download entry #" entry_index)
  }
  if (!has_url || !has_name) {
    parser_error("target '\''" target "'\'' has invalid download fields in entry #" entry_index)
  }
  print "DOWNLOAD" "\t" target "\t" entry_index "\t" url_value "\t" name_value
}
function parse_downloads(target,    c, entry_no) {
  entry_no = 0
  expect_char("[")
  skip_ws()
  if (substr(JSON, POS, 1) == "]") {
    POS++
    return
  }
  while (1) {
    entry_no++
    parse_download_entry(target, entry_no)
    skip_ws()
    c = substr(JSON, POS, 1)
    if (c == ",") {
      POS++
      continue
    }
    if (c == "]") {
      POS++
      return
    }
    parser_error("target '\''" target "'\'' has invalid downloads metadata")
  }
}
function parse_target(target,    key, c) {
  expect_char("{")
  print "TARGET" "\t" target
  skip_ws()
  if (substr(JSON, POS, 1) == "}") {
    POS++
    return
  }
  while (1) {
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
      continue
    }
    if (c == "}") {
      POS++
      return
    }
    parser_error("target '\''" target "'\'' must be an object")
  }
}
function parse_root(    target, c, count) {
  count = 0
  expect_char("{")
  skip_ws()
  if (substr(JSON, POS, 1) == "}") {
    parser_error("metadata must be a non-empty object")
  }
  while (1) {
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
      continue
    }
    if (c == "}") {
      POS++
      break
    }
    parser_error("expected ',' or '}' at byte " POS)
  }
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
' "$metadata_path"
}

list_targets() {
  records=$(parse_metadata_records)
  tab=$(printf '\t')
  targets=''
  while IFS="$tab" read -r record_type target _; do
    if [ "$record_type" = 'TARGET' ]; then
      targets="${targets}${targets:+ }${target}"
    fi
  done <<EOF
$records
EOF
  printf '%s\n' "$targets"
}

get_scalar_field() {
  target="$1"
  field_name="$2"
  records=$(parse_metadata_records)
  tab=$(printf '\t')
  found_target=0
  found_field=0
  value=''
  while IFS="$tab" read -r record_type record_target record_field record_value _; do
    if [ "$record_type" = 'TARGET' ] && [ "$record_target" = "$target" ]; then
      found_target=1
      continue
    fi
    if [ "$record_type" = 'SCALAR' ] && [ "$record_target" = "$target" ] && [ "$record_field" = "$field_name" ]; then
      found_field=1
      value="$record_value"
    fi
  done <<EOF
$records
EOF
  if [ "$found_target" -ne 1 ]; then
    fail "unknown target '$target'"
  fi
  if [ "$found_field" -ne 1 ] || [ -z "$value" ]; then
    fail "target '$target' is missing $field_name"
  fi
  printf '%s\n' "$value"
}

get_env() {
  target="$1"
  records=$(parse_metadata_records)
  tab=$(printf '\t')
  found_target=0
  env_count=0
  while IFS="$tab" read -r record_type record_target record_key record_value _; do
    if [ "$record_type" = 'TARGET' ] && [ "$record_target" = "$target" ]; then
      found_target=1
      continue
    fi
    if [ "$record_type" = 'ENV' ] && [ "$record_target" = "$target" ]; then
      env_count=$((env_count + 1))
      printf '%s=%s\n' "$record_key" "$record_value"
    fi
  done <<EOF
$records
EOF
  if [ "$found_target" -ne 1 ]; then
    fail "unknown target '$target'"
  fi
  if [ "$env_count" -eq 0 ]; then
    fail "target '$target' is missing env metadata"
  fi
}

get_release_files() {
  target="$1"
  records=$(parse_metadata_records)
  tab=$(printf '\t')
  found_target=0
  release_count=0
  while IFS="$tab" read -r record_type record_target record_value _; do
    if [ "$record_type" = 'TARGET' ] && [ "$record_target" = "$target" ]; then
      found_target=1
      continue
    fi
    if [ "$record_type" = 'RELEASE_FILE' ] && [ "$record_target" = "$target" ]; then
      release_count=$((release_count + 1))
      printf '%s\n' "$record_value"
    fi
  done <<EOF
$records
EOF
  if [ "$found_target" -ne 1 ]; then
    fail "unknown target '$target'"
  fi
  if [ "$release_count" -eq 0 ]; then
    fail "target '$target' is missing release_files"
  fi
}

get_official_version() {
  target="$1"
  records=$(parse_metadata_records)
  tab=$(printf '\t')
  found_target=0
  version_env_var=''
  while IFS="$tab" read -r record_type record_target record_field record_value _; do
    if [ "$record_type" = 'TARGET' ] && [ "$record_target" = "$target" ]; then
      found_target=1
      continue
    fi
    if [ "$record_type" = 'SCALAR' ] && [ "$record_target" = "$target" ] && [ "$record_field" = 'version_env_var' ]; then
      version_env_var="$record_value"
    fi
  done <<EOF
$records
EOF
  if [ "$found_target" -ne 1 ]; then
    fail "unknown target '$target'"
  fi
  if [ -z "$version_env_var" ]; then
    fail "target '$target' is missing version_env_var"
  fi

  env_found=0
  env_value=''
  while IFS="$tab" read -r record_type record_target record_key record_value _; do
    if [ "$record_type" = 'ENV' ] && [ "$record_target" = "$target" ] && [ "$record_key" = "$version_env_var" ]; then
      env_found=1
      env_value="$record_value"
      break
    fi
  done <<EOF
$records
EOF
  if [ "$env_found" -ne 1 ] || [ -z "$env_value" ]; then
    fail "target '$target' is missing official version value"
  fi
  printf '%s\n' "$env_value"
}

lookup_env_value() {
  lookup_target="$1"
  lookup_key="$2"
  LOOKUP_ENV_FOUND=0
  LOOKUP_ENV_VALUE=''
  while IFS="$tab" read -r record_type record_target record_key record_value _; do
    if [ "$record_type" = 'ENV' ] && [ "$record_target" = "$lookup_target" ] && [ "$record_key" = "$lookup_key" ]; then
      LOOKUP_ENV_FOUND=1
      LOOKUP_ENV_VALUE="$record_value"
      break
    fi
  done <<EOF
$records
EOF
  [ "$LOOKUP_ENV_FOUND" -eq 1 ]
}

render_download_template() {
  render_target="$1"
  entry_index="$2"
  field_name="$3"
  template="$4"
  rendered=''
  remainder="$template"
  while :; do
    case "$remainder" in
    *'{'*)
      prefix=${remainder%%\{*}
      rendered="${rendered}${prefix}"
      remainder=${remainder#*\{}
      case "$remainder" in
      *'}'*)
        placeholder=${remainder%%\}*}
        remainder=${remainder#*\}}
        if ! lookup_env_value "$render_target" "$placeholder"; then
          fail "target '$render_target' references unknown placeholder '$placeholder' in downloads[$entry_index].$field_name"
        fi
        rendered="${rendered}${LOOKUP_ENV_VALUE}"
        ;;
      *)
        rendered="${rendered}{${remainder}"
        break
        ;;
      esac
      ;;
    *)
      rendered="${rendered}${remainder}"
      break
      ;;
    esac
  done
  printf '%s\n' "$rendered"
}

get_target_by_tag() {
  tag="$1"
  records=$(parse_metadata_records)
  tab=$(printf '	')
  found=''
  while IFS="$tab" read -r record_type record_target record_field record_value _; do
    if [ "$record_type" = 'SCALAR' ] && [ "$record_field" = 'tag_prefix' ]; then
      case "$tag" in
      "${record_value}-"*)
        found="$record_target"
        break
        ;;
      esac
    fi
  done <<EOF
$records
EOF
  if [ -z "$found" ]; then
    fail "unable to resolve target for tag '$tag'"
  fi
  printf '%s
' "$found"
}

get_downloads() {
  target="$1"
  records=$(parse_metadata_records)
  tab=$(printf '\t')
  found_target=0
  env_count=0
  downloads_count=0
  while IFS="$tab" read -r record_type record_target _; do
    if [ "$record_type" = 'TARGET' ] && [ "$record_target" = "$target" ]; then
      found_target=1
      continue
    fi
    if [ "$record_type" = 'ENV' ] && [ "$record_target" = "$target" ]; then
      env_count=$((env_count + 1))
      continue
    fi
    if [ "$record_type" = 'DOWNLOAD' ] && [ "$record_target" = "$target" ]; then
      downloads_count=$((downloads_count + 1))
    fi
  done <<EOF
$records
EOF
  if [ "$found_target" -ne 1 ]; then
    fail "unknown target '$target'"
  fi
  if [ "$env_count" -eq 0 ]; then
    fail "target '$target' is missing env metadata"
  fi
  if [ "$downloads_count" -eq 0 ]; then
    return 0
  fi

  while IFS="$tab" read -r record_type record_target entry_index url_template name_template _; do
    if [ "$record_type" = 'DOWNLOAD' ] && [ "$record_target" = "$target" ]; then
      rendered_url=$(render_download_template "$target" "$entry_index" 'url' "$url_template")
      rendered_name=$(render_download_template "$target" "$entry_index" 'name' "$name_template")
      printf '%s\t%s\n' "$rendered_url" "$rendered_name"
    fi
  done <<EOF
$records
EOF
}

main() {
  if [ $# -lt 1 ]; then
    usage
  fi

  command="$1"
  shift

  case "$command" in
  list-targets)
    [ $# -eq 0 ] || usage
    list_targets
    ;;
  get-tag-prefix)
    [ $# -eq 1 ] || usage
    get_scalar_field "$1" 'tag_prefix'
    ;;
  get-official-version)
    [ $# -eq 1 ] || usage
    get_official_version "$1"
    ;;
  get-env)
    [ $# -eq 1 ] || usage
    get_env "$1"
    ;;
  get-release-files)
    [ $# -eq 1 ] || usage
    get_release_files "$1"
    ;;
  get-target-by-tag)
    [ $# -eq 1 ] || usage
    get_target_by_tag "$1"
    ;;
  get-downloads)
    [ $# -eq 1 ] || usage
    get_downloads "$1"
    ;;
  *)
    usage
    ;;
  esac
}

main "$@"

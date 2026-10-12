#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
  printf 'Usage: %s /path/to/linux/nginx\n' "$0" >&2
  exit 2
fi

NGINX_BIN=$1
if [ ! -x "$NGINX_BIN" ]; then
  printf 'NGINX binary is not executable: %s\n' "$NGINX_BIN" >&2
  exit 2
fi
if [ "$(uname -s)" != Linux ]; then
  printf 'The functional suite requires Linux. Use run-docker.sh on macOS.\n' >&2
  exit 2
fi

for command_name in curl date dnsmasq grep mktemp openssl sed setsid sleep socat tr wc; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    printf 'Required command is missing: %s\n' "$command_name" >&2
    exit 2
  fi
done

FIXTURE_DIR=
FIXTURE_PIDS=()
FIXTURE_GROUPS=()

# Checks whether a fixture process group still has a non-zombie member.
# @param group_id Process group ID to search for under procfs.
# @return 0 when the group still has a running process.
# @return-type int
process_group_running() {
  local group_id=$1
  local stat_file
  local stat_line
  local stat_fields
  local -a stat_parts
  local state
  local process_group
  for stat_file in /proc/[0-9]*/stat; do
    if IFS= read -r stat_line <"$stat_file"; then
      stat_fields=${stat_line##*) }
      read -r -a stat_parts <<<"$stat_fields"
      state=${stat_parts[0]}
      process_group=${stat_parts[2]}
      if [ "$process_group" = "$group_id" ] && [ "$state" != Z ]; then
        return 0
      fi
    fi
  done
  return 1
}

# Stops fixture process groups, waits for their leaders, and removes temporary files.
# @return The suite status that triggered cleanup.
# @return-type int
cleanup() {
  local suite_status=$?
  local group
  local pid
  local attempt
  local running
  trap - EXIT INT TERM
  for group in "${FIXTURE_GROUPS[@]}"; do
    if process_group_running "$group"; then
      if kill -TERM -- "-$group"; then
        :
      fi
    fi
  done
  for ((attempt = 0; attempt < 30; attempt++)); do
    running=0
    for group in "${FIXTURE_GROUPS[@]}"; do
      if process_group_running "$group"; then
        running=1
      fi
    done
    if [ "$running" -eq 0 ]; then
      break
    fi
    sleep 0.1
  done
  for group in "${FIXTURE_GROUPS[@]}"; do
    if process_group_running "$group"; then
      if kill -KILL -- "-$group"; then
        :
      fi
    fi
  done
  for pid in "${FIXTURE_PIDS[@]}"; do
    if wait "$pid"; then
      :
    fi
  done
  if [ -n "$FIXTURE_DIR" ]; then
    rm -rf -- "$FIXTURE_DIR"
  fi
  exit "$suite_status"
}

trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

FIXTURE_DIR=$(mktemp -d /tmp/healthcheck-suite.XXXXXX)
chmod 711 "$FIXTURE_DIR"
mkdir -p "$FIXTURE_DIR/logs"
METRICS_FILE="$FIXTURE_DIR/metrics"
HEADERS_FILE="$FIXTURE_DIR/headers"

# Starts a command in a dedicated process group and records its leader.
# @param command Command and arguments to start.
# @return 0 after recording the process group.
# @return-type int
start_fixture() {
  setsid "$@" &
  FIXTURE_PIDS+=("$!")
  FIXTURE_GROUPS+=("$!")
}

# Stops one recorded fixture process group and removes its identifiers from cleanup state.
# @param process_id Fixture process group leader PID.
# @return 0 after stopping and reaping the fixture leader.
# @return-type int
stop_fixture() {
  local process_id=$1
  local index
  local retained_pids=()
  local retained_groups=()
  if process_group_running "$process_id"; then
    if kill -TERM -- "-$process_id"; then
      :
    fi
  fi
  if wait "$process_id"; then
    :
  fi
  for index in "${!FIXTURE_PIDS[@]}"; do
    if [ "${FIXTURE_PIDS[$index]}" != "$process_id" ]; then
      retained_pids+=("${FIXTURE_PIDS[$index]}")
      retained_groups+=("${FIXTURE_GROUPS[$index]}")
    fi
  done
  FIXTURE_PIDS=("${retained_pids[@]}")
  FIXTURE_GROUPS=("${retained_groups[@]}")
}

# Creates trusted and intentionally invalid certificate chains for TLS probes.
# @return 0 after writing the CA and server certificates.
# @return-type int
create_certificates() {
  local valid_start
  local valid_end
  local expired_start
  local expired_end
  local future_start
  local future_end
  local name
  local subject
  local start_date
  local end_date
  mkdir -p "$FIXTURE_DIR/ca/newcerts"
  : >"$FIXTURE_DIR/ca/index.txt"
  printf '1000\n' >"$FIXTURE_DIR/ca/serial"
  openssl req -x509 -newkey rsa:2048 -nodes \
    -keyout "$FIXTURE_DIR/ca.key" \
    -out "$FIXTURE_DIR/ca.pem" \
    -days 3 \
    -subj /CN=healthcheck-suite-ca \
    -addext 'basicConstraints=critical,CA:TRUE' \
    -addext 'keyUsage=critical,keyCertSign,cRLSign'
  openssl req -x509 -newkey rsa:2048 -nodes \
    -keyout "$FIXTURE_DIR/wrong-ca.key" \
    -out "$FIXTURE_DIR/wrong-ca.pem" \
    -days 3 \
    -subj /CN=healthcheck-suite-wrong-ca \
    -addext 'basicConstraints=critical,CA:TRUE' \
    -addext 'keyUsage=critical,keyCertSign,cRLSign'
  cat >"$FIXTURE_DIR/ca.cnf" <<EOF
[ ca ]
default_ca = issuing_ca
[ issuing_ca ]
database = $FIXTURE_DIR/ca/index.txt
new_certs_dir = $FIXTURE_DIR/ca/newcerts
certificate = $FIXTURE_DIR/ca.pem
private_key = $FIXTURE_DIR/ca.key
serial = $FIXTURE_DIR/ca/serial
default_md = sha256
policy = subject_policy
unique_subject = no
x509_extensions = server
[ subject_policy ]
commonName = supplied
[ server ]
basicConstraints = critical,CA:FALSE
keyUsage = critical,digitalSignature,keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = DNS:backend.local,IP:127.0.0.1
EOF
  valid_start=$(date -u -d '5 minutes ago' '+%y%m%d%H%M%SZ')
  valid_end=$(date -u -d '2 days' '+%y%m%d%H%M%SZ')
  expired_start=$(date -u -d '2 days ago' '+%y%m%d%H%M%SZ')
  expired_end=$(date -u -d '1 day ago' '+%y%m%d%H%M%SZ')
  future_start=$(date -u -d '1 hour' '+%y%m%d%H%M%SZ')
  future_end=$(date -u -d '2 hours' '+%y%m%d%H%M%SZ')
  for name in valid expired future; do
    case "$name" in
      valid)
        subject=backend.local
        start_date=$valid_start
        end_date=$valid_end
        ;;
      expired)
        subject=backend-expired.local
        start_date=$expired_start
        end_date=$expired_end
        ;;
      future)
        subject=backend-future.local
        start_date=$future_start
        end_date=$future_end
        ;;
    esac
    openssl req -new -newkey rsa:2048 -nodes \
      -keyout "$FIXTURE_DIR/$name.key" \
      -out "$FIXTURE_DIR/$name.csr" \
      -subj "/CN=$subject"
    openssl ca -batch -config "$FIXTURE_DIR/ca.cnf" \
      -in "$FIXTURE_DIR/$name.csr" \
      -out "$FIXTURE_DIR/$name.pem" \
      -startdate "$start_date" \
      -enddate "$end_date" \
      -extensions server
  done
}

# Fetches health metrics and the worker PID that served the request.
# @return 0 after saving the metrics and response headers.
# @return-type int
fetch_metrics() {
  curl -fsS --max-time 2 -D "$HEADERS_FILE" \
    -o "$METRICS_FILE" http://127.0.0.1:19081/metrics
}

# Reads one named sample for an upstream peer and optional result label.
# @param family Prometheus metric family name.
# @param upstream Configured upstream name.
# @param peer Resolved peer address.
# @param result Optional result label value.
# @return The sample value when exactly one matching series exists.
# @return-type string
metric_value() {
  local family=$1
  local upstream=$2
  local peer=$3
  local result=${4:-}
  local line
  local labels
  local sample_value
  local sample_pattern=',backup="(true|false)"(,result="[A-Z0-9]+")?} [-+0-9.eE]+$'
  local matched=0
  while IFS= read -r line; do
    case "$line" in
      "$family{"*) ;;
      *) continue ;;
    esac
    if [[ ! "$line" =~ $sample_pattern ]]; then
      printf 'Invalid quoted peer labels in metric family %s.\n' "$family" >&2
      return 1
    fi
    labels=${line#*\{}
    labels=${labels%%\}*}
    case ",$labels," in
      *",upstream=\"$upstream\","*) ;;
      *) continue ;;
    esac
    case ",$labels," in
      *",peer=\"$peer\","*) ;;
      *) continue ;;
    esac
    if [ -n "$result" ]; then
      case ",$labels," in
        *",result=\"$result\","*) ;;
        *) continue ;;
      esac
    fi
    sample_value=${line##* }
    matched=$((matched + 1))
  done <"$METRICS_FILE"
  if [ "$matched" -ne 1 ]; then
    return 1
  fi
  printf '%s\n' "$sample_value"
}

# Checks all fixed result series for a peer and enforces one-hot values.
# @param family HTTP or stream result metric family.
# @param upstream Configured upstream name.
# @param peer Resolved peer address.
# @param expected Result name expected to have value one.
# @return 0 when all result series match the expected one-hot state.
# @return-type int
assert_one_hot() {
  local family=$1
  local upstream=$2
  local peer=$3
  local expected=$4
  local result
  local value
  local expected_value
  local failed=0
  local results=(UNKNOWN SOCKERR L4OK L4TOUT L4CON L6OK L6TOUT L6RSP L7OK L7TOUT L7RSP L7STS)
  for result in "${results[@]}"; do
    if value=$(metric_value "$family" "$upstream" "$peer" "$result"); then
      :
    else
      failed=1
      continue
    fi
    expected_value=0
    if [ "$result" = "$expected" ]; then
      expected_value=1
    fi
    if [ "$value" != "$expected_value" ]; then
      failed=1
    fi
  done
  [ "$failed" -eq 0 ]
}

# Polls metrics until a peer reports the expected one-hot result or the deadline expires.
# @param family HTTP or stream result metric family.
# @param upstream Configured upstream name.
# @param peer Resolved peer address.
# @param expected Result name expected to have value one.
# @return 0 when the peer reaches the expected result.
# @return-type int
wait_for_result() {
  local family=$1
  local upstream=$2
  local peer=$3
  local expected=$4
  local attempt
  for ((attempt = 0; attempt < 120; attempt++)); do
    if fetch_metrics && assert_one_hot "$family" "$upstream" "$peer" "$expected"; then
      return 0
    fi
    sleep 0.1
  done
  printf 'Timed out waiting for %s upstream %s peer %s to report %s.\n' \
    "$family" "$upstream" "$peer" "$expected" >&2
  return 1
}

# Polls a peer's readiness gauge until it reaches the requested value.
# @param family Readiness metric family.
# @param upstream Configured upstream name.
# @param peer Resolved peer address.
# @param expected Expected numeric readiness value.
# @return 0 when the readiness gauge reaches the expected value.
# @return-type int
wait_for_readiness() {
  local family=$1
  local upstream=$2
  local peer=$3
  local expected=$4
  local value
  local attempt
  for ((attempt = 0; attempt < 120; attempt++)); do
    if fetch_metrics && value=$(metric_value "$family" "$upstream" "$peer"); then
      if [ "$value" = "$expected" ]; then
        return 0
      fi
    fi
    sleep 0.1
  done
  printf 'Timed out waiting for %s upstream %s peer %s readiness=%s.\n' \
    "$family" "$upstream" "$peer" "$expected" >&2
  return 1
}

# Verifies one TLS directive validation failure using nginx -t.
# @param config_path Invalid NGINX configuration file.
# @param expected_message Required configuration error substring.
# @return 0 when configuration validation rejects the file as expected.
# @return-type int
assert_config_rejected() {
  local config_path=$1
  local expected_message=$2
  local output_path="$FIXTURE_DIR/config-check.log"
  if "$NGINX_BIN" -t -p "$FIXTURE_DIR" -c "$config_path" >"$output_path" 2>&1; then
    printf 'NGINX unexpectedly accepted invalid configuration: %s\n' "$config_path" >&2
    return 1
  fi
  if ! grep -F "$expected_message" "$output_path"; then
    printf 'NGINX rejected configuration for an unexpected reason: %s\n' "$config_path" >&2
    cat "$output_path" >&2
    return 1
  fi
}

# Sends a bounded request through the stream proxy and checks its first response line.
# @param host TCP destination host.
# @param port TCP destination port.
# @param expected Expected response substring.
# @return 0 when a response arrives and contains the expected text.
# @return-type int
assert_stream_response() {
  local host=$1
  local port=$2
  local expected=$3
  local descriptor
  local response
  if ! exec {descriptor}<>"/dev/tcp/$host/$port"; then
    printf 'Could not connect to stream listener %s:%s.\n' "$host" "$port" >&2
    return 1
  fi
  if [ "$port" = 19083 ]; then
    printf 'PING\r\n' >&"$descriptor"
  fi
  if ! IFS= read -r -t 3 response <&"$descriptor"; then
    exec {descriptor}>&-
    printf 'Timed out waiting for stream response from %s:%s.\n' "$host" "$port" >&2
    return 1
  fi
  exec {descriptor}>&-
  response=${response%$'\r'}
  case "$response" in
    *"$expected"*) ;;
    *)
      printf 'Unexpected response from %s:%s: %s\n' "$host" "$port" "$response" >&2
      return 1
      ;;
  esac
}

# Confirms health metrics remain visible when both NGINX workers serve requests.
# @return 0 after observing two worker PIDs with the expected shared peer state.
# @return-type int
assert_multiworker_metrics() {
  local worker
  local attempt
  local first_worker=
  local second_worker=
  for ((attempt = 0; attempt < 120; attempt++)); do
    if fetch_metrics; then
      worker=$(sed -n 's/^[Xx]-[Ww]orker: *//p' "$HEADERS_FILE" | tr -d '\r')
      if [ -n "$worker" ] && assert_one_hot nginx_healthcheck_check_result \
        dynamic_http 127.0.0.2:19080 L7OK; then
        if [ -z "$first_worker" ]; then
          first_worker=$worker
        elif [ "$worker" != "$first_worker" ]; then
          second_worker=$worker
          printf 'Shared health metrics served by workers %s and %s.\n' \
            "$first_worker" "$second_worker"
          return 0
        fi
      fi
    fi
    sleep 0.05
  done
  printf 'Did not observe the shared result metric from two NGINX workers.\n' >&2
  return 1
}

create_certificates
chmod 600 "$FIXTURE_DIR"/*.key
printf '127.0.0.2 backend.local\n127.0.0.3 backend.local\n' >"$FIXTURE_DIR/hosts"
chmod 644 "$FIXTURE_DIR/hosts"
start_fixture dnsmasq --keep-in-foreground --no-resolv --no-hosts \
  --addn-hosts="$FIXTURE_DIR/hosts" --port=15353 --listen-address=127.0.0.1 \
  --bind-interfaces --pid-file="$FIXTURE_DIR/dnsmasq.pid"
start_fixture socat TCP-LISTEN:19443,bind=127.0.0.1,reuseaddr,fork EXEC:'sleep 5' \
  >"$FIXTURE_DIR/tls-stall.log" 2>&1
start_fixture socat TCP-LISTEN:19086,bind=127.0.0.1,reuseaddr,fork EXEC:'sleep 5' \
  >"$FIXTURE_DIR/http-stall.log" 2>&1
start_fixture socat \
  "OPENSSL-LISTEN:19455,bind=127.0.0.1,reuseaddr,fork,cert=$FIXTURE_DIR/valid.pem,key=$FIXTURE_DIR/valid.key,verify=0" \
  'SYSTEM:printf "malformed response\\r\\n"' \
  >"$FIXTURE_DIR/http-malformed.log" 2>&1

cat >"$FIXTURE_DIR/nginx.conf" <<EOF
worker_processes 2;
error_log stderr notice;
pid $FIXTURE_DIR/nginx.pid;
events {
    worker_connections 2048;
    accept_mutex off;
}
http {
    resolver 127.0.0.1:15353 valid=1s ipv6=off;
    resolver_timeout 1s;
    vhost_traffic_status_zone;
    upstream dynamic_http {
        zone dynamic_http 256k;
        server backend.local:19080 resolve;
        server 127.0.0.1:19999 backup;
        healthcheck type=http interval=100ms timeout=200ms fall=2 rise=2;
    }
    upstream l4_http {
        zone l4_http 256k;
        server 127.0.0.2:19080;
        healthcheck type=tcp interval=100ms timeout=200ms fall=1 rise=2;
    }
    upstream l4_http_failure {
        zone l4_http_failure 256k;
        server 127.0.0.1:19999;
        healthcheck type=tcp interval=100ms timeout=200ms fall=1 rise=2;
    }
    upstream ipv6_http {
        zone ipv6_http 256k;
        server [::1]:19084;
        healthcheck type=tcp interval=100ms timeout=200ms fall=1 rise=2;
    }
    upstream tls_only {
        zone tls_only 256k;
        server 127.0.0.1:19444;
        healthcheck type=tls interval=100ms timeout=300ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate $FIXTURE_DIR/ca.pem;
    }
    upstream tls_wrong_name {
        zone tls_wrong_name 256k;
        server 127.0.0.1:19444;
        healthcheck type=tls interval=100ms timeout=300ms fall=1 rise=2;
        healthcheck_tls_name wrong.local;
        healthcheck_trusted_certificate $FIXTURE_DIR/ca.pem;
    }
    upstream tls_wrong_ca {
        zone tls_wrong_ca 256k;
        server 127.0.0.1:19444;
        healthcheck type=tls interval=100ms timeout=300ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate $FIXTURE_DIR/wrong-ca.pem;
    }
    upstream tls_plaintext {
        zone tls_plaintext 256k;
        server 127.0.0.2:19080;
        healthcheck type=tls interval=100ms timeout=300ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate $FIXTURE_DIR/ca.pem;
    }
    upstream tls_stall {
        zone tls_stall 256k;
        server 127.0.0.1:19443;
        healthcheck type=tls interval=100ms timeout=300ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate $FIXTURE_DIR/ca.pem;
    }
    upstream tls_expired {
        zone tls_expired 256k;
        server 127.0.0.1:19448;
        healthcheck type=tls interval=100ms timeout=300ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate $FIXTURE_DIR/ca.pem;
    }
    upstream tls_future {
        zone tls_future 256k;
        server 127.0.0.1:19447;
        healthcheck type=tls interval=100ms timeout=300ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate $FIXTURE_DIR/ca.pem;
    }
    upstream tls_recovery {
        zone tls_recovery 256k;
        server 127.0.0.1:19445;
        healthcheck type=tls interval=100ms timeout=300ms fall=2 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate $FIXTURE_DIR/ca.pem;
    }
    upstream https_success {
        zone https_success 256k;
        server 127.0.0.1:19454;
        healthcheck type=https interval=100ms timeout=500ms fall=1 rise=2 uri=/ok host=backend.local;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate $FIXTURE_DIR/ca.pem;
    }
    upstream https_status {
        zone https_status 256k;
        server 127.0.0.1:19454;
        healthcheck type=https interval=100ms timeout=500ms fall=1 rise=2 uri=/reject host=backend.local;
        healthcheck_statuses 200;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate $FIXTURE_DIR/ca.pem;
    }
    upstream https_timeout {
        zone https_timeout 256k;
        server 127.0.0.1:19454;
        healthcheck type=https interval=100ms timeout=300ms fall=1 rise=2 uri=/slow host=backend.local;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate $FIXTURE_DIR/ca.pem;
    }
    upstream https_malformed {
        zone https_malformed 256k;
        server 127.0.0.1:19455;
        healthcheck type=https interval=100ms timeout=500ms fall=1 rise=2 uri=/ok host=backend.local;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate $FIXTURE_DIR/ca.pem;
    }
    server {
        listen 127.0.0.2:19080;
        listen 127.0.0.4:19080;
        location / { return 200 "\$server_addr\\n"; }
    }
    server {
        listen 127.0.0.3:19080;
        location / {
            if (-f $FIXTURE_DIR/recovered-http) { return 200 "\$server_addr\\n"; }
            return 503;
        }
    }
    server {
        listen 127.0.0.1:19081;
        location /route { proxy_pass http://dynamic_http; }
        location /metrics {
            add_header X-Worker \$pid always;
            healthcheck_metrics;
        }
        location /traffic {
            vhost_traffic_status_display;
            vhost_traffic_status_display_format prometheus;
        }
    }
    server {
        listen 127.0.0.1:19454 ssl;
        ssl_certificate $FIXTURE_DIR/valid.pem;
        ssl_certificate_key $FIXTURE_DIR/valid.key;
        location /ok { return 200 ready; }
        location /reject { return 503 unavailable; }
        location /slow { proxy_pass http://127.0.0.1:19086; }
    }
}
stream {
    resolver 127.0.0.1:15353 valid=1s ipv6=off;
    resolver_timeout 1s;
    upstream dynamic_tcp {
        zone dynamic_tcp 256k;
        server backend.local:19082 resolve;
        server 127.0.0.1:19999 backup;
        healthcheck_tcp type=tcp interval=100ms timeout=200ms fall=2 rise=2;
        healthcheck_send "PING\\r\\n";
        healthcheck_expect "PONG" min_recv=4;
    }
    upstream l4_stream {
        zone l4_stream 256k;
        server 127.0.0.2:19082;
        healthcheck_tcp type=tcp interval=100ms timeout=200ms fall=1 rise=2;
    }
    upstream l4_stream_failure {
        zone l4_stream_failure 256k;
        server 127.0.0.1:19999;
        healthcheck_tcp type=tcp interval=100ms timeout=200ms fall=1 rise=2;
    }
    upstream tls_stream {
        zone tls_stream 256k;
        server 127.0.0.1:19444;
        healthcheck_tcp type=tls interval=100ms timeout=300ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate $FIXTURE_DIR/ca.pem;
    }
    upstream stream_wrong_name {
        zone stream_wrong_name 256k;
        server 127.0.0.1:19444;
        healthcheck_tcp type=tls interval=100ms timeout=300ms fall=1 rise=2;
        healthcheck_tls_name wrong.local;
        healthcheck_trusted_certificate $FIXTURE_DIR/ca.pem;
    }
    upstream stream_wrong_ca {
        zone stream_wrong_ca 256k;
        server 127.0.0.1:19444;
        healthcheck_tcp type=tls interval=100ms timeout=300ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate $FIXTURE_DIR/wrong-ca.pem;
    }
    upstream stream_plaintext {
        zone stream_plaintext 256k;
        server 127.0.0.2:19080;
        healthcheck_tcp type=tls interval=100ms timeout=300ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate $FIXTURE_DIR/ca.pem;
    }
    upstream stream_stall {
        zone stream_stall 256k;
        server 127.0.0.1:19443;
        healthcheck_tcp type=tls interval=100ms timeout=300ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate $FIXTURE_DIR/ca.pem;
    }
    upstream stream_expired {
        zone stream_expired 256k;
        server 127.0.0.1:19448;
        healthcheck_tcp type=tls interval=100ms timeout=300ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate $FIXTURE_DIR/ca.pem;
    }
    upstream stream_future {
        zone stream_future 256k;
        server 127.0.0.1:19447;
        healthcheck_tcp type=tls interval=100ms timeout=300ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate $FIXTURE_DIR/ca.pem;
    }
    upstream stream_tls_recovery {
        zone stream_tls_recovery 256k;
        server 127.0.0.1:19445;
        healthcheck_tcp type=tls interval=100ms timeout=300ms fall=2 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate $FIXTURE_DIR/ca.pem;
    }
    upstream stream_tls_payload {
        zone stream_tls_payload 256k;
        server 127.0.0.1:19446;
        healthcheck_tcp type=tls interval=100ms timeout=500ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate $FIXTURE_DIR/ca.pem;
        healthcheck_send "PING\\r\\n";
        healthcheck_expect "PONG" min_recv=4;
    }
    log_format application_bytes '\$bytes_received';
    server {
        listen 127.0.0.1:19444 ssl;
        ssl_certificate $FIXTURE_DIR/valid.pem;
        ssl_certificate_key $FIXTURE_DIR/valid.key;
        access_log $FIXTURE_DIR/tls-application.log application_bytes;
        return "TLS ready\\n";
    }
    server {
        listen 127.0.0.1:19446 ssl;
        ssl_certificate $FIXTURE_DIR/valid.pem;
        ssl_certificate_key $FIXTURE_DIR/valid.key;
        proxy_pass 127.0.0.2:19082;
    }
    server {
        listen 127.0.0.1:19447 ssl;
        ssl_certificate $FIXTURE_DIR/future.pem;
        ssl_certificate_key $FIXTURE_DIR/future.key;
        return "Future certificate\\n";
    }
    server {
        listen 127.0.0.1:19448 ssl;
        ssl_certificate $FIXTURE_DIR/expired.pem;
        ssl_certificate_key $FIXTURE_DIR/expired.key;
        return "Expired certificate\\n";
    }
    server {
        listen 127.0.0.2:19082;
        listen 127.0.0.4:19082;
        return "\$server_addr PONG\\r\\n";
    }
    server {
        listen 127.0.0.3:19082;
        return "FAIL\\n";
    }
    server {
        listen 127.0.0.1:19083;
        proxy_pass dynamic_tcp;
    }
    server {
        listen [::1]:19084;
        return "IPv6 ready\\n";
    }
}
EOF

cat >"$FIXTURE_DIR/http-missing-name.conf" <<EOF
events {}
http {
    upstream bad_http {
        zone bad_http 64k;
        server 127.0.0.1:19454;
        healthcheck type=https interval=1s timeout=1s;
        healthcheck_trusted_certificate $FIXTURE_DIR/ca.pem;
    }
}
EOF
cat >"$FIXTURE_DIR/http-missing-ca.conf" <<EOF
events {}
http {
    upstream bad_http {
        zone bad_http 64k;
        server 127.0.0.1:19454;
        healthcheck type=https interval=1s timeout=1s;
        healthcheck_tls_name backend.local;
    }
}
EOF
cat >"$FIXTURE_DIR/stream-missing-name.conf" <<EOF
events {}
stream {
    upstream bad_stream {
        zone bad_stream 64k;
        server 127.0.0.1:19444;
        healthcheck_tcp type=tls interval=1s timeout=1s;
        healthcheck_trusted_certificate $FIXTURE_DIR/ca.pem;
    }
}
EOF
cat >"$FIXTURE_DIR/stream-missing-ca.conf" <<EOF
events {}
stream {
    upstream bad_stream {
        zone bad_stream 64k;
        server 127.0.0.1:19444;
        healthcheck_tcp type=tls interval=1s timeout=1s;
        healthcheck_tls_name backend.local;
    }
}
EOF

assert_config_rejected "$FIXTURE_DIR/http-missing-name.conf" \
  'HTTPS healthcheck requires healthcheck_tls_name and healthcheck_trusted_certificate'
assert_config_rejected "$FIXTURE_DIR/http-missing-ca.conf" \
  'HTTPS healthcheck requires healthcheck_tls_name and healthcheck_trusted_certificate'
assert_config_rejected "$FIXTURE_DIR/stream-missing-name.conf" \
  'TLS healthcheck requires healthcheck_tls_name and healthcheck_trusted_certificate'
assert_config_rejected "$FIXTURE_DIR/stream-missing-ca.conf" \
  'TLS healthcheck requires healthcheck_tls_name and healthcheck_trusted_certificate'

"$NGINX_BIN" -t -p "$FIXTURE_DIR" -c "$FIXTURE_DIR/nginx.conf"
start_fixture "$NGINX_BIN" -p "$FIXTURE_DIR" -c "$FIXTURE_DIR/nginx.conf" -g 'daemon off;'

wait_for_result nginx_healthcheck_check_result dynamic_http 127.0.0.2:19080 L7OK
wait_for_result nginx_healthcheck_check_result dynamic_http 127.0.0.3:19080 L7STS
wait_for_result nginx_healthcheck_check_result l4_http 127.0.0.2:19080 L4OK
wait_for_result nginx_healthcheck_check_result l4_http_failure 127.0.0.1:19999 L4CON
wait_for_result nginx_healthcheck_check_result ipv6_http '[::1]:19084' L4OK
wait_for_result nginx_healthcheck_check_result tls_only 127.0.0.1:19444 L6OK
wait_for_result nginx_healthcheck_check_result tls_wrong_name 127.0.0.1:19444 L6RSP
wait_for_result nginx_healthcheck_check_result tls_wrong_ca 127.0.0.1:19444 L6RSP
wait_for_result nginx_healthcheck_check_result tls_plaintext 127.0.0.2:19080 L6RSP
wait_for_result nginx_healthcheck_check_result tls_stall 127.0.0.1:19443 L6TOUT
wait_for_result nginx_healthcheck_check_result tls_expired 127.0.0.1:19448 L6RSP
wait_for_result nginx_healthcheck_check_result tls_future 127.0.0.1:19447 L6RSP
wait_for_result nginx_healthcheck_check_result https_success 127.0.0.1:19454 L7OK
wait_for_result nginx_healthcheck_check_result https_status 127.0.0.1:19454 L7STS
wait_for_result nginx_healthcheck_check_result https_timeout 127.0.0.1:19454 L7TOUT
wait_for_result nginx_healthcheck_check_result https_malformed 127.0.0.1:19455 L7RSP
wait_for_result nginx_stream_healthcheck_check_result dynamic_tcp 127.0.0.2:19082 L7OK
wait_for_result nginx_stream_healthcheck_check_result dynamic_tcp 127.0.0.3:19082 L7RSP
wait_for_result nginx_stream_healthcheck_check_result l4_stream 127.0.0.2:19082 L4OK
wait_for_result nginx_stream_healthcheck_check_result l4_stream_failure 127.0.0.1:19999 L4CON
wait_for_result nginx_stream_healthcheck_check_result tls_stream 127.0.0.1:19444 L6OK
wait_for_result nginx_stream_healthcheck_check_result stream_wrong_name 127.0.0.1:19444 L6RSP
wait_for_result nginx_stream_healthcheck_check_result stream_wrong_ca 127.0.0.1:19444 L6RSP
wait_for_result nginx_stream_healthcheck_check_result stream_plaintext 127.0.0.2:19080 L6RSP
wait_for_result nginx_stream_healthcheck_check_result stream_stall 127.0.0.1:19443 L6TOUT
wait_for_result nginx_stream_healthcheck_check_result stream_expired 127.0.0.1:19448 L6RSP
wait_for_result nginx_stream_healthcheck_check_result stream_future 127.0.0.1:19447 L6RSP
wait_for_result nginx_stream_healthcheck_check_result stream_tls_payload 127.0.0.1:19446 L7OK
wait_for_readiness nginx_healthcheck_peer_up dynamic_http 127.0.0.2:19080 1
wait_for_readiness nginx_stream_healthcheck_peer_up dynamic_tcp 127.0.0.2:19082 1
wait_for_readiness nginx_healthcheck_peer_up ipv6_http '[::1]:19084' 1

if fetch_metrics && [ "$(metric_value nginx_healthcheck_check_status tls_only 127.0.0.1:19444)" = 1 ] &&
  [ "$(metric_value nginx_healthcheck_check_code tls_only 127.0.0.1:19444)" = 0 ]; then
  :
else
  printf 'TLS-only HTTP check did not preserve boolean status and empty HTTP code.\n' >&2
  exit 1
fi
if fetch_metrics && [ "$(metric_value nginx_healthcheck_check_status https_success 127.0.0.1:19454)" = 1 ] &&
  [ "$(metric_value nginx_healthcheck_check_code https_success 127.0.0.1:19454)" = 200 ] &&
  [ "$(metric_value nginx_healthcheck_check_status https_status 127.0.0.1:19454)" = 0 ] &&
  [ "$(metric_value nginx_healthcheck_check_code https_status 127.0.0.1:19454)" = 503 ]; then
  :
else
  printf 'HTTPS checks did not preserve the boolean status and parsed HTTP code gauges.\n' >&2
  exit 1
fi
if [ ! -s "$FIXTURE_DIR/tls-application.log" ] || grep -Ev '^0$' "$FIXTURE_DIR/tls-application.log"; then
  printf 'TLS-only stream probes sent application bytes.\n' >&2
  exit 1
fi
if [ "$(wc -l <"$FIXTURE_DIR/tls-application.log")" -lt 2 ]; then
  printf 'The HTTP and stream TLS-only checks were not both observed by the TLS fixture.\n' >&2
  exit 1
fi

if ! curl -fsS --max-time 2 http://127.0.0.1:19081/route | grep -Fx 127.0.0.2; then
  printf 'HTTP upstream did not route through the healthy resolved peer.\n' >&2
  exit 1
fi
assert_stream_response 127.0.0.1 19083 '127.0.0.2 PONG'
assert_stream_response ::1 19084 'IPv6 ready'
if ! curl -fsS --max-time 2 http://127.0.0.1:19081/traffic | grep '^nginx_vts_'; then
  printf 'VTS metrics endpoint did not export a metric.\n' >&2
  exit 1
fi
assert_multiworker_metrics
wait_for_readiness nginx_healthcheck_peer_up dynamic_http 127.0.0.3:19080 0
wait_for_readiness nginx_stream_healthcheck_peer_up dynamic_tcp 127.0.0.3:19082 0
printf '127.0.0.4 backend.local\n127.0.0.3 backend.local\n' >"$FIXTURE_DIR/hosts"
dnsmasq_pid=${FIXTURE_PIDS[0]}
kill -HUP "$dnsmasq_pid"
wait_for_result nginx_healthcheck_check_result dynamic_http 127.0.0.4:19080 L7OK
wait_for_result nginx_stream_healthcheck_check_result dynamic_tcp 127.0.0.4:19082 L7OK
wait_for_readiness nginx_healthcheck_peer_up dynamic_http 127.0.0.4:19080 1
wait_for_readiness nginx_stream_healthcheck_peer_up dynamic_tcp 127.0.0.4:19082 1
if ! curl -fsS --max-time 2 http://127.0.0.1:19081/route | grep -Fx 127.0.0.4; then
  printf 'HTTP upstream did not route through the replacement DNS peer.\n' >&2
  exit 1
fi
assert_stream_response 127.0.0.1 19083 '127.0.0.4 PONG'
if fetch_metrics && grep -F 'upstream="dynamic_http"' "$METRICS_FILE" | grep -F 'peer="127.0.0.2:19080"'; then
  printf 'Removed HTTP DNS peer remained in health metrics.\n' >&2
  exit 1
fi
if fetch_metrics && grep -F 'upstream="dynamic_tcp"' "$METRICS_FILE" | grep -F 'peer="127.0.0.2:19082"'; then
  printf 'Removed stream DNS peer remained in health metrics.\n' >&2
  exit 1
fi
touch "$FIXTURE_DIR/recovered-http"
wait_for_readiness nginx_healthcheck_peer_up dynamic_http 127.0.0.3:19080 1
fetch_metrics
grep -F 'upstream="dynamic_http"' "$METRICS_FILE" | grep -F 'server="127.0.0.1:19999"' | grep -F 'peer="127.0.0.1:19999"' | grep -F 'backup="true"'
grep -F 'upstream="dynamic_tcp"' "$METRICS_FILE" | grep -F 'server="127.0.0.1:19999"' | grep -F 'peer="127.0.0.1:19999"' | grep -F 'backup="true"'

start_fixture openssl s_server -accept 127.0.0.1:19445 \
  -cert "$FIXTURE_DIR/wrong-ca.pem" -key "$FIXTURE_DIR/wrong-ca.key" \
  -quiet -ign_eof >"$FIXTURE_DIR/recovery-server.log" 2>&1
wait_for_result nginx_healthcheck_check_result tls_recovery 127.0.0.1:19445 L6RSP
wait_for_result nginx_stream_healthcheck_check_result stream_tls_recovery 127.0.0.1:19445 L6RSP
wait_for_readiness nginx_healthcheck_peer_up tls_recovery 127.0.0.1:19445 0
wait_for_readiness nginx_stream_healthcheck_peer_up stream_tls_recovery 127.0.0.1:19445 0
recovery_pid=${FIXTURE_PIDS[${#FIXTURE_PIDS[@]} - 1]}
stop_fixture "$recovery_pid"
start_fixture openssl s_server -accept 127.0.0.1:19445 \
  -cert "$FIXTURE_DIR/valid.pem" -key "$FIXTURE_DIR/valid.key" \
  -quiet -ign_eof >"$FIXTURE_DIR/recovery-server.log" 2>&1
wait_for_readiness nginx_healthcheck_peer_up tls_recovery 127.0.0.1:19445 1
wait_for_readiness nginx_stream_healthcheck_peer_up stream_tls_recovery 127.0.0.1:19445 1

printf 'L4/L6/L7 classification, DNS replacement, recovery, IPv6, multi-worker visibility, and VTS checks passed.\n'
cleanup

#!/bin/sh
set -eu
server_pid=

# Stops and reaps the bounded scratch-root nginx supervisor.
# @return The status that triggered cleanup.
# @return-type int
cleanup() {
  status=$?
  trap - EXIT INT TERM
  if [ -n "$server_pid" ]; then
    kill -QUIT "$server_pid" || true
    wait "$server_pid" || true
  fi
  exit "$status"
}

trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

chroot /runtime /sbin/nginx -p / -t
openssl req -x509 -newkey rsa:2048 -nodes -days 1 -subj /CN=healthcheck-build-ca \
  -keyout /tmp/healthcheck-build-ca.key -out /runtime/conf/backend-ca.pem
printf '127.0.0.1 maintenance.internal\n' >/runtime/etc/hosts
chroot /runtime /sbin/nginx -p / -t -c conf/upstream-healthcheck.conf
rm -f /tmp/healthcheck-build-ca.key /runtime/conf/backend-ca.pem /runtime/etc/hosts
timeout --signal=QUIT --kill-after=2s 20s chroot /runtime /sbin/nginx -p / -g 'daemon off;' &
server_pid=$!
timeout --signal=TERM --kill-after=1s 15s curl --retry 10 --retry-connrefused --retry-delay 1 --retry-max-time 10 --max-time 2 -fsS http://127.0.0.1/ >/dev/null
kill -0 "$(cat /runtime/logs/nginx.pid)"
kill -QUIT "$server_pid"
wait "$server_pid"
server_pid=
rm -f /runtime/logs/access.log /runtime/logs/error.log

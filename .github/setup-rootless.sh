#!/usr/bin/env sh
set -eu

if [ "$(id -u)" -eq 0 ]; then
  printf 'Error: Run rootless provisioning as the non-root build user.\n' >&2
  exit 1
fi
sudo apt-get update
sudo apt-get install --yes --no-install-recommends rootlesskit runc uidmap
for prerequisite in rootlesskit runc newuidmap newgidmap setsid timeout; do
  command -v "$prerequisite"
done
user=$(id -un)
uid=$(id -u)
for mapping in /etc/subuid /etc/subgid; do
  if ! awk -F: -v user="$user" -v uid="$uid" '($1 == user || $1 == uid) && $2 > 0 && $3 >= 65536 { found = 1 } END { exit !found }' "$mapping"; then
    printf 'Error: Provision a subordinate ID range of at least 65536 for %s in %s.\n' "$user" "$mapping" >&2
    exit 1
  fi
done
mapping_program="
  FNR == 1 { root = 0; subordinate = 0 }
  \$1 == 0 && \$3 == 1 { root = 1 }
  \$1 > 0 && \$3 >= 65536 { subordinate = 1 }
  FNR == 2 && root && subordinate { valid++ }
  END { exit valid != 2 }
"
if ! timeout --kill-after=5 30 rootlesskit --pidns --reaper=true awk "$mapping_program" /proc/self/uid_map /proc/self/gid_map; then
  printf 'Error: Rootless namespace preflight failed; provision mappings and permitted user namespaces without weakening AppArmor, seccomp, or host policy.\n' >&2
  exit 1
fi

#!/usr/bin/env sh
set -eu
if [ "$#" -gt 3 ]; then
  printf 'Usage: %s [<target> [package-version] [release-tag]]\n' "$0" >&2
  exit 1
fi
ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
target="${1:-}"
package_version="${2:-}"
release_tag="${3:-}"
if [ -z "$target" ]; then
  targets="$(sh "${ROOT_DIR}/scripts/metadata.sh" list-targets)"
  printf '%s\n' 'include:'
  for t in $targets; do
    pv="$(sh "${ROOT_DIR}/scripts/metadata.sh" get-official-version "$t")"
    printf '%s\n' '  - component: $CI_SERVER_FQDN/$CI_PROJECT_PATH/static-release@$CI_COMMIT_SHA'
    printf '%s\n' '    inputs:'
    printf '%s\n' '      stage: release'
    printf '      target: %s\n' "$t"
    printf '      package_name: %s\n' "${t}-${pv}"
    printf '%s\n' '      run_policy: manual'
  done
  exit 0
fi
sh "${ROOT_DIR}/scripts/metadata.sh" get-tag-prefix "$target" >/dev/null
if [ -z "$package_version" ]; then
  package_version="$(sh "${ROOT_DIR}/scripts/metadata.sh" get-official-version "$target")"
fi
package_name="${target}-${package_version}"
if [ -n "$release_tag" ]; then
  printf '%s\n' 'stages:'
  printf '%s\n' '  - build'
  printf '%s\n' '  - publish'
  printf '%s\n' ''
fi
printf '%s\n' 'include:'
printf '%s\n' '  - component: $CI_SERVER_FQDN/$CI_PROJECT_PATH/static-release@$CI_COMMIT_SHA'
printf '%s\n' '    inputs:'
if [ -n "$release_tag" ]; then
  printf '%s\n' '      stage: build'
else
  printf '%s\n' '      stage: release'
fi
printf '      target: %s\n' "$target"
printf '      package_name: %s\n' "$package_name"
if [ -n "$release_tag" ]; then
  printf '%s\n' ''
  printf '%s\n' 'publish-release:'
  printf '%s\n' '  stage: publish'
  printf '%s\n' '  image: registry.gitlab.com/gitlab-org/release-cli:latest'
  printf '%s\n' '  needs:'
  printf '    - job: gitlab-static-package-%s\n' "$target"
  printf '%s\n' '      artifacts: true'
  printf '%s\n' '  before_script:'
  printf '%s\n' '    - apk add --no-cache curl'
  printf '%s\n' '  script:'
  printf '%s\n' '    - |'
  printf '      curl --header "JOB-TOKEN: $CI_JOB_TOKEN" \\\n'
  printf '        --upload-file "%s.tar.gz" \\\n' "$package_name"
  printf '        "$CI_API_V4_URL/projects/$CI_PROJECT_ID/packages/generic/%s/%s/%s.tar.gz"\n' "$target" "$release_tag" "$release_tag"
  printf '%s\n' '  release:'
  printf '    tag_name: %s\n' "$release_tag"
  printf '    name: %s\n' "$release_tag"
  printf '    description: "Automated static binary release from tag %s."\n' "$release_tag"
  printf '%s\n' '    assets:'
  printf '%s\n' '      links:'
  printf '        - name: %s.tar.gz\n' "$release_tag"
  printf '          url: $CI_API_V4_URL/projects/$CI_PROJECT_ID/packages/generic/%s/%s/%s.tar.gz\n' "$target" "$release_tag" "$release_tag"
  printf '%s\n' '          link_type: package'
fi

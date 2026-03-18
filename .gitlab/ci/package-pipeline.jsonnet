local metadata = import '../../metadata.json';
local branch = std.extVar('branch');
local targetFilter = std.extVar('target');

local COMPONENT_REF = '$CI_SERVER_FQDN/$CI_PROJECT_PATH/static-release@$CI_COMMIT_SHA';

local versionSuffix =
  if std.length(branch) > 0 && std.startsWith(branch, 'feature/')
  then '-beta'
  else '';

local officialVersion(target) =
  metadata[target].env[metadata[target].version_env_var];

local packageName(target) =
  '%s-%s%s' % [target, officialVersion(target), versionSuffix];

local targets =
  if targetFilter != ''
  then [targetFilter]
  else std.objectFields(metadata);

local allTargetsMode = targetFilter == '';

local includes = [
  {
    component: COMPONENT_REF,
    inputs:
      { stage: 'build', target: t, package_name: packageName(t) }
      + (if allTargetsMode then { run_policy: 'manual' } else {}),
  }
  for t in targets
];

local publishJobs = {
  ['publish-' + t]: {
    stage: 'publish',
    image: 'curlimages/curl',
    needs: [{ job: 'gitlab-static-package-' + t, artifacts: true }],
    script: [
      |||
        curl --header "JOB-TOKEN: $CI_JOB_TOKEN" \
          --upload-file "%(pn)s.tar.gz" \
          "$CI_API_V4_URL/projects/$CI_PROJECT_ID/packages/generic/%(t)s/%(pn)s/%(pn)s.tar.gz"
      ||| % { pn: packageName(t), t: t },
    ],
  }
  for t in targets
};

std.manifestYamlDoc(
  { stages: ['build', 'publish'], include: includes } + publishJobs,
  indent_array_in_object=true,
  quote_keys=false,
)

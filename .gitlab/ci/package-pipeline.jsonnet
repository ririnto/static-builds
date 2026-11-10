local metadata = import '../../metadata.json';
local branch = std.extVar('branch');
local targetFilter = std.extVar('target');

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

local includes = [
  {
    component: std.extVar('component_ref'),
    inputs:
      { stage: 'build', target: t, package_name: packageName(t) },
  }
  for t in targets
];

std.manifestYamlDoc(
  { stages: ['build'], include: includes },
  indent_array_in_object=true,
  quote_keys=false,
)

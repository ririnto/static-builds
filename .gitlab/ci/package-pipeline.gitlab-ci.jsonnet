function(release_tag='')
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
    if release_tag != '' then release_tag
    else '%s-%s%s' % [target, officialVersion(target), versionSuffix];

  local targets =
    if targetFilter != ''
    then [targetFilter]
    else std.objectFields(metadata);

  local includes = [
    {
      component: std.extVar('component_ref'),
      inputs:
        { stage: 'build', target: t, package_name: packageName(t), release_tag: release_tag },
    }
    for t in targets
  ];

  assert release_tag == '' || targetFilter != '' : 'release tags require one target';
  std.manifestYamlDoc(
    { stages: ['build', 'release'], include: includes },
    indent_array_in_object=true,
    quote_keys=false,
  )

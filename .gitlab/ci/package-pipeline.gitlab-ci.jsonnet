function(release_tag='')
  local metadata = import '../../metadata.json';
  local branch = std.extVar('branch');
  local targetFilter = std.extVar('target');

  local versionSuffix =
    if std.length(branch) > 0 && std.startsWith(branch, 'feature/')
    then '-beta'
    else '';

  local targets =
    if targetFilter != ''
    then [targetFilter]
    else std.objectFields(metadata);

  assert release_tag == '' || targetFilter != '' : 'release tags require one target';
  std.manifestYamlDoc(
    { stages: ['build', 'release'], include: [
      {
        component: std.extVar('component_ref'),
        inputs:
          { stage: 'build', target: t, package_name: if release_tag != '' then release_tag else '%s-%s%s' % [t, metadata[t].env[metadata[t].version_env_var], versionSuffix], release_tag: release_tag },
      }
      for t in targets
    ] },
    indent_array_in_object=true,
    quote_keys=false,
  )

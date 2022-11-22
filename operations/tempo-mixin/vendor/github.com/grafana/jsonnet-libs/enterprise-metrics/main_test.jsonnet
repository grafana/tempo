local l = (import 'main.libsonnet') {
  _config+:: {
    commonArgs+:: {
      'admin.client.backend-type': 'test',
      'blocks-storage.backend': 'test',
      'cluster-name': 'test',
    },
  },
};

{
  // allKubernetesObjects returns all objects with 'obj' that have a 'apiVersion' and 'kind' fields and therefore are probably Kubernetes Objects.
  local allKubernetesObjects(obj) =
    if std.isObject(obj) then
      if std.objectHas(obj, 'apiVersion') && std.objectHas(obj, 'kind') then
        [obj]
      else
        std.flatMap(function(field) allKubernetesObjects(obj[field]), std.objectFields(obj))
    else
      [],

  // Ensure that no resources have a configured namespace.
  testNoNamespace:
    local assertNoNamespace(obj) =
      assert !std.objectHas(obj.metadata, 'namespace') : 'objects should not have the namespace field configured, obj=%s' % std.toString(obj); true;
    std.foldr(function(obj, acc) acc && assertNoNamespace(obj), allKubernetesObjects(l), true),

  // Ensure no container has arguments that contain the string 'namespace'.
  testNoNamespaceInArgs:
    local modules = std.flatMap(
      function(field)
        if std.objectHasAll(l[field], 'container') && std.objectHas(l[field].container, 'args') then
          [{ name: field, args: l[field].container.args }]
        else [],
      std.objectFields(l),
    );
    assert std.length(modules) > 0 : 'expected at least one module';
    local assertNoNamespaceInArgs(module) =
      std.foldr(function(arg, acc)
                  acc && assert std.length(std.findSubstr('namespace', arg)) == 0 :
                                'container arguments should not contain the string namespace, module=%s, arg=%s' % [module.name, arg]; true,
                module.args,
                true);
    std.foldr(function(module, acc) acc && assertNoNamespaceInArgs(module), modules, true),

  testNoVisibleDocsonnetFields:
    local isDocsonnetField(field) = std.startsWith(field, '#');
    local docsonnetFields(obj) = std.filter(isDocsonnetField, std.objectFields(obj))
                                 + std.flatMap(function(field) docsonnetFields(obj[field]),
                                               std.filter(function(field) std.isObject(obj[field]), std.objectFieldsAll(obj)));
    local fields = docsonnetFields(l);
    assert std.length(fields) == 0 : 'docsonnet fields should not be visible, fields=%s' % std.toString(fields); true,
}

{
  grizzlyAPI:: 'grizzly.grafana.com/v1alpha1',
  get(obj, key, default):: if std.objectHasAll(obj, key) then obj[key] else default,

  makeResource(kind, name, resource, metadata={}):: {
    apiVersion: $.grizzlyAPI,
    kind: kind,
    metadata: {
      name: name,
    } + metadata,
    spec: resource,
  },
}

{
  defaultApiVersion:: 'grizzly.grafana.com/v1alpha1',
  new(kind, name):: {
    apiVersion: $.defaultApiVersion,
    kind: kind,
    metadata: {
      name: name,
    },
  },
  withApiVersion(apiVersion):: {
    defaultApiVersion:: apiVersion,
    apiVersion: apiVersion,
  },
  addMetadata(name, value):: {
    metadata+: {
      [name]: value,
    },
  },
  withSpec(spec):: {
    spec: spec,
  },
}

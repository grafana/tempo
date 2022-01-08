local d = import 'github.com/jsonnet-libs/docsonnet/doc-util/main.libsonnet';
{
  local this = self,

  '#new':
    d.fn(
      |||
        `new` initiates an [inline Tanka environment](https://tanka.dev/inline-environments#inline-environments)
      |||,
      [
        d.arg('name', d.T.string),
        d.arg('namespace', d.T.string),
        d.arg('apiserver', d.T.string),
      ]
    ),
  new(name, namespace, apiserver)::
    {
      apiVersion: 'tanka.dev/v1alpha1',
      kind: 'Environment',
    }
    + this.withName(name)
    + this.withNamespace(namespace)
    + this.withApiServer(apiserver)
  ,

  '#withData'::
    d.fn(
      '`withData` adds the actual Kubernetes resources to the inline environment.',
      [d.arg('data', d.T.string)]
    ),
  withData(data):: {
    data: data,
  },

  '#withDataMixin'::
    d.fn(
      |||
        `withDataMixin` adds the actual Kubernetes resources to the inline environment.
        *Note:* This function appends passed data to existing values
      |||,
      [d.arg('data', d.T.string)]
    ),
  withDataMixin(data):: {
    data+: data,
  },


  '#withName'::
    d.fn(
      '`withName` sets the environment `name`.',
      [d.arg('name', d.T.string)]
    ),
  withName(name):: {
    metadata+: {
      name: name,
    },
  },

  '#withApiServer'::
    d.fn(
      |||
        `withApiServer` sets the Kubernetes cluster this environment should apply to.
        Must be the full URL, e.g. https://cluster.fqdn:6443
      |||,
      [d.arg('apiserver', d.T.string)]
    ),
  withApiServer(apiserver):: {
    spec+: {
      apiServer: apiserver,
    },
  },

  '#withNamespace'::
    d.fn(
      "`withNamespace` sets the default namespace for objects that don't explicitely specify one.",
      [d.arg('namespace', d.T.string)]
    ),
  withNamespace(namespace):: {
    spec+: {
      namespace: namespace,
    },
  },

  '#withLabels'::
    d.fn(
      '`withLabels` adds arbitrary key:value labels.',
      [d.arg('labels', d.T.string)]
    ),
  withLabels(labels):: {
    metadata+: {
      labels:
        std.mapWithKey(
          function(k, v)
            if std.isString(v)
            then v
            else error '%s has non-string value' % k,
          labels
        ),
    },
  },

  '#withLabelsMixin'::
    d.fn(
      |||
        `withLabelsMixin` adds arbitrary key:value labels.
        *Note:* This function appends passed data to existing values
      |||,
      [d.arg('labels', d.T.string)]
    ),
  withLabelsMixin(labels):: {
    metadata+: {
      labels+:
        std.mapWithKey(
          function(k, v)
            if std.isString(v)
            then v
            else error '%s has non-string value' % k,
          labels
        ),
    },
  },

  '#withInjectLabels'::
    d.fn(
      |||
        `withInjectLabels` adds a "tanka.dev/environment" label to each created resource.
        Required for [garbage collection](https://tanka.dev/garbage-collection).
      |||,
      [d.arg('bool', d.T.string)]
    ),
  withInjectLabels(bool=true):: {
    spec+: {
      injectLabels: bool,
    },
  },

  '#withResourceDefaults'::
    d.fn(
      '`withResourceDefaults` sets defaults for all resources in this environment.',
      [d.arg('labels', d.T.string)]
    ),
  withResourceDefaults(defaults):: {
    spec+: {
      resourceDefaults: defaults,
    },
  },

  '#withResourceDefaultsMixin'::
    d.fn(
      |||
        `withResourceDefaultsMixin` sets defaults for all resources in this environment.
        *Note:* This function appends passed data to existing values
      |||,
      [d.arg('labels', d.T.string)]
    ),
  withResourceDefaultsMixin(defaults):: {
    spec+: {
      resourceDefaults+: defaults,
    },
  },
}

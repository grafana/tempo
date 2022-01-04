local clusters = import 'clusters.libsonnet';
local tanka = import 'github.com/grafana/jsonnet-libs/tanka-util/main.libsonnet';

{
  environment(cluster)::
    tanka.environment.new(
      name='grafana/' + cluster.name,
      namespace=cluster.namespace,
      apiserver=cluster.apiServer,
    )
    + tanka.environment.withLabels({ cluster: cluster.name })
    + tanka.environment.withData( cluster.data {

      _config+:: {
        namespace: cluster.namespace,
      },

    } + cluster.dataOverride)
    + {
      spec+: {
        injectLabels: true,
      },
    },

  envs: {
    [cluster.name]: $.environment(cluster)
    for cluster in clusters
  },
}

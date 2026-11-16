// Minimal stub of the ksonnet-util helpers that grafana/configmaps.libsonnet
// uses, so the sharding logic can be exercised in isolation without vendoring
// the full k8s-libsonnet (`k.libsonnet`) tree. Only the object/mount
// constructors touched by the dashboard-sharding code path are implemented.
{
  core+: { v1+: { configMap+: {
    new(name):: { kind: 'ConfigMap', metadata: { name: name }, data: {} },
    withData(d):: { data+: d },
    withDataMixin(d):: { data+: d },
    mixin+: { metadata+: { withLabels(l):: { metadata+: { labels: l } } } },
  } } },
  apps+: { v1+: { deployment+: { mixin+: { spec+: { template+: { metadata+: {
    withAnnotationsMixin(a):: { annotations+: a },
  } } } } } } },
  util+: {
    manifestYaml(o):: std.toString(o),
    volumeMountItem(n, p):: { name: n, path: p },
    configMapVolumeMountItem(cm, p):: { cm: cm.metadata.name, path: p },
    volumeMounts(m):: { mounts: m },
  },
}

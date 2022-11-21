local tanka = import 'github.com/grafana/jsonnet-libs/tanka-util/main.libsonnet';
local helm = tanka.helm.new(std.thisFile);
local kustomize = tanka.kustomize.new(std.thisFile);

{
  // render the Grafana Chart, set namespace to "test"
  grafana: helm.template('grafana', './charts/grafana', {
    values: {
      persistence: { enabled: true },
      plugins: ['grafana-clock-panel'],
    },
    namespace: 'test',
  }),

  // render the Prometheus Kustomize
  // then entrypoint for `kustomize build` will be ./base/prometheus/kustomization.yaml
  prometheus: kustomize.build('./base/prometheus'),
}

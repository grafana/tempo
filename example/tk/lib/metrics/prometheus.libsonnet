local ksm = import 'github.com/grafana/jsonnet-libs/kube-state-metrics/main.libsonnet';
local prometheus = import 'prometheus/prometheus.libsonnet';
local scrape_configs = import 'prometheus/scrape_configs.libsonnet';

local namespace = 'default';

prometheus {
  ksm: ksm.new(namespace),

  _config+:: {
    prometheus_external_hostname: 'http://prometheus',
  },

  prometheus_config+:: {
    scrape_configs: [
      scrape_configs.kubernetes_pods,
      scrape_configs.kube_dns,
      ksm.scrape_config(namespace),
    ],
  },
}

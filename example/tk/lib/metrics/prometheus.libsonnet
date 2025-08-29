local ksm = import 'kube-state-metrics/main.libsonnet';
local prometheus = import 'prometheus/prometheus.libsonnet';
local scrape_configs = import 'prometheus/scrape_configs.libsonnet';

prometheus {
  ksm: ksm.new($._config.namespace),

  _config+:: {
    prometheus_external_hostname: 'http://prometheus',
  },

  prometheus_config+:: {
    scrape_configs: [
      scrape_configs.kubernetes_pods {
        // add the label 'cluster' to every scraped metric
        relabel_configs+: [
          {
            source_labels: ['__address__'],  // always exists
            regex: '.*',  // always matches
            target_label: 'cluster',
            replacement: $._config.cluster,
            action: 'replace',
          },
        ],
      },
      scrape_configs.kube_dns,
      ksm.scrape_config($._config.namespace),
    ],
  },
}

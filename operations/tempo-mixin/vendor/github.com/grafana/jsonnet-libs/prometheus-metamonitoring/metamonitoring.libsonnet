local alertmanager_mixin = import 'alertmanager-mixin/mixin.libsonnet';
local k = import 'k.libsonnet';
local prometheus_mixin = import 'prometheus-mixin/mixin.libsonnet';
local prometheus = import 'prometheus/prometheus.libsonnet';

local relabel_configs = import 'relabel_configs.libsonnet';
local util = import 'util.libsonnet';

{
  _config+:: {
    name: 'prometheus-meta',

    namespace: error 'must provide namespace',
    cluster_name: error 'must provide cluster_name',

    // clusters are objects with an apiServer directive and optional
    // cluster_name and secret_name directives.
    // If not provided, cluster_name will be the key and secret_name will be the
    // cluster_name prefixed with 'metamonitoring-')
    // The secret must contain a token and ca.crt to connect to the
    // corresponding cluster.
    // clusters: {
    //   clusterA: {
    //     apiServer: 'https://1.2.3.4',
    //     cluster_name: 'clusterA',
    //     secret_name: 'metamonitoring-clusterA',
    //   },
    // },
    clusters: error 'must provide clusters',

    // alertmanagers are found by service discovery in the same we as we find
    // prometheis. By default we look for pods in the default namespace with
    // port 9093.
    alertmanager_clusters: {
      [c]: $._config.clusters[c] {
        namespace: 'default',
        path: '/alertmanager/',
        port: 9093,
      }
      for c in std.objectFields(self.clusters)
    },
  },

  prometheus:
    prometheus
    + prometheus.withHighAvailability()
    + {
      local this = self,

      local clusters = $._config.clusters,

      _config+:: $._config {
        clusters:
          // derive secret_name from key if not set
          std.foldr(
            function(k, m) m {
              [k]: clusters[k] {
                cluster_name: (
                  if std.objectHas(clusters[k], 'cluster_name')
                  then clusters[k].cluster_name
                  else k
                ),
                secret_name: (
                  if std.objectHas(clusters[k], 'secret_name')
                  then clusters[k].secret_name
                  else 'metamonitoring-%s' % self.cluster_name
                ),
              },
            },
            std.objectFields(clusters),
            {}
          ),
      },

      mixins+:: {
        prometheus:
          prometheus_mixin {
            _config+:: {
              prometheusSelector: 'job=~".*/.*prometheus.*"',
              prometheusHAGroupLabels: 'job,cluster,namespace',
              prometheusHAGroupName: '{{$labels.job}} in {{$labels.cluster}}',
            },
          },

        alertmanager:
          alertmanager_mixin
          {
            _config+:: {
              alertmanagerSelector: 'job="default/alertmanager"',
              alertmanagerClusterLabels: 'job, namespace',
              alertmanagerName: '{{$labels.instance}} in {{$labels.cluster}}',
            },
          },
      },

      prometheusAlerts+:: {
        groups+: [{
          name: 'metamonitoring',
          rules: [
            {
              alert: 'AlwaysFiringAlert',
              expr: 'vector(1)',
              'for': '1m',
              labels: {
                cluster: this._config.cluster_name,
                namespace: this._config.namespace,
              },
              annotations: {
                message: 'Always firing alert for Prometheus metamonitoring.',
              },
            },
          ],
        }],
      },

      prometheus_container+:
        k.core.v1.container.withVolumeMountsMixin([
          k.core.v1.volumeMount.new(this._config.clusters[c].secret_name, '/secrets/%s' % this._config.clusters[c].cluster_name)
          for c in std.objectFields(this._config.clusters)
        ]),
      prometheus_statefulset+:
        util.statefulSet.withMetamonitoring() +
        k.apps.v1.statefulSet.spec.template.spec.withVolumesMixin([
          k.core.v1.volume.fromSecret(this._config.clusters[c].secret_name,
                                      this._config.clusters[c].secret_name)
          for c in std.objectFields(this._config.clusters)
        ]),

      prometheus_config+:: {
        scrape_configs: [
          local cluster = this._config.clusters[c];
          this.service_discovery(cluster) {
            job_name: 'prometheus-%s' % cluster.cluster_name,

            relabel_configs:
              relabel_configs
              + [
                {
                  target_label: 'cluster',
                  replacement: cluster.cluster_name,
                },
              ],
          }
          for c in std.objectFields(this._config.clusters)
        ],

        alerting+: {
          alertmanagers: std.prune([
            local cluster = this._config.alertmanager_clusters[c];
            this.service_discovery(cluster) {
              api_version: 'v2',
              path_prefix: cluster.path,

              relabel_configs: [{
                source_labels: ['__meta_kubernetes_pod_label_name'],
                regex: 'alertmanager',
                action: 'keep',
              }, {
                source_labels: ['__meta_kubernetes_namespace'],
                regex: cluster.namespace,
                action: 'keep',
              }, {
                // This prevents port-less containers and the gossip ports
                // from showing up.
                source_labels: ['__meta_kubernetes_pod_container_port_number'],
                regex: cluster.port,
                action: 'keep',
              }],
            }
            for c in std.objectFields(this._config.alertmanager_clusters)
          ]),
        },
      },

      service_discovery(cluster):: {
        kubernetes_sd_configs: [
          {
            role: 'pod',
            api_server: cluster.apiServer,
            bearer_token_file: '/secrets/%s/token' % cluster.cluster_name,
            tls_config: {
              ca_file: '/secrets/%s/ca.crt' % cluster.cluster_name,
              insecure_skip_verify: false,
            },
          },
        ],

        bearer_token_file: '/secrets/%s/token' % cluster.cluster_name,
        tls_config: {
          ca_file: '/secrets/%s/ca.crt' % cluster.cluster_name,
          insecure_skip_verify: false,
        },
      },
    },
}

local kausal = import 'ksonnet-util/kausal.libsonnet';

function(replicas=2) {
  local this = self,
  local _config = self._config,
  local k = kausal {
    _config+:: _config,
  } + (
    // an attempt at providing compat with the original ksonnet-lib
    if std.objectHas(kausal, '__ksonnet')
    then
      {
        core+: { v1+: {
          envVar: kausal.core.v1.container.envType,
        } },
      }
    else {}
  ),

  _config+:: {
    prometheus_config_file: _config.prometheus_config_dir + '/$(POD_NAME).yml',
  },

  // The '__replica__' label is used by Cortex for deduplication.
  // We add a different one to each HA replica but remove it from
  // alerts to not break deduplication of alerts in the Alertmanager.
  prometheus_config+: {
    alerting+: {
      alert_relabel_configs+: [
        {
          regex: '__replica__',
          action: 'labeldrop',
        },
      ],
    },
  },

  local configMap = k.core.v1.configMap,
  prometheus_config_maps:
    [
      configMap.new('%s-config' % _config.name) +
      configMap.withData(
        std.foldr(
          function(i, acc)
            local name = _config.name + '-' + i;
            local config =
              this.prometheus_config {
                global+: {
                  external_labels+: {
                    __replica__: name,
                  },
                },
              };
            acc {
              [name + '.yml']: k.util.manifestYaml(config),
            },
          std.range(0, replicas - 1),

          {}
        ),
      ),
    ]
    + (
      if std.prune(this.prometheusAlerts) != {}
      then [
        configMap.new('%s-alerts' % _config.name) +
        configMap.withData({
          'alerts.rules': k.util.manifestYaml(this.prometheusAlerts),
        }),
      ]
      else []
    ) + (
      if std.prune(this.prometheusRules) != {}
      then [
        configMap.new('%s-recording' % _config.name) +
        configMap.withData({
          'recording.rules': k.util.manifestYaml(this.prometheusRules),
        }),
      ]
      else []
    ),

  local container = k.core.v1.container,
  local envVar = k.core.v1.envVar,

  prometheus_container+::
    container.withEnv([
      envVar.fromFieldPath('POD_NAME', 'metadata.name'),
    ])
    + container.mixin.readinessProbe.httpGet.withPath('%(prometheus_path)s-/ready' % self._config)
    + container.mixin.readinessProbe.httpGet.withPort(self._config.prometheus_port)
    + container.mixin.readinessProbe.withInitialDelaySeconds(15)
    + container.mixin.readinessProbe.withTimeoutSeconds(1)
  ,

  prometheus_watch_container+::
    container.withEnv([
      envVar.fromFieldPath('POD_NAME', 'metadata.name'),
    ]),

  local statefulset = k.apps.v1.statefulSet,

  prometheus_statefulset+:
    statefulset.mixin.spec.withReplicas(replicas)
    + k.util.antiAffinityStatefulSet,
}

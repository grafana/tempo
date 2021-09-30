{
  local k = import 'ksonnet-util/kausal.libsonnet',
  local configMap = k.core.v1.configMap,
  local container = k.core.v1.container,
  local volumeMount = k.core.v1.volumeMount,
  local deployment = k.apps.v1.deployment,
  local volume = k.core.v1.volume,
  local service = k.core.v1.service,
  local servicePort = service.mixin.spec.portsType,

  _config +:: {
    tempo_query_url: 'http://tempo:3200',
  },

  grafana_configmap:
    configMap.new('grafana-datasources') +
    configMap.withData({
      'datasource.yaml': k.util.manifestYaml({
        apiVersion: 1,
        datasources: [
          {
            name: 'Tempo',
            type: 'tempo',
            access: 'proxy',
            orgId: 1,
            url: $._config.tempo_query_url,
            basicAuth: false,
            isDefault: true,
            version: 1,
            editable: false,
            apiVersion: 1,
            uid: 'tempo',
          },
        ],
      }),
    }),

  grafana_container::
    container.new('grafana', 'grafana/grafana:7.5.7') +
    container.withVolumeMounts([
      volumeMount.new('grafana-datasources', '/etc/grafana/provisioning/datasources'),
    ]) +
    container.withEnvMap({
      GF_AUTH_ANONYMOUS_ENABLED: 'true',
      GF_AUTH_ANONYMOUS_ORG_ROLE: 'Admin',
      GF_AUTH_DISABLE_LOGIN_FORM: 'true',
    }),

  grafana_deployment:
    deployment.new('grafana', 1, [ $.grafana_container ], { app: 'grafana' }) +
    deployment.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap('grafana-datasources', $.grafana_configmap.metadata.name),
    ]),

  grafana_service:
    k.util.serviceFor($.grafana_deployment)
    + service.mixin.spec.withPortsMixin([
      servicePort.newNamed(
         name='http',
         port=3000,
         targetPort=3000,
      ),
    ]),
}

(import 'ksonnet-util/kausal.libsonnet') +
(import 'configmap.libsonnet') +
(import 'config.libsonnet') + 
{
  local container = $.core.v1.container,
  local containerPort = $.core.v1.containerPort,
  local volumeMount = $.core.v1.volumeMount,
  local pvc = $.core.v1.persistentVolumeClaim,
  local deployment = $.apps.v1.deployment,
  local volume = $.core.v1.volume,
  local service = $.core.v1.service,
  local servicePort = service.mixin.spec.portsType,

  local frigg_config_volume = 'frigg-conf',
  local frigg_query_config_volume = 'frigg-query-conf',
  local frigg_data_volume = 'frigg-data',

  namespace:
    $.core.v1.namespace.new($._config.namespace),

  frigg_pvc:
    pvc.new() +
    pvc.mixin.spec.resources
    .withRequests({ storage: $._config.pvc_size }) +
    pvc.mixin.spec
    .withAccessModes(['ReadWriteOnce'])
    .withStorageClassName($._config.pvc_storage_class) +
    pvc.mixin.metadata
    .withLabels({ app: 'frigg' })
    .withNamespace($._config.namespace)
    .withName('frigg-pvc') +
    { kind: 'PersistentVolumeClaim', apiVersion: 'v1' },

  frigg_container::
    container.new('frigg', $._images.frigg) +
    container.withPorts([
      containerPort.new('prom-metrics', $._config.port),
    ]) +
    container.withArgs([
      '-config.file=/conf/tempo.yaml',
      '-mem-ballast-size-mbs=' + $._config.ballast_size_mbs,
    ]) +
    container.withVolumeMounts([
      volumeMount.new(frigg_config_volume, '/conf'),
      volumeMount.new(frigg_data_volume, '/var/tempo'),
    ]),

  frigg_query_container::
    container.new('frigg-query', $._images.frigg_query) +
    container.withPorts([
      containerPort.new('jaeger-ui', 16686),
    ]) +
    container.withArgs([
      '--query.base-path=' + $._config.jaeger_ui.base_path,
      '--grpc-storage-plugin.configuration-file=/conf/tempo-query.yaml',
    ]) +
    container.withVolumeMounts([
      volumeMount.new(frigg_query_config_volume, '/conf'),
    ]),

  frigg_deployment:
    deployment.new('frigg',
                   1,
                   [
                     $.frigg_container,
                     $.frigg_query_container,
                   ],
                   { app: 'frigg' }) +
    deployment.mixin.spec.template.metadata.withAnnotations({
      config_hash: std.md5(std.toString($.frigg_configmap)),
    }) +
    deployment.mixin.spec.strategy.rollingUpdate.withMaxSurge(0) +
    deployment.mixin.spec.strategy.rollingUpdate.withMaxUnavailable(1) +
    deployment.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap(frigg_query_config_volume, $.frigg_query_configmap.metadata.name),
      volume.fromConfigMap(frigg_config_volume, $.frigg_configmap.metadata.name),
      volume.fromPersistentVolumeClaim(frigg_data_volume, $.frigg_pvc.metadata.name),
    ]),

  frigg_service:
    $.util.serviceFor($.frigg_deployment)
    + service.mixin.spec.withPortsMixin([
      servicePort.withName('http').withPort(80).withTargetPort(16686),
    ]),
}

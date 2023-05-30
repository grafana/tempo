(import 'configmap.libsonnet') +
(import 'config.libsonnet') +
{
  local k = import 'ksonnet-util/kausal.libsonnet',
  local container = k.core.v1.container,
  local containerPort = k.core.v1.containerPort,
  local volumeMount = k.core.v1.volumeMount,
  local pvc = k.core.v1.persistentVolumeClaim,
  local statefulset = k.apps.v1.statefulSet,
  local volume = k.core.v1.volume,
  local service = k.core.v1.service,
  local servicePort = service.mixin.spec.portsType,

  local tempo_config_volume = 'tempo-conf',
  local tempo_query_config_volume = 'tempo-query-conf',
  local tempo_data_volume = 'tempo-data',

  namespace:
    k.core.v1.namespace.new($._config.namespace),

  tempo_pvc::
    pvc.new() +
    pvc.mixin.spec.resources
    .withRequests({ storage: $._config.pvc_size }) +
    pvc.mixin.spec
    .withAccessModes(['ReadWriteOnce']) +
    pvc.mixin.spec
    .withStorageClassName($._config.pvc_storage_class) +
    pvc.mixin.metadata
    .withLabels({ app: 'tempo' }) +
    pvc.mixin.metadata
    .withNamespace($._config.namespace) +
    pvc.mixin.metadata
    .withName(tempo_data_volume) +
    { kind: 'PersistentVolumeClaim', apiVersion: 'v1' },

  tempo_container::
    container.new('tempo', $._images.tempo) +
    container.withPorts([
      containerPort.new('prom-metrics', $._config.tempo.port),
      containerPort.new('memberlist', 9095),
      containerPort.new('otlp', 4317),
    ]) +
    container.withArgs([
      '-target=scalable-single-binary',
      '-config.file=/conf/tempo.yaml',
      '-mem-ballast-size-mbs=' + $._config.ballast_size_mbs,
    ]) +
    container.withVolumeMounts([
      volumeMount.new(tempo_config_volume, '/conf'),
      volumeMount.new(tempo_data_volume, '/var/tempo'),
    ]) +
    k.util.resourcesRequests('3', '3Gi') +
    k.util.resourcesLimits('5', '5Gi'),

  tempo_query_container::
    container.new('tempo-query', $._images.tempo_query) +
    container.withPorts([
      containerPort.new('jaeger-ui', 16686),
      containerPort.new('jaeger-metrics', 16687),
    ]) +
    container.withArgs([
      '--query.base-path=' + $._config.jaeger_ui.base_path,
      '--grpc-storage-plugin.configuration-file=/conf/tempo-query.yaml',
    ]) +
    container.withVolumeMounts([
      volumeMount.new(tempo_query_config_volume, '/conf'),
    ]),

  tempo_statefulset:
    statefulset.new('tempo',
                    $._config.tempo.replicas,
                    std.prune([
                      $.tempo_container,
                      if $._config.tempo_query.enabled then $.tempo_query_container,
                    ]),
                    self.tempo_pvc,
                    { app: 'tempo' }) +
    statefulset.mixin.spec.withServiceName('tempo') +
    statefulset.mixin.spec.template.metadata.withAnnotations({
      config_hash: std.md5(std.toString($.tempo_configmap.data['tempo.yaml'])),
    }) +
    statefulset.mixin.metadata.withLabels({ app: $._config.tempo.headless_service_name, name: 'tempo' }) +
    statefulset.mixin.spec.selector.withMatchLabels({ name: 'tempo' }) +
    statefulset.mixin.spec.template.metadata.withLabels({ name: 'tempo', app: $._config.tempo.headless_service_name }) +
    statefulset.mixin.spec.template.spec.withVolumes(std.prune([
      if $._config.tempo_query.enabled then volume.fromConfigMap(tempo_query_config_volume, $.tempo_query_configmap.metadata.name),
      volume.fromConfigMap(tempo_config_volume, $.tempo_configmap.metadata.name),
    ])),

  tempo_service:
    k.util.serviceFor($.tempo_statefulset),

  tempo_headless_service:
    service.new(
      $._config.tempo.headless_service_name,
      { app: $._config.tempo.headless_service_name },
      []
    ) +
    service.mixin.spec.withClusterIP('None') +
    service.mixin.spec.withPublishNotReadyAddresses(true),
}

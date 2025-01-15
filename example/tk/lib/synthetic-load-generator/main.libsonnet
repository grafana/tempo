{
  local k = import 'ksonnet-util/kausal.libsonnet',
  local configMap = k.core.v1.configMap,
  local container = k.core.v1.container,
  local volumeMount = k.core.v1.volumeMount,
  local deployment = k.apps.v1.deployment,
  local volume = k.core.v1.volume,

  synthetic_load_generator_configmap:
    configMap.new('synthetic-load-generator') +
    configMap.withData({
      'load-generator.json': importstr './load-generator.json',
    }),

  synthetic_load_generator_container::
    container.new('synthetic-load-gen', 'omnition/synthetic-load-generator:1.0.25') +
    container.withVolumeMounts([
      volumeMount.new('conf', '/conf'),
    ]) +
    container.withEnvMap({
      TOPOLOGY_FILE: '/conf/load-generator.json',
      JAEGER_COLLECTOR_URL: 'http://tempo:14268',
    }),

  synthetic_load_generator_deployment:
    deployment.new('synthetic-load-generator',
                   1,
                   [ $.synthetic_load_generator_container ],
                   { app: 'synthetic_load_generator' }) +
    deployment.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap('conf', $.synthetic_load_generator_configmap.metadata.name),
    ]),
}

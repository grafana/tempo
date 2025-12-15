{
  local k = import 'ksonnet-util/kausal.libsonnet',
  local configMap = k.core.v1.configMap,
  local container = k.core.v1.container,
  local containerPort = k.core.v1.containerPort,
  local deployment = k.apps.v1.deployment,
  local volumeMount = k.core.v1.volumeMount,
  local volume = k.core.v1.volume,

  seaweedfs_config_map::
    configMap.new('seaweedfs-s3-config') +
    configMap.withData({
      's3.json': std.manifestJsonEx({
        identities: [{
          name: 'tempo',
          credentials: [{
            accessKey: 'tempo',
            secretKey: 'supersecret',
          }],
          actions: ['Admin', 'Read', 'Write', 'List', 'Tagging'],
        }],
      }, '  '),
    }),

  seaweedfs_container::
    container.new('seaweedfs', 'chrislusf/seaweedfs:latest') +
    container.withPorts([
      containerPort.new('s3', 9000),
      containerPort.new('master', 9333),
    ]) +
    container.withCommand([
      'weed',
      'server',
      '-s3',
      '-s3.port=9000',
      '-s3.config=/etc/seaweedfs/s3.json',
      '-dir=/data',
    ]) +
    container.withVolumeMounts([
      volumeMount.new('s3-config', '/etc/seaweedfs', readOnly=true),
    ]),

  seaweedfs_deployment:
    deployment.new('seaweedfs',
                   1,
                   [$.seaweedfs_container],
                   { app: 'seaweedfs' }) +
    deployment.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap('s3-config', 'seaweedfs-s3-config'),
    ]),

  seaweedfs_service:
    k.util.serviceFor($.seaweedfs_deployment),
}

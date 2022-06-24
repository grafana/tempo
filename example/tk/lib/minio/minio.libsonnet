{
  local k = import 'ksonnet-util/kausal.libsonnet',
  local configMap = k.core.v1.configMap,
  local container = k.core.v1.container,
  local containerPort = k.core.v1.containerPort,
  local deployment = k.apps.v1.deployment,

  minio_container::
    container.new('minio', 'minio/minio:latest') +
    container.withPorts([
      containerPort.new('minio', 9000),
      containerPort.new('minio-console', 9010),
    ]) +
    container.withCommand([
      'sh',
      '-euc',
      'mkdir -p /data/tempo && minio server /data --console-address=:9010',
    ]) +
    container.withEnvMap({
      MINIO_ACCESS_KEY: 'tempo',
      MINIO_SECRET_KEY: 'supersecret',
    }),

  minio_deployment:
    deployment.new('minio',
                   1,
                   [$.minio_container],
                   { app: 'minio' }),

  minio_service:
    k.util.serviceFor($.minio_deployment),
}

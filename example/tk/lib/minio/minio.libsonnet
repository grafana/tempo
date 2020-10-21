{
  local configMap = $.core.v1.configMap,
  local container = $.core.v1.container,
  local containerPort = $.core.v1.containerPort,
  local deployment = $.apps.v1.deployment,

  minio_container::
    container.new('minio', 'minio/minio:RELEASE.2020-07-27T18-37-02Z') +
    container.withPorts([
      containerPort.new('minio', 9000),
    ]) +
    container.withCommand([
        'sh',
        '-euc',
        'mkdir -p /data/tempo && /usr/bin/minio server /data',
    ]) +
    container.withEnvMap({
      MINIO_ACCESS_KEY: 'tempo',
      MINIO_SECRET_KEY: 'supersecret',
    }),

  minio_deployment:
    deployment.new('minio',
                   1,
                   [ $.minio_container ],
                   { app: 'minio' }),

  minio_service:
    $.util.serviceFor($.minio_deployment)
}

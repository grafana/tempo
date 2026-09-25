{
  local k = import 'ksonnet-util/kausal.libsonnet',
  local configMap = k.core.v1.configMap,
  local container = k.core.v1.container,
  local containerPort = k.core.v1.containerPort,
  local deployment = k.apps.v1.deployment,

  rustfs_container::
    container.new('rustfs', 'rustfs/rustfs:1.0.0') +
    container.withPorts([
      containerPort.new('rustfs', 9000),
      containerPort.new('rustfs-console', 9010),
    ]) +
    container.withCommand([
      'sh',
      '-euc',
      'mkdir -p /data/tempo && rustfs /data',
    ]) +
    container.withEnvMap({
      RUSTFS_ACCESS_KEY: 'tempo',
      RUSTFS_SECRET_KEY: 'supersecret',
      RUSTFS_CONSOLE_ENABLE: 'true',
      RUSTFS_CONSOLE_ADDRESS: ':9010',
    }),

  rustfs_deployment:
    deployment.new('rustfs',
                   1,
                   [$.rustfs_container],
                   { app: 'rustfs' }),

  rustfs_service:
    k.util.serviceFor($.rustfs_deployment),
}

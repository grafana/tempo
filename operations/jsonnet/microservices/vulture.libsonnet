{
  local k = import 'ksonnet-util/kausal.libsonnet',
  local container = k.core.v1.container,
  local containerPort = k.core.v1.containerPort,
  local deployment = k.apps.v1.deployment,

  local target_name = 'vulture',
  local port = 8080,

  tempo_vulture_container::
    container.new(target_name, $._images.tempo_vulture) +
    container.withPorts([
      containerPort.new('prom-metrics', port),
    ]) +
    container.withArgs([
      '-prometheus-listen-address=:' + port,
      '-tempo-push-url=' + $._config.vulture.tempoPushUrl,
      '-tempo-query-url=' + $._config.vulture.tempoQueryUrl,
      '-tempo-org-id=' + $._config.vulture.tempoOrgId,
      '-tempo-retention-duration=' + $._config.vulture.tempoRetentionDuration,
      '-tempo-search-backoff-duration=' + $._config.vulture.tempoSearchBackoffDuration,
      '-tempo-read-backoff-duration=' + $._config.vulture.tempoReadBackoffDuration,
      '-tempo-write-backoff-duration=' + $._config.vulture.tempoWriteBackoffDuration,
    ]) +
    k.util.resourcesRequests('50m', '100Mi') +
    k.util.resourcesLimits('100m', '500Mi'),

  tempo_vulture_deployment:
    deployment.new(target_name,
                   $._config.vulture.replicas,
                   [
                     $.tempo_vulture_container,
                   ],
                   {
                     app: target_name,
                   }),
}

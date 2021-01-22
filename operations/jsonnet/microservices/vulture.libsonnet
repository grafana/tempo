{
  local container = $.core.v1.container,
  local containerPort = $.core.v1.containerPort,
  local deployment = $.apps.v1.deployment,

  local target_name = 'vulture',
  local port = 8080,

  tempo_vulture_container::
    container.new(target_name, $._images.tempo_vulture) +
    container.withPorts([
      containerPort.new('prom-metrics', port),
    ]) +
    container.withArgs([
      '-loki-base-url=' + $._config.vulture.loki_base_url,
      '-loki-query=' + $._config.vulture.loki_query,
      '-loki-user=' + $._config.vulture.loki_user,
      '-loki-pass=' + $._config.vulture.loki_pass,
      '-prometheus-listen-address=:' + port,
      '-tempo-base-url=http://query-frontend:3100',
      '-logtostderr=true',
      '-tempo-org-id=1',
    ]) +
    $.util.resourcesRequests('50m', '100Mi') +
    $.util.resourcesLimits('100m', '500Mi'),

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

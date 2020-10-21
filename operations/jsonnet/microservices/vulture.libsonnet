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
      '-loki-base-url=https://logs-dev-ops-tools1.grafana.net',
      '-loki-query={cluster="ops-tools1", job="cortex-ops/query-frontend"} |= "query_range"',
      '-loki-user=29',
      '-loki-pass=eyJrIjoiZWFlYmJmNjQzMTIwYWJmZjBiOTRiZjJlN2Y5OWQwYWZhYThjNjBmMiIsIm4iOiJsb2tpLXJlYWQtbG9raS1jYW5hcnkiLCJpZCI6NTAwMH0=',
      '-prometheus-listen-address=:' + port,
      '-tempo-base-url=http://querier:3100',
      '-logtostderr=true',
      '-tempo-org-id=1',
    ]) +
    $.util.resourcesRequests('50m', '100Mi') +
    $.util.resourcesLimits('100m', '500Mi'),

  tempo_vulture_deployment:
    deployment.new(target_name,
                   1,
                   [
                     $.tempo_vulture_container,
                   ],
                   {
                     app: target_name,
                   }),
}

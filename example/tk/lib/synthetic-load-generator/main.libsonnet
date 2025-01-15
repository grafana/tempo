{
  local k = import 'ksonnet-util/kausal.libsonnet',
  local container = k.core.v1.container,
  local deployment = k.apps.v1.deployment,

  synthetic_load_generator_container::
    container.new('synthetic-load-gen', 'ghcr.io/grafana/xk6-client-tracing:v0.0.5') +
    container.withEnvMap({
      ENDPOINT: 'http://tempo:4317',
    }),

  synthetic_load_generator_deployment:
    deployment.new('synthetic-load-generator',
                   1,
                   [ $.synthetic_load_generator_container ],
                   { app: 'synthetic_load_generator' }),
}

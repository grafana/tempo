local k = import 'ksonnet-util/kausal.libsonnet';

{
  new(
    name,
    data_source_uri='$(HOSTNAME):$(PORT)/postgres',
    ssl=true,
    image='quay.io/prometheuscommunity/postgres-exporter:v0.10.0',
  ):: {
    local this = self,

    local container = k.core.v1.container,
    local containerPort = k.core.v1.containerPort,
    container::
      container.new('postgres-exporter', image)
      + container.withPorts(containerPort.new('http-metrics', 9187))
    ,

    local ssl_suffix =
      if ssl
      then ''
      else '?sslmode=disable',

    local deployment = k.apps.v1.deployment,
    deployment:
      deployment.new(
        name,
        1,
        [
          this.container
          // Force DATA_SOURCE_URI to be declared after the variables it references
          + container.withEnvMap({
            DATA_SOURCE_URI: data_source_uri + ssl_suffix,
          }),
        ]
      ),
  },

  // withEnv is used to declare env vars for:
  // DATA_SOURCE_USER
  // DATA_SOURCE_PASS
  // HOSTNAME
  // PORT
  // Argument `env` is an array with k.core.v1.envVar objects
  withEnv(env):: {
    container+:
      k.core.v1.container.withEnv(env),
  },

  withImage(image):: {
    container+:: k.core.v1.container.withImage(image),
  },

  withAutoDiscover():: {
    container+:
      k.core.v1.container.withEnvMixin([
        k.core.v1.envVar.new(
          'PG_EXPORTER_AUTO_DISCOVER_DATABASES',
          'true',
        ),
      ]),
  },

  withExcludeDatabases(databases):: {
    container+:
      k.core.v1.container.withEnvMixin([
        k.core.v1.envVar.new(
          'PG_EXPORTER_EXCLUDE_DATABASES',
          std.join(',', databases),
        ),
      ]),
  },
}

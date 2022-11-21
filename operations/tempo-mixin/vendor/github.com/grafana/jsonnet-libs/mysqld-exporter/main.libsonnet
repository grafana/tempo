local k = import 'ksonnet-util/kausal.libsonnet';
local container = k.core.v1.container;

{
  local root = self,

  new(
    name,
    user,
    host,
    port=3306,
    image='prom/mysqld-exporter:v0.13.0',
    tlsMode='preferred',
  ):: {
    local this = self,

    envMap:: {
      MYSQL_USER: user,
      MYSQL_HOST: host,
      MYSQL_PORT: std.toString(port),
      MYSQL_TLS_MODE: tlsMode,
    },

    local containerPort = k.core.v1.containerPort,
    container::
      container.new('mysqld-exporter', image)
      + container.withPorts(k.core.v1.containerPort.new('http-metrics', 9104))
      + container.withArgsMixin([
        '--collect.info_schema.innodb_metrics',
      ])
      + container.withEnvMap(this.envMap)
    ,

    local deployment = k.apps.v1.deployment,
    deployment:
      deployment.new(
        name,
        1,
        [
          this.container
          // Force DATA_SOURCE_NAME to be declared after the variables it references
          + container.withEnvMap({
            DATA_SOURCE_NAME: '$(MYSQL_USER):$(MYSQL_PASSWORD)@tcp($(MYSQL_HOST):$(MYSQL_PORT))/?tls=$(MYSQL_TLS_MODE)',
          }),
        ]
      ),

    service:
      k.util.serviceFor(this.deployment),
  },

  withPassword(password):: {
    secret:
      k.core.v1.secret.new(
        super.name + '-password',
        {
          password: std.base64(password),
        },
        'Opaque',
      ),

    container+:: root.withPasswordSecretRef(self.secret.metadata.name).container,
  },

  withPasswordSecretRef(secretRefName, key='password'):: {
    local envVar = k.core.v1.envVar,
    container+:: container.withEnvMixin([
      envVar.fromSecretRef('MYSQL_PASSWORD', secretRefName, key),
    ]),
  },

  withImage(image):: {
    container+:: container.withImage(image),
  },

  withTLSMode(mode):: {
    local envVar = k.core.v1.envVar,
    container+:: container.withEnvMixin([
      envVar.new('MYSQL_TLS_MODE', mode),
    ]),
  },

  args:: {
    lockWaitTimeout(timeout):: {
      container+::
        container.withArgsMixin([
          '--exporter.lock_wait_timeout=' + timeout,
        ]),
    },
    collect:: {
      perfSchema(value):: {
        container+::
          container.withArgsMixin([
            '--collect.perf_schema.' + value,
          ]),
      },
      tablelocks:: self.perfSchema('tablelocks'),
      memoryEvents:: self.perfSchema('memory_events'),
    },
  },
}

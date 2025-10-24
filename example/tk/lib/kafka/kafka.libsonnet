{

  local k = import 'ksonnet-util/kausal.libsonnet',
  local deployment = k.apps.v1.deployment,
  local container = k.core.v1.container,
  local containerPort = k.core.v1.containerPort,
  local service = k.core.v1.service,
  local servicePort = k.core.v1.servicePort,

  _config+:: {
    kafka_cluster_id: 'zH1GDqcNTzGMDCXm5VZQdg',
    kafka_image: 'confluentinc/cp-kafka:latest',
    kafka_broker_id: 1,
    kafka_num_partitions: 100,
  },

  kafka_container::
    container.new('kafka', $._config.kafka_image)
    + container.withPorts([
      containerPort.new('broker', 9092),
      containerPort.new('controller', 9093),
      containerPort.new('external', 29092),
    ])
    + container.withEnv([
      { name: 'CLUSTER_ID', value: $._config.kafka_cluster_id },
      { name: 'KAFKA_BROKER_ID', value: std.toString($._config.kafka_broker_id) },
      { name: 'KAFKA_NUM_PARTITIONS', value: std.toString($._config.kafka_num_partitions) },
      { name: 'KAFKA_PROCESS_ROLES', value: 'broker,controller' },
      { name: 'KAFKA_LISTENERS', value: 'PLAINTEXT://:9092,CONTROLLER://:9093,PLAINTEXT_HOST://:29092' },
      { name: 'KAFKA_ADVERTISED_LISTENERS', value: 'PLAINTEXT://kafka:9092,PLAINTEXT_HOST://localhost:29092' },
      { name: 'KAFKA_LISTENER_SECURITY_PROTOCOL_MAP', value: 'PLAINTEXT:PLAINTEXT,CONTROLLER:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT' },
      { name: 'KAFKA_INTER_BROKER_LISTENER_NAME', value: 'PLAINTEXT' },
      { name: 'KAFKA_CONTROLLER_LISTENER_NAMES', value: 'CONTROLLER' },
      { name: 'KAFKA_CONTROLLER_QUORUM_VOTERS', value: '1@localhost:9093' },
      { name: 'KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR', value: '1' },
      { name: 'KAFKA_LOG_RETENTION_CHECK_INTERVAL_MS', value: '10000' },
      { name: 'KAFKA_LOG4J_ROOT_LOGLEVEL', value: 'WARN' },
    ])
    + container.readinessProbe.tcpSocket.withPort(9092)
    + container.readinessProbe.withInitialDelaySeconds(30)
    + container.readinessProbe.withTimeoutSeconds(1)
    + container.readinessProbe.withPeriodSeconds(1)
    + container.readinessProbe.withFailureThreshold(30)
    + k.util.resourcesRequests('500m', '1Gi'),

  kafka_deployment:
    deployment.new('kafka', 1, [$.kafka_container])
    + deployment.mixin.metadata.withNamespace($._config.namespace)
    + deployment.mixin.spec.selector.withMatchLabels({ name: 'kafka' })
    + deployment.mixin.spec.template.metadata.withLabels({ name: 'kafka' })
    // Set enableServiceLink to false to prevent environment variables from being injected into the pod
    + deployment.mixin.spec.template.spec.withEnableServiceLinks(false),

  kafka_service:
    k.util.serviceFor($.kafka_deployment)
    + service.mixin.metadata.withName('kafka')
    + service.mixin.spec.withPorts([
      servicePort.newNamed('broker', 9092, 9092),
      servicePort.newNamed('controller', 9093, 9093),
      servicePort.newNamed('external', 29092, 29092),
    ]),

  // Redpanda Console for Kafka UI
  redpanda_console_container::
    container.new('redpanda-console', 'docker.redpanda.com/redpandadata/console:v2.7.0')
    + container.withPorts([
      containerPort.new('http', 8080),
    ])
    + container.withEnv([
      { name: 'KAFKA_BROKERS', value: 'kafka:9092' },
    ])
    + k.util.resourcesRequests('100m', '128Mi'),

  redpanda_console_deployment:
    deployment.new('redpanda-console', 1, [$.redpanda_console_container])
    + deployment.mixin.metadata.withNamespace($._config.namespace)
    + deployment.mixin.spec.selector.withMatchLabels({ name: 'redpanda-console' })
    + deployment.mixin.spec.template.metadata.withLabels({ name: 'redpanda-console' }),

  redpanda_console_service:
    k.util.serviceFor($.redpanda_console_deployment)
    + service.mixin.metadata.withName('redpanda-console')
    + service.mixin.spec.withPorts([
      servicePort.newNamed('http', 8080, 8080),
    ]),
}

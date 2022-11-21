{
  grafanaDashboards: {
    'kafka-overview.json': (import 'dashboards/kafka-overview.json'),
    'kafka-topics.json': (import 'dashboards/kafka-topics.json'),
    'zookeeper-overview.json': (import 'dashboards/zookeeper-overview.json'),
    'ksqldb-overview.json': (import 'dashboards/ksqldb-overview.json'),
    'connect-overview.json': (import 'dashboards/connect-overview.json'),
    'schema-registry-overview.json': (import 'dashboards/schema-registry-overview.json'),
    'kafka-lag-overview.json': (import 'dashboards/kafka-lag-overview.json'),
  },

  // Helper function to ensure that we don't override other rules, by forcing
  // the patching of the groups list, and not the overall rules object.
  local importRules(rules) = {
    groups+: std.native('parseYaml')(rules)[0].groups,
  },

  prometheusAlerts+:
    importRules(importstr 'alerts/KafkaAlerts.yml'),
}

{
  grafanaDashboards: {
    'MongoDB_Instance.json': (import 'dashboards/MongoDB_Instance.json'),
    'MongoDB_ReplicaSet.json': (import 'dashboards/MongoDB_ReplicaSet.json'),
    'MongoDB_Cluster.json': (import 'dashboards/MongoDB_Cluster.json'),
  },

  // Helper function to ensure that we don't override other rules, by forcing
  // the patching of the groups list, and not the overall rules object.
  local importRules(rules) = {
    groups+: std.native('parseYaml')(rules)[0].groups,
  },

  prometheusAlerts+:
    importRules(importstr 'alerts/mongodbAlerts.yaml'),
}

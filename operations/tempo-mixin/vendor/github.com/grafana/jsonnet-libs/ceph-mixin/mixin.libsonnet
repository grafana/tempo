{
  grafanaDashboards: {
    'ceph-cluster.json': (import 'dashboards/ceph-cluster.json'),
  },

  // Helper function to ensure that we don't override other rules, by forcing
  // the patching of the groups list, and not the overall rules object.
  local importRules(rules) = {
    groups+: std.native('parseYaml')(rules)[0].groups,
  },

  prometheusAlerts+:
    importRules(importstr 'alerts/CephClusterAlerts.yml'),
}

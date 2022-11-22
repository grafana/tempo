{
  grafanaDashboards: {
    'API_Metrics.json': (import 'dashboards/API_Metrics.json'),
    'Cluster_Metrics.json': (import 'dashboards/Cluster_Metrics.json'),
    'Node_Metrics.json': (import 'dashboards/Node_Metrics.json'),
    'Inbound_Endpoint_Metrics.json': (import 'dashboards/Inbound_Endpoint_Metrics.json'),
    'Proxy_Service_Metrics.json': (import 'dashboards/Proxy_Service_Metrics.json'),
  },

  // Helper function to ensure that we don't override other rules, by forcing
  // the patching of the groups list, and not the overall rules object.
  local importRules(rules) = {
    groups+: std.native('parseYaml')(rules)[0].groups,
  },

}

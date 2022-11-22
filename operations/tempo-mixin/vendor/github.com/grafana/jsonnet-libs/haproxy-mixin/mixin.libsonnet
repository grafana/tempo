{
  grafanaDashboards: {
    'haproxy-overview.json': (import 'dashboards/haproxy-overview.json'),
    'haproxy-frontend.json': (import 'dashboards/haproxy-frontend.json'),
    'haproxy-backend.json': (import 'dashboards/haproxy-backend.json'),
    'haproxy-server.json': (import 'dashboards/haproxy-server.json'),
  },

  // Helper function to ensure that we don't override other rules, by forcing
  // the patching of the groups list, and not the overall rules object.
  local importRules(rules) = {
    groups+: std.native('parseYaml')(rules)[0].groups,
  },

  prometheusRules+: importRules(importstr 'rules/rules.yaml'),

  prometheusAlerts+:
    importRules(importstr 'alerts/general.yaml'),
}

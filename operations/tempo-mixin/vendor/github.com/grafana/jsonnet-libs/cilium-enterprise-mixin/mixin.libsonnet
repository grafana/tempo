(import 'alerts/ciliumAlerts.libsonnet') +
{
  grafanaDashboards: {
    'cilium-agent-overview.json': (import 'dashboards/cilium-agent-overview.json'),
    'cilium-overview.json': (import 'dashboards/cilium-overview.json'),
    'cilium-operator.json': (import 'dashboards/cilium-operator.json'),
    'hubble-overview.json': (import 'dashboards/hubble-overview.json'),
    'hubble-timescape.json': (import 'dashboards/hubble/hubble-timescape.json'),
    // 'cilium-actionable.json': (import 'dashboards/cilium-agent/cilium-actionable.json'),
    'cilium-agent.json': (import 'dashboards/cilium-agent/cilium-agent.json'),
    'cilium-api.json': (import 'dashboards/cilium-agent/cilium-api.json'),
    'cilium-bpf.json': (import 'dashboards/cilium-agent/cilium-bpf.json'),
    'cilium-conntrack.json': (import 'dashboards/cilium-agent/cilium-conntrack.json'),
    'cilium-datapath.json': (import 'dashboards/cilium-agent/cilium-datapath.json'),
    'cilium-external-fqdn-proxy.json': (import 'dashboards/cilium-agent/cilium-external-fqdn-proxy.json'),
    'cilium-fqdn-proxy.json': (import 'dashboards/cilium-agent/cilium-fqdn-proxy.json'),
    'cilium-identities.json': (import 'dashboards/cilium-agent/cilium-identities.json'),
    'cilium-kubernetes.json': (import 'dashboards/cilium-agent/cilium-kubernetes.json'),
    'cilium-L3-policy.json': (import 'dashboards/cilium-agent/cilium-L3-policy.json'),
    'cilium-L7-proxy.json': (import 'dashboards/cilium-agent/cilium-L7-proxy.json'),
    'cilium-network.json': (import 'dashboards/cilium-agent/cilium-network.json'),
    'cilium-nodes.json': (import 'dashboards/cilium-agent/cilium-nodes.json'),
    'cilium-policy.json': (import 'dashboards/cilium-agent/cilium-policy.json'),
    'cilium-resource-utilization.json': (import 'dashboards/cilium-agent/cilium-resource-utilization.json'),
  },

  // Helper function to ensure that we don't override other rules, by forcing
  // the patching of the groups list, and not the overall rules object.
  local importRules(rules) = {
    groups+: std.native('parseYaml')(rules)[0].groups,
  },
}

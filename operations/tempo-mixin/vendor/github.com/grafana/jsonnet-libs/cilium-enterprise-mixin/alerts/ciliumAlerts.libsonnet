{
  // TODO: All alerts should have configurable cluster, or at least specify which cluster the alert is coming from
  prometheusAlerts+:: {
    groups+: [
      {
        name: 'Cilium Endpoints',
        rules: [
          {
            alert: 'CiliumAgentEndpointFailures',
            expr: 'sum(cilium_endpoint_state{endpoint_state="invalid"}) by (pod)',
            annotations:
              {
                summary: 'Cilium Agent endpoints in the invalid state.',
                description: 'Cilium Agent {{$labels.pod}} has endpoints that are in an invalid state. This may result in problems with scheduling Pods, or network connectivity issues.',
              },
            labels: {
              severity: 'warning',
            },
            'for': '5m',
          },
          {
            alert: 'CiliumAgentEndpointUpdateFailure',
            expr: 'sum(rate(cilium_k8s_client_api_calls_total{method=~"(PUT|POST|PATCH)", endpoint="endpoint",return_code!~"2[0-9][0-9]"}[5m])) by (pod, method, return_code)',
            annotations: {
              summary: 'API calls to Cilium Agent API to create or update Endpoints are failing.',
              description: 'API calls to Cilium Agent API to create or update Endpoints are failing on pod {{$labels.pod}} ({{$labels.method}} {{$labels.return_code}}).\n\nThis may cause problems for Pod scheduling',
            },
            labels: {
              severity: 'warning',
            },
            'for': '5m',
          },
          {
            alert: 'CiliumAgentContainerNetworkInterfaceApiErrorEndpointCreate',
            expr: 'sum(rate(cilium_api_limiter_processed_requests_total{api_call=~"endpoint-create", outcome="fail"}[1m])) by (pod,api_call)',
            annotations: {
              summary: 'Cilium Endpoint API endpoint rate limiter is reporting errors while doing endpoint create.',
              description: 'Cilium Endpoint API endpoint rate limiter on Pod {{$labels.pod}} is reporting errors while doing endpoint create.\nThis may cause CNI and prevent Cilium scheduling.',
            },
            labels: {
              severity: 'info',
            },
            'for': '5m',
          },
          {
            alert: 'CiliumAgentApiEndpointErrors',
            expr: 'sum(rate(cilium_agent_api_process_time_seconds_count{return_code=~"5[0-9][0-9]", path="/v1/endpoint"}[5m])) by (pod, return_code)',
            labels: {
              severity: 'warning',
            },
            'for': '5m',
            annotations:
              {
                summary: 'API calls to Cilium Endpoints API are failing due to server errors.',
                description: 'API calls to Cilium Endpoints API on Agent Pod {{$labels.pod}} are failing due to server errors ({{$labels.return_code}}).\n\nThis could indicate issues with Ciliums ability to create endpoints which can result in failure to schedule Kubernetes Pods.',
              },
          },
        ],
      },
      {
        name: 'Cilium IPAM',
        rules: [
          {
            alert: 'CiliumOperatorExhaustedIpamIps',
            expr: 'sum(cilium_operator_ipam_ips{type="available"}) >= 1',
            annotations: {
              summary: 'Cilium Operator has exhausted its IPAM IPs.',
              description: 'Cilium Operator {{$labels.pod}} has exhausted its IPAM IPs. This is a critical issue which may cause Pods to fail to be scheduled.\n\nThis may be caused by number of Pods being scheduled exceeding the you cloud platforms network limits or issues with Cilium rate limiting.',
            },
            labels: {
              severity: 'critical',
            },
            'for': '5m',
          },
          {
            // Should be relative time range of 600-0
            alert: 'CiliumOperatorLowAvailableIpamIps',
            expr: '(sum(cilium_operator_ipam_ips{type!="available"}) / sum(cilium_operator_ipam_ips)) > 0.9',
            annotations: {
              summary: 'Cilium Operator has used up over 90% of its available IPs.',
              description: 'Cilium Operator {{$labels.pod}} has used up over 90% of its available IPs. If available IPs become exhausted then the operator may not be able to schedule Pods.\n\nThis may be caused by number of Pods being scheduled exceeding the you cloud platforms network limits or issues with Cilium rate limiting.',
            },
            labels: {
              severity: 'warning',
            },
            'for': '5m',
          },
        ],
      },
      {
        name: 'Cilium Maps',
        rules: [
          {
            alert: 'CiliumAgentMapOperationFailures',
            expr: 'sum(rate(cilium_bpf_map_ops_total{k8s_app="cilium", outcome="fail",pod=~"$pod"}[5m])) by (map_name, pod) > 0',
            labels: {
              severity: 'warning',
            },
            annotations: {
              summary: 'Cilium Agent is experiencing errors updating BPF maps on Agent Pod.',
              description: 'Cilium Agent {{$labels.pod}} is experiencing errors updating BPF maps on Agent Pod {{$labels.pod}}. Effects may vary depending on map type(s) being affected however this is likely to cause issues with Cilium.',
            },
            'for': '5m',
          },
          {
            alert: 'CiliumAgentBpfMapPressure',
            expr: 'cilium_bpf_map_pressure{} > 0.9',
            annotations: {
              summary: 'Map on Cilium Agent Pod is currently experiencing high map pressure.',
              description: 'Map {{$labels.map_name}} on Cilium Agent Pod is currently experiencing high map pressure. The map is currently over 90% full. Full maps will begin to experience errors on updates which may result in unexpected behaviour.',
            },
            labels: {
              severity: 'warning',
            },
            'for': '5m',
          },
        ],
      },
      {
        name: 'Cilium NAT',
        rules: [
          {
            alert: 'CiliumAgentNatTableFull',
            expr: 'sum(rate(cilium_drop_count_total{reason="No mapping for NAT masquerade"}[1m])) by (pod)',
            annotations: {
              summary: 'Cilium Agent Pod is dropping packets due to "No mapping for NAT masquerade" errors.',
              description: 'Cilium Agent Pod {{$labels.pod}} is dropping packets due to "No mapping for NAT masquerade" errors. This likely means that the Cilium agents NAT table is full.\nThis is a potentially critical issue that can lead to connection issues for packets leaving the cluster network.\n\nSee: https://docs.cilium.io/en/v1.9/concepts/networking/masquerading/ for more info.',
            },
            labels: {
              severity: 'critical',
            },
            'for': '5m',
          },
        ],
      },
      {
        name: 'Cilium API',
        rules: [
          {
            alert: 'CiliumAgentApiHighErrorRate',
            expr: 'sum(rate(cilium_k8s_client_api_calls_total{endpoint!="metrics",return_code!~"2[0-9][0-9]"}[5m])) by (pod, endpoint, return_code)',
            annotations: {
              summary: 'Cilium Agent API on Pod is experiencing a high error rate.',
              description: 'Cilium Agent API on Pod {{$labels.pod}} is experiencing a high error rate for response code: {{$labels.response_code}} on endpoint {{$labels.endpoint}}.',
            },
            labels: {
              severity: 'info',
            },
            'for': '5m',
          },
        ],
      },
      {
        name: 'Cilium Conntrack',
        rules: [
          {
            alert: 'CiliumAgentConntrackTableFull',
            expr: 'sum(rate(cilium_drop_count_total{reason="CT: Map insertion failed"}[5m])) by (pod)',
            annotations: {
              summary: 'Ciliums conntrack map is failing on new insertions on Agent Pod.',
              description: 'Ciliums conntrack map is failing on new insertions on agent Pod {{$labels.pod}}, this likely means that the conntrack BPF map is full. This is a potentially critical issue and may result in unexpected packet drops.\n\nIf this is firing, it is recommend to look at both CPU/memory resource utilization dashboards. As well as conntrack GC run dashboards for more details on what the issue is.',
            },
            labels: {
              severity: 'critical',
            },
            'for': '5m',
          },
          {
            // TODO: According to alert dump this should have two conditions/time ranges
            alert: 'CiliumAgentConnTrackFailedGarbageCollectorRuns',
            expr: 'sum(rate(cilium_datapath_conntrack_gc_runs_total{status="uncompleted"}[5m])) by (pod) > 0',
            annotations: {
              summary: 'Cilium Agent Conntrack GC runs are failing on Agent Pod.',
              description: 'Cilium Agent Conntrack GC runs on Agent Pod {{$labels.pod}} has been reported as not completing. Runs reported "uncompleted" may indicate a problem with ConnTrack GC.\nCilium failing to GC its ConnTrack table may cause further ConnTrack issues later. This may result in dropped packets or other issues.',
            },
            labels: {
              severity: 'warning',
            },
            'for': '5m',
          },
        ],
      },
      {
        name: 'Cilium Drops',
        rules: [
          {
            alert: 'CiliumAgentHighDeniedRate',
            expr: 'sum(rate(cilium_drop_count_total{reason="Policy denied"}[1m])) by (reason, pod)',
            annotations: {
              summary: 'Cilium Agent is experiencing a high drop rate due to policy rule denies.',
              description: 'Cilium Agent Pod {{$labels.pod}} is experiencing a high drop rate due to policy rule denies. This could mean that a network policy is not configured correctly, or that a Pod is sending unexpected network traffic',
            },
            labels: {
              severity: 'info',
            },
            'for': '5m',
          },
        ],
      },
      {
        name: 'Cilium Policy',
        rules: [
          {
            alert: 'CiliumAgentPolicyMapPressure',
            expr: 'sum(cilium_bpf_map_pressure{map_name=~"cilium_policy_.*"}) by (pod) > 0.9',
            annotations: {
              summary: 'Cilium Agent is experiencing high BPF map pressure.',
              description: 'Cilium Agent {{$labels.pod}} is experiencing high BPF map pressure (over 90% full) on policy map: {{$labels.map_name}}. This means that the map is running low on capacity. A full policy map may result in packet drops.',
            },
            labels: {
              severity: 'warning',
            },
            'for': '5m',
          },
        ],
      },
      {
        name: 'Cilium Identity',
        rules: [
          {
            alert: 'CiliumNodeLocalHighIdentityAllocation',
            expr: '(sum(cilium_identity{type="node_local"}) by (pod) / (2^16-1)) > 0.8',
            annotations: {
              summary: 'Cilium is using a very high percent (over 80%) of its maximum per-node identity limit (65535).',
              description: 'Cilium agent Pod {{$labels.pod}} is using a very high percent (over 80%) of its maximum per-node identity limit (65535).\n\nIf this capacity is exhausted Cilium may be unable to allocate new identities. Very high identity allocations can also indicate other problems',
            },
            labels: {
              severity: 'warning',
            },
            'for': '5m',
          },
          {
            alert: 'RunningOutOfCiliumClusterIdentities',
            expr: 'sum(cilium_identity{type="cluster_local"}) / (2^16-256) > .8',
            annotations: {
              summary: 'Cilium is using a very high percent of its maximum cluster identity limit (65280).',
              description: 'Cilium is using a very high percent of its maximum cluster identity limit ({{value}}/65280) . If this capacity is exhausted Cilium may be unable to allocate new identities. Very high identity allocations can also indicate other problems',
            },
            labels: {
              severity: 'warning',
            },
            'for': '5m',
          },
        ],
      },
      {
        name: 'Cilium Nodes',
        rules: [
          {
            alert: 'CiliumUnreachableNodes',
            expr: 'sum(cilium_unreachable_nodes{}) by (pod) > 0',
            labels: {
              severity: 'info',
            },
            annotations: {
              summary: 'Cilium Agent is reporting unreachable Nodes in the cluster.',
              description: 'Cilium Agent {{$labels.pod}} is reporting unreachable Nodes in the cluster.',
            },
            'for': '15m',
          },
        ],
      },
    ],
  },
}

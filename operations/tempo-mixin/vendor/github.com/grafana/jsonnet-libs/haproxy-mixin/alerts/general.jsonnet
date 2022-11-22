{
  groups: [
    {
      name: 'HAProxyAlerts',
      rules: [
        {
          alert: 'HAProxyDroppingLogs',
          expr: 'rate(haproxy_process_dropped_logs_total[5m]) != 0',
          'for': '5s',
          labels: {
            severity: 'error',
          },
          annotations: {
            description: 'HAProxy {{$labels.job}} on {{$labels.instance}} is dropping logs.',
            summary: 'HAProxy is dropping logs',
          },
        },
        {
          alert: 'HAProxyBackendCheckFlapping',
          expr: 'rate(haproxy_backend_check_up_down_total[5m]) != 0',
          'for': '1m',
          labels: {
            severity: 'error',
          },
          annotations: {
            description: 'HAProxy {{$labels.job}} backend {{$labels.proxy}} on {{$labels.instance}} has flapping checks.',
            summary: 'HAProxy backend checks are flapping',
          },
        },
        {
          alert: 'HAProxyServerCheckFlapping',
          expr: 'rate(haproxy_server_check_up_down_total[5m]) != 0',
          'for': '1m',
          labels: {
            severity: 'error',
          },
          annotations: {
            description: 'HAProxy {{$labels.job}} server {{$labels.server}} on {{$labels.instance}} has flapping checks.',
            summary: 'HAProxy server checks are flapping',
          },
        },
      ],
    },
  ],
}

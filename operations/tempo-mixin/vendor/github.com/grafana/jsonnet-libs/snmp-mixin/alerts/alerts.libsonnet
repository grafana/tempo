{
  prometheusAlerts+:: {
    groups+: [
      {
        name: 'snmp',
        rules: [
          {
            alert: 'SNMPTargetDown',
            expr: 'up{job_snmp=~"integrations/snmp.*"} == 0',
            labels: {
              severity: 'critical',
            },
            annotations: {
              summary: 'SNMP Target is down.',
              description: 'SNMP target {{$labels.snmp_target}} on instance {{$labels.instance}} from job {{$labels.job}} is down.',
            },
            'for': '5m',
          },
          {
            alert: 'SNMPTargetInterfaceDown',
            expr: 'ifOperStatus{job_snmp=~"integrations/snmp.*"} == 2',
            labels: {
              severity: 'warning',
            },
            annotations: {
              summary: 'Network interface on SNMP target is down.',
              description: 'SNMP interface {{$labels.ifDescr}} on target {{$labels.snmp_target}} on instance {{$labels.instance}} from job {{$labels.job}} is down.',
            },
            'for': '5m',
          },
        ],
      },
    ],
  },
}

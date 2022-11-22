{
  prometheusAlerts+:: {
    groups+: [
      {
        name: 'DiscourseAlerts',
        rules: [
          {
            alert: 'DiscourseRequestsHigh5xxErrors',
            expr: |||
              100 * rate(discourse_http_requests{status="500"}[5m]) / on() group_left() (sum(rate(discourse_http_requests[5m])) by (instance)) > %(alertsCritical5xxResponses)s
            ||| % $._config,
            'for': '0',
            labels: {
              severity: 'critical',
            },
            annotations: {
              summary: 'More than %(alertsCritical5xxResponses)s%% of all requests result in a 5XX.' % $._config,
              description:
                ('{{ printf "%%.2f" $value }}%% of all requests are resulting in 500 status codes, ' +
                 'which is above the threshold %(alertsCritical5xxResponses)s%%, ' +
                 'indicating a potentially larger issue for {{$labels.instance}}') % $._config,
            },
          },
          {
            alert: 'DiscourseRequestsHigh4xxErrors',
            expr: |||
              100 * rate(discourse_http_requests{status=~"^4.*"}[5m]) / on() group_left() (sum(rate(discourse_http_requests[5m])) by (instance)) > %(alertsWarning4xxResponses)s
            ||| % $._config,
            'for': '0',
            labels: {
              severity: 'warning',
            },
            annotations: {
              summary: 'More than %(alertsWarning4xxResponses)s%% of all requests result in a 4XX.' % $._config,
              description:
                ('{{ printf "%%.2f" $value }}%% of all requests are resulting in 400 status code, ' +
                 'which is above the threshold %(alertsWarning4xxResponses)s%%, ' +
                 'indicating a potentially larger issue for {{$labels.instance}}') % $._config,
            },
          },
        ],
      },
    ],
  },
}

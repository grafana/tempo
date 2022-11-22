{
  prometheusAlerts+:: {
    groups+: [
      {
        name: 'apache-http',
        rules: [
                 {
                   alert: 'ApacheDown',
                   expr: 'apache_up == 0',
                   labels: {
                     severity: 'warning',
                   },
                   annotations: {
                     summary: 'Apache is down.',
                     description: 'Apache is down on {{ $labels.instance }}.',
                   },
                   'for': '5m',
                 },
                 {
                   alert: 'ApacheRestart',
                   expr: 'apache_uptime_seconds_total / 60 < 1',
                   labels: {
                     severity: 'info',
                   },
                   annotations: {
                     summary: 'Apache restart.',
                     description: 'Apache has just been restarted on {{ $labels.instance }}.',
                   },
                   'for': '0',
                 },
                 {
                   alert: 'ApacheWorkersLoad',
                   expr: |||
                     (sum by (instance) (apache_workers{state="busy"}) / sum by (instance) (apache_scoreboard) ) * 100 > %(alertsWarningWorkersBusy)s
                   ||| % $._config,
                   'for': '15m',
                   labels: {
                     severity: 'warning',
                   },
                   annotations: {
                     summary: 'Apache workers load is too high.',
                     description: |||
                       Apache workers in busy state approach the max workers count %(alertsWarningWorkersBusy)s%% workers busy on {{ $labels.instance }}.
                       The currect value is {{ $value }}%%.
                     ||| % $._config,
                   },
                 },
                 {
                   alert: 'ApacheResponseTimeTooHigh',
                   expr: |||
                     increase(apache_duration_ms_total[5m])/increase(apache_accesses_total[5m]) > %(alertsWarningResponseTimeMs)s
                   ||| % $._config,
                   'for': '15m',
                   labels: {
                     severity: 'warning',
                   },
                   annotations: {
                     summary: 'Apache response time is too high.',
                     description: |||
                       Apache average response time is above the threshold of %(alertsWarningResponseTimeMs)s ms on {{ $labels.instance }}.
                       The currect value is {{ $value }} ms.
                     ||| % $._config,
                   },
                 },
               ]
               +
               if $._config.enableLokiLogs then
                 [
                   {
                     alert: 'ApacheErrorsRateTooHigh',
                     expr: |||
                       avg by (job, instance)
                       (
                       (
                         increase(apache_response_http_codes_bucket{le=~"499"}[5m])
                       - ignoring(le)
                         increase(apache_response_http_codes_bucket{le=~"399"}[5m])
                       )
                       /
                       increase(apache_response_http_codes_count{}[5m]) * 100
                       )
                       > %(alertsCriticalErrorsRate)s
                       unless 
                       # at least 100 calls
                       increase(apache_accesses_total{}[5m]) > 100
                     ||| % $._config,
                     'for': '5m',
                     labels: {
                       severity: 'critical',
                     },
                     annotations: {
                       summary: 'Apache errors rate is too high.',
                       description: |||
                         Apache errors rate (4xx and 5xx HTTP codes) is above the threshold of %(alertsCriticalErrorsRate)s%% on {{ $labels.instance }}.
                         The currect value is {{ $value }}%%.
                       ||| % $._config,
                     },
                   },
                 ] else [],
      },
    ],
  },
}

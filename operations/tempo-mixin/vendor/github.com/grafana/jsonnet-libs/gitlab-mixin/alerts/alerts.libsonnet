{
  prometheusAlerts+:: {
    groups+: [
      {
        name: 'GitLabAlerts',
        rules: [
          {
            alert: 'GitLabHighJobRegistrationFailures',
            expr: |||
              100 * rate(job_register_attempts_failed_total{}[5m]) / rate(job_register_attempts_total{}[5m]) 
              > %(alertsWarningRegistrationFailures)s
            ||| % $._config,
            'for': '5m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              summary: 'Large percentage of failed attempts to register a job.',
              description:
                ('{{ printf "%%.2f" $value }}%% of job registrations have failed on {{$labels.instance}}, ' +
                 'which is above threshold of %(alertsWarningRegistrationFailures)s%%.') % $._config,
            },
          },
          {
            alert: 'GitLabHighRunnerAuthFailure',
            expr: |||
              100 * sum by (instance) (rate(gitlab_ci_runner_authentication_failure_total{}[5m]))  / 
              (sum by (instance) (rate(gitlab_ci_runner_authentication_success_total{}[5m]))  + sum by (instance) (rate(gitlab_ci_runner_authentication_failure_total{}[5m])))
              > %(alertsWarningRunnerAuthFailures)s
            ||| % $._config,
            'for': '5m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              summary: 'Large percentage of runner authentication failures.',
              description:
                ('{{ printf "%%.2f" $value }}%% of GitLab runner authentication attempts are failing on {{$labels.instance}}, ' +
                 'which is above the threshold of %(alertsWarningRunnerAuthFailures)s%%.') % $._config,
            },
          },
          {
            alert: 'GitLabHigh5xxResponses',
            expr: |||
              100 * sum by (instance) (rate(http_requests_total{status=~"^5.*"}[5m])) / sum by (instance) (rate(http_requests_total{}[5m])) 
              > %(alertsCritical5xxResponses)s
            ||| % $._config,
            'for': '5m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              summary: 'Large rate of HTTP 5XX errors.',
              description:
                ('{{ printf "%%.2f" $value }}%% of all requests returned 5XX HTTP responses, ' +
                 'which is above the threshold %(alertsCritical5xxResponses)s%%, ' +
                 'indicating a system issue on {{$labels.instance}}.') % $._config,
            },
          },
          {
            alert: 'GitLabHigh4xxResponses',
            expr: |||
              100 * sum by (instance) (rate(http_requests_total{status=~"^4.*"}[5m])) / sum by (instance) (rate(http_requests_total{}[5m]))
              > %(alertsWarning4xxResponses)s
            ||| % $._config,
            'for': '5m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              summary: 'Large rate of HTTP 4XX errors.',
              description:
                ('{{ printf "%%.2f" $value }}%% of all requests returned 4XX HTTP responses, ' +
                 'which is above the threshold %(alertsWarning4xxResponses)s%%, ' +
                 'indicating many failed requests on {{$labels.instance}}.') % $._config,
            },
          },
        ],
      },
    ],
  },
}

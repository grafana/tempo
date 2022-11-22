{
  _config+:: {
    dashboardTags: ['gitlab-mixin'],
    dashboardPeriod: 'now-1h',
    dashboardTimezone: 'default',
    dashboardRefresh: '1m',
    dashboardRailsExceptionFilename: '/var/log/gitlab/gitlab-rails/exceptions_json.log',

    // for alerts
    alertsWarningRegistrationFailures: '10',  // %
    alertsWarningRunnerAuthFailures: '10',  // %
    alertsCritical5xxResponses: '10',  // %
    alertsWarning4xxResponses: '10',  // %

    // enable Loki logs
    enableLokiLogs: true,
  },
}

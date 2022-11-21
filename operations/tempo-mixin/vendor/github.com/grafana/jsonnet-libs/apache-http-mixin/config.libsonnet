{
  _config+:: {
    dashboardTags: ['apache-http-mixin'],
    dashboardPeriod: 'now-1h',
    dashboardTimezone: 'default',
    dashboardRefresh: '1m',

    // for alerts
    alertsWarningWorkersBusy: '80',  // %
    alertsWarningResponseTimeMs: '5000',  // ms
    alertsCriticalErrorsRate: '20',  // ratio of 4xx and 5xx responses to all calls

    // enable Loki logs
    enableLokiLogs: false,
  },
}

{
  _config+:: {
    dashboardTags: ['discourse-mixin'],
    dashboardPeriod: 'now-1h',
    dashboardTimezone: 'default',
    dashboardRefresh: '1m',

    // for alerts
    alertsCritical5xxResponses: '10',  // %
    alertsWarning4xxResponses: '30',  // %
  },
}

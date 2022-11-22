{
  local makeGroupBy(groups) = std.join(', ', groups),

  _config+:: {
    dashboardTags: ['grafana-agent'],
    dashboardPeriod: 'now-1h',
    dashboardRefresh: '1m',
    dashboardTimezone: 'default',
  },
}

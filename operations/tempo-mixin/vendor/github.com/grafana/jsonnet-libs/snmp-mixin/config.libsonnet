{
  local makeGroupBy(groups) = std.join(', ', groups),

  _config+:: {
    dashboardTags: ['snmp'],
    dashboardPeriod: 'now-1h',
    dashboardRefresh: '1m',
    dashboardTimezone: 'default',
  },
}

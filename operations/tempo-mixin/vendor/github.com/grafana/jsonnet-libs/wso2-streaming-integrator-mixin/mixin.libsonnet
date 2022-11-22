{
  grafanaDashboards: {
    'Siddhi_overall.json': (import 'dashboards/Siddhi_overall.json'),
    'Siddhi_server.json': (import 'dashboards/Siddhi_server.json'),
    'Siddhi_query.json': (import 'dashboards/Siddhi_query.json'),
    'Siddhi_source.json': (import 'dashboards/Siddhi_source.json'),
    'Siddhi_sink.json': (import 'dashboards/Siddhi_sink.json'),
    'Siddhi_table.json': (import 'dashboards/Siddhi_table.json'),
    'Siddhi_aggregation.json': (import 'dashboards/Siddhi_aggregation.json'),
    'Siddhi_ondemandquery.json': (import 'dashboards/Siddhi_ondemandquery.json'),
    'Siddhi_stream.json': (import 'dashboards/Siddhi_stream.json'),
    'StreamingIntegrator_apps.json': (import 'dashboards/StreamingIntegrator_apps.json'),
    'StreamingIntegrator_overall.json': (import 'dashboards/StreamingIntegrator_overall.json'),
  },
  local importRules(rules) = {
    groups+: std.native('parseYaml')(rules)[0].groups,
  },

}

local dashboards = import 'dashboards.libsonnet';
local datasources = import 'datasources.libsonnet';
local grafana = import 'grafana/grafana.libsonnet';
local mixins = import 'mixins.libsonnet';

{
  deploy(frontend_url='http://query-frontend'):
    grafana
    + grafana.withReplicas(1)
    + grafana.withImage('grafana/grafana:11.4.0')
    + grafana.withRootUrl('http://grafana')
    + grafana.withTheme('dark')
    + grafana.withAnonymous()

    + grafana.withGrafanaIniConfig({
      sections+: {
        feature_toggles: {
          enable: 'traceqlEditor',
        },
      },
    })

    + grafana.addDatasource('Tempo', datasources.tempo(frontend_url))
    + grafana.addDatasource('Prometheus', datasources.prometheus)

    + grafana.addMixinDashboards(mixins),
}

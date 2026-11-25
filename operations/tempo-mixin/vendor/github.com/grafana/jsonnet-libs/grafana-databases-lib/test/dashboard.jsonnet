// Example dashboard (dashboard.grafana.app/v2) built from the library, so it
// can be rendered and checked end to end. It shows the two ways to use the
// library in a dashboard: whole rows built by the library, and a custom row
// assembled from its named panels. test_legacy.libsonnet shows the
// grafana-builder path.
local g = import '../g.libsonnet';
local dblib = import '../main.libsonnet';

local databases =
  dblib.new()
  + dblib.autoscaling.withConfigMixin({
    components: {
      distributor: { title: 'Distributor', hpaName: 'keda-hpa-distributor', keda: false },
      querier: { title: 'Querier', hpaName: 'keda-hpa-querier.*' },
      ruler: {
        title: 'Ruler',
        hpaName: 'keda-hpa-ruler',
        scalers: { cpu: { title: 'CPU' }, memory: { title: 'memory' } },
      },
    },
  })
  // Overriding a named panel changes it in its row too.
  + dblib.autoscaling.withPanelMixin('querier', 'replicas', { spec+: { title: 'Querier replicas' } });

local rows = databases.autoscaling.rows;
local panels = databases.autoscaling.panels;

local dashboardRows = [
  // Whole rows built by the library.
  rows.distributor,
  rows.querier,
  rows.ruler,
  // A row the library doesn't build: the ruler's replicas and failures only,
  // from its named panels.
  dblib.row('Ruler - replicas and failures', [panels.ruler.replicas, panels.ruler.failures]),
];

// The variables the library's queries use: $datasource, $cluster and $namespace.
local option(value) = { text: value, value: value };
local variable(kind, spec) = {
  kind: kind,
  spec: {
    current: option(''),
    options: [],
    multi: false,
    includeAll: false,
    hide: 'dontHide',
    skipUrlSync: false,
    allowCustomValue: true,
  } + spec,
};

local dash = g.apps.dashboard.v2;
dash.new('grafana-databases-lib-test', 'grafana-databases-lib test')
+ dash.spec.withElements(dblib.elements(dashboardRows))
+ dash.spec.withLayout(dblib.layout(dashboardRows))
+ dash.spec.withVariables([
  variable('DatasourceVariable', { name: 'datasource', pluginId: 'prometheus', refresh: 'onDashboardLoad', regex: '' }),
  variable('CustomVariable', { name: 'cluster', query: '' }),
  variable('CustomVariable', { name: 'namespace', query: '' }),
])
+ dash.spec.withAnnotations([])
+ dash.spec.withLinks([])
+ dash.spec.withTags([])
+ dash.spec.withCursorSync('Off')
+ dash.spec.withPreload(false)
+ dash.spec.withTimeSettings({
  from: 'now-6h',
  to: 'now',
  autoRefresh: '',
  autoRefreshIntervals: ['5m', '15m', '1h'],
  hideTimepicker: false,
  fiscalYearStartMonth: 0,
})

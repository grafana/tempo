// Tests for the legacy helpers, which convert the library's panels and rows
// for dashboards still built with grafana-builder.
local test = import 'github.com/jsonnet-libs/testonnet/main.libsonnet';
local builder = import 'grafana-builder/grafana.libsonnet';
local dblib = import 'grafana-databases-lib/main.libsonnet';

local databases =
  dblib.new()
  + dblib.autoscaling.withConfigMixin({
    components: { querier: { title: 'Querier', hpaName: 'keda-hpa-querier.*' } },
  });

// A whole library row added to a grafana-builder row.
local wholeRow = std.foldl(
  function(row, panel) row.addPanel(panel),
  dblib.legacy.rowPanels(databases.autoscaling.rows.querier),
  builder.row('Querier - autoscaling'),
);

// A grafana-builder row mixing a library panel with one of its own.
local mixedRow =
  builder.row('Querier')
  .addPanel(dblib.legacy.panel(databases.autoscaling.panels.querier.replicas))
  .addPanel(builder.panel('Own panel'));

local classic = dblib.legacy.panel(databases.autoscaling.panels.querier.failures);

test.new(std.thisFile)

+ test.case.new(
  name='legacy.panel returns a classic panel',
  test=test.expect.eq(
    actual={
      type: classic.type,
      datasource: classic.datasource,
      target: std.objectFields(classic.targets[0]),
      fieldConfig: std.objectFields(classic.fieldConfig),
    },
    expected={
      type: 'timeseries',
      datasource: { type: 'prometheus', uid: '$datasource' },
      // Classic panels leave refIds to Grafana; empty overrides are left out.
      target: ['datasource', 'expr', 'legendFormat'],
      fieldConfig: ['defaults'],
    },
  ),
)

+ test.case.new(
  name='legacy.panel hides the gridPos',
  test=test.expect.eq(
    actual=std.objectHas(dblib.legacy.panel(databases.autoscaling.panels.querier.replicas), 'gridPos'),
    expected=false,
  ),
)
+ test.case.new(
  name='A whole library row in a grafana-builder row',
  test=test.expect.eq(
    actual=[[p.title, p.span, std.objectHas(p, 'gridPos')] for p in wholeRow.panels],
    expected=[
      ['Replicas', 4, false],
      ['Desired replicas', 4, false],
      ['Autoscaler failures rate', 4, false],
    ],
  ),
)
+ test.case.new(
  name='Library and grafana-builder panels in one row',
  test=test.expect.eq(
    actual=[[p.title, p.span] for p in mixedRow.panels],
    expected=[['Replicas', 6], ['Own panel', 6]],
  ),
)

// Tests for the autoscaling area: the panels and rows the library builds,
// custom rows from its named panels, overrides and config mixins.
local test = import 'github.com/jsonnet-libs/testonnet/main.libsonnet';
local dblib = import 'grafana-databases-lib/main.libsonnet';

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
  });

local panels = databases.autoscaling.panels;
local rows = databases.autoscaling.rows;

local titles(row) = [p.spec.title for p in row.panels];
local queries(panel) = [q.spec.query.spec for q in panel.spec.data.spec.queries];
local defaults(panel) = panel.spec.vizConfig.spec.fieldConfig.defaults;

test.new(std.thisFile)

// Whole rows built by the library.
+ test.case.new(
  name='Row title',
  test=test.expect.eq(actual=rows.querier.spec.title, expected='Querier - autoscaling'),
)
+ test.case.new(
  name='Row of a KEDA component',
  test=test.expect.eq(
    actual=titles(rows.querier),
    expected=['Replicas', 'Desired replicas', 'Autoscaler failures rate'],
  ),
)
+ test.case.new(
  name='Row of a component without KEDA triggers',
  test=test.expect.eq(actual=titles(rows.distributor), expected=['Replicas']),
)
+ test.case.new(
  name='Row of a component with scalers',
  test=test.expect.eq(
    actual=titles(rows.ruler),
    expected=[
      'Replicas',
      'Scaling metric (CPU): Desired replicas',
      'Scaling metric (memory): Desired replicas',
      'Autoscaler failures rate',
    ],
  ),
)
+ test.case.new(
  name='Row layout refers to its panels by element name',
  test=test.expect.eq(
    actual=[i.spec.element.name for i in rows.ruler.spec.layout.spec.items],
    expected=[
      'autoscaling-ruler-replicas',
      'autoscaling-ruler-desired-replicas-cpu',
      'autoscaling-ruler-desired-replicas-memory',
      'autoscaling-ruler-failures',
    ],
  ),
)

// Named panels, and the way overrides reach the rows that use them.
+ test.case.new(
  name='Overriding a named panel changes its row',
  test=test.expect.eq(
    actual=titles((databases { autoscaling+: { panels+: { querier+: { replicas+: { spec+: { title: 'X' } } } } } }).autoscaling.rows.querier)[0],
    expected='X',
  ),
)
+ test.case.new(
  name='withPanelMixin patches a named panel and its row',
  test=test.expect.eq(
    actual=(
      local patched = databases + dblib.autoscaling.withPanelMixin('querier', 'replicas', { spec+: { title: 'X' } });
      [patched.autoscaling.panels.querier.replicas.spec.title, titles(patched.autoscaling.rows.querier)[0]]
    ),
    expected=['X', 'X'],
  ),
)
+ test.case.new(
  name='Overriding a row leaves its panels alone',
  test=test.expect.eq(
    actual=(databases { autoscaling+: { rows+: { querier+: { spec+: { title: 'Y' } } } } }).autoscaling.panels.querier.replicas.spec.title,
    expected='Replicas',
  ),
)

// Config mixins apply wherever they are added.
+ test.case.new(
  name='new() takes the library-wide config',
  test=test.expect.eq(
    actual=std.length(std.findSubstr(
      'k8s_cluster=~"$cluster"',
      queries((dblib.new({ clusterLabel: 'k8s_cluster' }) + dblib.autoscaling.withConfigMixin({
                 components: { querier: { hpaName: 'keda-hpa-querier' } },
               })).autoscaling.panels.querier.replicas)[0].expr,
    )) > 0,
    expected=true,
  ),
)
+ test.case.new(
  name='A config mixin added last re-derives the selectors',
  test=test.expect.eq(
    actual=std.length(std.findSubstr(
      'k8s_cluster=~"$cluster", namespace=~"$namespace"',
      queries((databases + dblib.withConfigMixin({ clusterLabel: 'k8s_cluster' })).autoscaling.panels.querier.replicas)[0].expr,
    )) > 0,
    expected=true,
  ),
)
+ test.case.new(
  name='Trigger legend',
  test=test.expect.eq(
    actual=[
      queries(panels.querier.desiredReplicas)[0].legendFormat,
      queries((databases + dblib.autoscaling.withConfigMixin({ triggerLegend: '{{ scaler }}' })).autoscaling.panels.querier.failures)[0].legendFormat,
    ],
    expected=['{{ scaledObject }} {{ metric }}', '{{ scaler }} failures'],
  ),
)

// Empty-state text and axis.
+ test.case.new(
  name='No-value text',
  test=test.expect.eq(
    actual=[
      defaults(panels.querier.replicas).noValue,
      defaults(panels.ruler.desiredReplicasByScaler.memory).noValue,
      std.objectHas(defaults((databases + dblib.autoscaling.withConfigMixin({ noValue: null })).autoscaling.panels.querier.replicas), 'noValue'),
    ],
    expected=['Not auto-scaled', 'Not scaled on memory', false],
  ),
)
+ test.case.new(
  name='Y-axis minimum',
  test=test.expect.eq(
    actual=[
      std.objectHas(defaults(panels.querier.replicas), 'min'),
      defaults((databases + dblib.autoscaling.withConfigMixin({ axisMin: 0 })).autoscaling.panels.querier.replicas).min,
    ],
    expected=[false, 0],
  ),
)

// Extra replica series.
+ test.case.new(
  name='Extra replica queries',
  test=test.expect.eq(
    actual=[
      q.legendFormat
      for q in queries((databases + dblib.autoscaling.withConfigMixin({
                          components+: { querier+: { extraReplicaQueries: [{ expr: 'up', legend: 'Extra' }] } },
                        })).autoscaling.panels.querier.replicas)
    ],
    expected=[
      'Min {{ scaletargetref_name }}',
      'Current {{ scaletargetref_name }}',
      'Desired {{ scaletargetref_name }}',
      'Max {{ scaletargetref_name }}',
      'Extra',
    ],
  ),
)

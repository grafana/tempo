// Tests for the building blocks: panel elements, queries, rows, and the
// elements and layout of a dashboard assembled from rows.
local test = import 'github.com/jsonnet-libs/testonnet/main.libsonnet';
local dblib = import 'grafana-databases-lib/main.libsonnet';


local a = dblib.panel('a', 'timeseries', 'A', queries=[
  dblib.query('prometheus', '$datasource', { expr: 'up' }),
  dblib.query('prometheus', '$datasource', { expr: 'down' }),
]);
local b = dblib.panel('b', 'stat', 'B');
local rows = [dblib.row('First', [a, b]), dblib.row('Second', [b])];

test.new(std.thisFile)

+ test.case.new(
  name='Panel element',
  test=test.expect.eq(
    actual=[a.kind, a.spec.vizConfig.kind, a.spec.vizConfig.group, a.spec.data.kind, a.spec.vizConfig.spec.fieldConfig.overrides],
    expected=['Panel', 'VizConfig', 'timeseries', 'QueryGroup', []],
  ),
)
+ test.case.new(
  name='Queries get refIds in order',
  test=test.expect.eq(
    actual=[[q.spec.refId, q.spec.query.kind, q.spec.query.group, q.spec.query.datasource.name] for q in a.spec.data.spec.queries],
    expected=[['A', 'DataQuery', 'prometheus', '$datasource'], ['B', 'DataQuery', 'prometheus', '$datasource']],
  ),
)
+ test.case.new(
  name='withViz merges into the visualization config',
  test=test.expect.eq(
    actual=(a + dblib.withViz({ fieldConfig+: { defaults+: { unit: 's' } } })).spec.vizConfig.spec.fieldConfig,
    expected={ defaults: { unit: 's' }, overrides: [] },
  ),
)
+ test.case.new(
  name='Row columns default to its panel count, up to 4',
  test=test.expect.eq(
    actual=[r.spec.layout.spec.maxColumnCount for r in rows] + [dblib.row('Wide', [a, a, a, a, a]).spec.layout.spec.maxColumnCount],
    expected=[2, 1, 4],
  ),
)
+ test.case.new(
  name='A panel in several rows is one element, with unique ids',
  test=test.expect.eq(
    actual={ [name]: dblib.elements(rows)[name].spec.id for name in std.objectFields(dblib.elements(rows)) },
    expected={ a: 1, b: 2 },
  ),
)
+ test.case.new(
  name='Every layout reference has an element',
  test=test.expect.eq(
    actual=std.all([
      std.objectHas(dblib.elements(rows), item.spec.element.name)
      for row in dblib.layout(rows).spec.rows
      for item in row.spec.layout.spec.items
    ]),
    expected=true,
  ),
)
+ test.case.new(
  name='Changing a row panels changes its layout',
  test=test.expect.eq(
    actual=[i.spec.element.name for i in (dblib.row('Third', [a]) { panels:: [b, a] }).spec.layout.spec.items],
    expected=['b', 'a'],
  ),
)

local d = import 'github.com/jsonnet-libs/docsonnet/doc-util/main.libsonnet';

{
  '#new':: d.func.new(
    'Creates a new row panel with a title.',
    args=[d.arg('title', d.T.string)]
  ),
  new(title):
    self.withTitle(title)
    + self.withType()
    + self.withCollapsed(false)
    + self.gridPos.withX(0)
    + self.gridPos.withH(1)
    + self.gridPos.withW(24),

  '#gridPos':: {},  // use withGridPos instead
  '#withGridPos':: d.func.new(
    '`withGridPos` sets the Y-axis on a row panel. x, width and height are fixed values.',
    args=[d.arg('y', d.T.number)]
  ),
  withGridPos(y):
    self.gridPos.withX(0)
    + self.gridPos.withY(y)
    + self.gridPos.withH(1)
    + self.gridPos.withW(24),
}

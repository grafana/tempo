local util = import './util/main.libsonnet';
local d = import 'github.com/jsonnet-libs/docsonnet/doc-util/main.libsonnet';

{
  '#new':: d.func.new(
    'Creates a new dashboard with a title.',
    args=[d.arg('title', d.T.string)]
  ),
  new(title):
    self.withTitle(title)
    + self.withSchemaVersion()
    + self.withTimezone('utc')
    + self.time.withFrom('now-6h')
    + self.time.withTo('now'),

  '#withSchemaVersion': { 'function'+: { args: [d.arg('value', d.T.integer, default=39)] } },
  withSchemaVersion(value=39): {
    schemaVersion: value,
  },

  '#withPanels':: d.func.new(
    '`withPanels` sets the panels on a dashboard authoratively. It automatically adds IDs to the panels, this can be disabled with `setPanelIDs=false`.',
    args=[
      d.arg('panels', d.T.array),
      d.arg('setPanelIDs', d.T.bool, default=true),
    ]
  ),
  withPanels(panels, setPanelIDs=true): {
    _panels:: if std.isArray(panels) then panels else [panels],
    panels:
      if setPanelIDs
      then util.panel.setPanelIDs(self._panels)
      else self._panels,
  },
  '#withPanelsMixin':: d.func.new(
    '`withPanelsMixin` adds more panels to a dashboard.',
    args=[
      d.arg('panels', d.T.array),
      d.arg('setPanelIDs', d.T.bool, default=true),
    ]
  ),
  withPanelsMixin(panels, setPanelIDs=true): {
    _panels+:: if std.isArray(panels) then panels else [panels],
    panels:
      if setPanelIDs
      then util.panel.setPanelIDs(self._panels)
      else self._panels,
  },

  graphTooltip+: {
    // 0 - Default
    // 1 - Shared crosshair
    // 2 - Shared tooltip
    '#withSharedCrosshair':: d.func.new(
      'Share crosshair on all panels.',
    ),
    withSharedCrosshair():
      { graphTooltip: 1 },

    '#withSharedTooltip':: d.func.new(
      'Share crosshair and tooltip on all panels.',
    ),
    withSharedTooltip():
      { graphTooltip: 2 },
  },
}
+ (import './dashboard/annotation.libsonnet')
+ (import './dashboard/link.libsonnet')
+ (import './dashboard/variable.libsonnet')

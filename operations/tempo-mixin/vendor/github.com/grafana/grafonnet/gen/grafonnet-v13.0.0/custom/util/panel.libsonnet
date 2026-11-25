local d = import 'github.com/jsonnet-libs/docsonnet/doc-util/main.libsonnet';
local xtd = import 'github.com/jsonnet-libs/xtd/main.libsonnet';

{
  local this = self,

  // used in ../dashboard.libsonnet
  '#setPanelIDs':: d.func.new(
    |||
      `setPanelIDs` ensures that all `panels` have a unique ID, this function is used in `dashboard.withPanels` and `dashboard.withPanelsMixin` to provide a consistent experience.

      `overrideExistingIDs` can be set to not replace existing IDs, consider validating the IDs with `validatePanelIDs()` to ensure there are no duplicate IDs.
    |||,
    args=[
      d.arg('panels', d.T.array),
      d.arg('overrideExistingIDs', d.T.bool, default=true),
    ]
  ),
  setPanelIDs(panels, overrideExistingIDs=true):
    local infunc(panels, start=1) =
      std.foldl(
        function(acc, panel)
          acc + {
            index:  // Track the index to ensure no duplicates exist.
              acc.index
              + 1
              + (if panel.type == 'row'
                    && 'panels' in panel
                 then std.length(panel.panels)
                 else 0),

            panels+: [
              panel
              + (
                if overrideExistingIDs
                   || std.get(panel, 'id', null) == null
                then { id: acc.index }
                else {}
              )
              + (
                if panel.type == 'row'
                   && 'panels' in panel
                then {
                  panels:
                    infunc(
                      panel.panels,
                      acc.index + 1
                    ),
                }
                else {}
              ),
            ],
          },
        panels,
        { index: start, panels: [] }
      ).panels;
    infunc(panels),

  '#getPanelIDs':: d.func.new(
    |||
      `getPanelIDs` returns an array with all panel IDs including IDs from panels in rows.
    |||,
    args=[
      d.arg('panels', d.T.array),
    ]
  ),
  getPanelIDs(panels):
    std.flattenArrays(
      std.map(
        function(panel)
          [panel.id]
          + (if panel.type == 'row'
             then this.getPanelIDs(std.get(panel, 'panels', []))
             else []),
        panels
      )
    ),

  '#validatePanelIDs':: d.func.new(
    |||
      `validatePanelIDs` validates returns `false` if there are duplicate panel IDs in `panels`.
    |||,
    args=[
      d.arg('panels', d.T.array),
    ]
  ),
  validatePanelIDs(panels):
    local ids = this.getPanelIDs(panels);
    std.set(ids) == std.sort(ids),

  '#sanitizePanel':: d.func.new(
    |||
      `sanitizePanel` ensures the panel has a valid `gridPos` and row panels have `collapsed` and `panels`. This function is recursively applied to panels inside row panels.

      The default values for x,y,h,w are only applied if not already set.
    |||,
    [
      d.arg('panel', d.T.object),
      d.arg('defaultX', d.T.number, default=0),
      d.arg('defaultY', d.T.number, default=0),
      d.arg('defaultHeight', d.T.number, default=8),
      d.arg('defaultWidth', d.T.number, default=8),
    ]
  ),
  sanitizePanel(panel, defaultX=0, defaultY=0, defaultHeight=8, defaultWidth=8):
    local infunc(panel) =
      panel
      + (
        local gridPos = std.get(panel, 'gridPos', {});
        if panel.type == 'row'
        then {
          collapsed: std.get(panel, 'collapsed', false),
          panels: std.map(infunc, std.get(panel, 'panels', [])),
          gridPos: {  // x, h, w are fixed
            x: 0,
            y: std.get(gridPos, 'y', defaultY),
            h: 1,
            w: 24,
          },
        }
        else {
          gridPos: {
            x: std.get(gridPos, 'x', defaultX),
            y: std.get(gridPos, 'y', defaultY),
            h: std.get(gridPos, 'h', defaultHeight),
            w: std.get(gridPos, 'w', defaultWidth),
          },
        }
      );
    infunc(panel),

  '#sortPanelsByXY':: d.func.new(
    |||
      `sortPanelsByXY` applies a simple sorting algorithm, first by x then again by y. This does not take width and height into account.
    |||,
    [
      d.arg('panels', d.T.array),
    ]
  ),
  sortPanelsByXY(panels):
    std.sort(
      std.sort(
        panels,
        function(panel)
          panel.gridPos.x
      ),
      function(panel)
        panel.gridPos.y
    ),

  '#sortPanelsInRow':: d.func.new(
    |||
      `sortPanelsInRow` applies `sortPanelsByXY` on the panels in a rowPanel.
    |||,
    [
      d.arg('rowPanel', d.T.object),
    ]
  ),
  sortPanelsInRow(rowPanel):
    rowPanel + { panels: this.sortPanelsByXY(rowPanel.panels) },

  '#groupPanelsInRows':: d.func.new(
    |||
      `groupPanelsInRows` ensures that panels that come after a row panel in an array are added to the `row.panels` attribute. This can be useful to apply intermediate functions to only the panels that belong to a row. Finally the panel array should get processed by `resolveCollapsedFlagOnRows` to "unfold" the rows that are not collapsed into the main array.
    |||,
    [
      d.arg('panels', d.T.array),
    ]
  ),
  groupPanelsInRows(panels):
    // Add panels that come after a row to row.panels
    local grouped =
      xtd.array.filterMapWithIndex(
        function(i, p) p.type == 'row',
        function(i, p)
          p + {
            panels+:
              this.getPanelsBeforeNextRow(panels[i + 1:]),
          },
        panels,
      );

    // Get panels that come before the rowGroups
    local panelsBeforeRowGroups = this.getPanelsBeforeNextRow(panels);

    panelsBeforeRowGroups + grouped,

  '#getPanelsBeforeNextRow':: d.func.new(
    |||
      `getPanelsBeforeNextRow` returns all panels in an array up until a row has been found. Used in `groupPanelsInRows`.
    |||,
    [
      d.arg('panels', d.T.array),
    ]
  ),
  getPanelsBeforeNextRow(panels):
    local rowIndexes =
      xtd.array.filterMapWithIndex(
        function(i, p) p.type == 'row',
        function(i, p) i,
        panels,
      );
    if std.length(rowIndexes) != 0
    then panels[0:rowIndexes[0]]
    else panels[0:],  // if no row panels found, return all remaining panels

  '#resolveCollapsedFlagOnRows':: d.func.new(
    |||
      `resolveCollapsedFlagOnRows` should be applied to the final panel array to "unfold" the rows that are not collapsed into the main array.
    |||,
    [
      d.arg('panels', d.T.array),
    ]
  ),
  resolveCollapsedFlagOnRows(panels):
    std.foldl(
      function(acc, panel)
        acc + (
          if panel.type == 'row'
             && !panel.collapsed
          then  // If not collapsed, then move panels to main array below the row panel
            [panel + { panels: [] }]
            + panel.panels
          else [panel]
        ),
      panels,
      [],
    ),

  '#normalizeY':: d.func.new(
    |||
      `normalizeY` applies negative gravity on the inverted Y axis. This mimics the behavior of Grafana: when a panel is created without panel above it, then it'll float upward.

      This is strictly not required as Grafana will do this on dashboard load, however it might be helpful when used when calculating the correct `gridPos`.
    |||,
    [
      d.arg('panels', d.T.array),
    ]
  ),
  normalizeY(panels):
    std.foldl(
      function(acc, i)
        acc + [
          panels[i] + {
            gridPos+: {
              y: this.calculateLowestYforPanel(panels[i], acc),
            },
          },
        ],
      std.range(0, std.length(panels) - 1),
      []
    ),

  '#calculateLowestYforPanel':: d.func.new(
    |||
      `calculateLowestYforPanel` calculates Y for a given `panel` from the `gridPos` of an array of `panels`. This function is used in `normalizeY`.
    |||,
    [
      d.arg('panel', d.T.object),
      d.arg('panels', d.T.array),
    ]
  ),
  calculateLowestYforPanel(panel, panels):
    xtd.number.maxInArray(  // the new position is highest value (max) on the Y-scale
      std.filterMap(
        function(p)  // find panels that overlap on X-scale
          local v1 = panel.gridPos.x;
          local v2 = panel.gridPos.x + panel.gridPos.w;
          local x1 = p.gridPos.x;
          local x2 = p.gridPos.x + p.gridPos.w;
          (v1 >= x1 && v1 < x2)
          || (v2 >= x1 && v2 < x2),
        function(p)  // return new position on Y-scale
          p.gridPos.y + p.gridPos.h,
        panels,
      ),
    ),

  '#normalizeYInRow':: d.func.new(
    |||
      `normalizeYInRow` applies `normalizeY` to the panels in a row panel.
    |||,
    [
      d.arg('rowPanel', d.T.object),
    ]
  ),
  normalizeYInRow(rowPanel):
    rowPanel + {
      panels:
        std.map(
          function(p)
            p + {
              gridPos+: {
                y:  // Increase panel Y with the row Y to put them below the row when not collapsed.
                  p.gridPos.y
                  + rowPanel.gridPos.y
                  + rowPanel.gridPos.h,
              },
            },
          this.normalizeY(rowPanel.panels)
        ),
    },

  '#mapToRows':: d.func.new(
    |||
      `mapToRows` is a little helper function that applies `func` to all row panels in an array. Other panels in that array are returned ad verbatim.
    |||,
    [
      d.arg('func', d.T.func),
      d.arg('panels', d.T.array),
    ]
  ),
  mapToRows(func, panels):
    std.map(
      function(p)
        if p.type == 'row'
        then func(p)
        else p,
      panels
    ),


  '#setRefIDs':: d.func.new(
    |||
      `setRefIDs` calculates the `refId` field for each target on a panel.
    |||,
    args=[
      d.arg('panel', d.T.object),
      d.arg('overrideExistingIDs', d.T.bool, default=true),
    ]
  ),
  setRefIDs(panel, overrideExistingIDs=true):
    local calculateRefID(n) =
      // From: https://github.com/grafana/grafana/blob/bffd87107b786930edd091060143ee013843efac/packages/grafana-data/src/query/refId.ts#L15
      local letters = std.map(std.char, std.range(std.codepoint('A'), std.codepoint('Z')));
      if n < std.length(letters)
      then letters[n]
      else calculateRefID(std.floor(n / std.length(letters)) - 1) + letters[std.mod(n, std.length(letters))];
    panel + {
      targets:
        std.mapWithIndex(
          function(i, target)
            if overrideExistingIDs
               || !std.objectHas(target, 'refId')
            then target + {
              refId: calculateRefID(i),
            }
            else target,
          panel.targets,
        ),
    },

  '#setRefIDsOnPanels':: d.func.new(
    |||
      `setRefIDsOnPanels` applies `setRefIDs on all `panels`.
    |||,
    args=[
      d.arg('panels', d.T.array),
    ]
  ),
  setRefIDsOnPanels(panels):
    std.map(self.setRefIDs, panels),

  '#dedupeQueryTargets':: d.func.new(
    |||
      `dedupeQueryTargets` dedupes the query targets in a set of panels and replaces the duplicates with a ['shared query'](https://grafana.com/docs/grafana/latest/panels-visualizations/query-transform-data/share-query/). Sharing query results across panels reduces the number of queries made to your data source, which can improve the performance of your dashboard.

      This function requires that the query targets have `refId` set, `setRefIDs` and `setRefIDsOnPanels` can help with that.
    |||,
    args=[
      d.arg('panels', d.T.array),
    ]
  ),
  dedupeQueryTargets(panels):
    // Hide ref so it doesn't compare in equality
    local targetWithoutRef(target) =
      target + { refId:: target.refId };

    // Find targets that are the same
    local findTargets(targets, target) =
      std.filter(
        function(t)
          targetWithoutRef(t) == targetWithoutRef(target),
        targets
      );

    // Get a flat array of all targets including their panelId
    local targets = std.flattenArrays([
      std.map(function(t) t + { panelId:: panel.id }, panel.targets)
      for panel in panels
    ]);

    std.map(
      function(panel)
        // Replace target with 'shared query' target if found in other panels
        local replaceTarget(target) =
          local found = findTargets(targets, target);
          if std.length(found) > 0
             // Do not reference queries from the same panel
             && found[0].panelId != panel.id
          then {
            datasource: {
              type: 'datasource',
              uid: '-- Dashboard --',
            },
            refId: found[0].refId,
            panelId: found[0].panelId,
          }
          else target;

        panel + {
          targets:
            std.map(
              replaceTarget,
              panel.targets,
            ),
        },
      panels
    ),
}

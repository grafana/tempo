local d = import 'github.com/jsonnet-libs/docsonnet/doc-util/main.libsonnet';
local xtd = import 'github.com/jsonnet-libs/xtd/main.libsonnet';

local panelUtil = import './panel.libsonnet';

{
  local root = self,

  local gridWidth = 24,

  '#makeGrid':: d.func.new(
    |||
      `makeGrid` returns an array of `panels` organized in a grid with equal `panelWidth`
      and `panelHeight`. Row panels are used as "linebreaks", if a Row panel is collapsed,
      then all panels below it will be folded into the row.

      This function will use the full grid of 24 columns, setting `panelWidth` to a value
      that can divide 24 into equal parts will fill up the page nicely. (1, 2, 3, 4, 6, 8, 12)
      Other value for `panelWidth` will leave a gap on the far right.

      Optional `startY` can be provided to place generated grid above or below existing panels.
    |||,
    args=[
      d.arg('panels', d.T.array),
      d.arg('panelWidth', d.T.number),
      d.arg('panelHeight', d.T.number),
      d.arg('startY', d.T.number),
    ],
  ),
  makeGrid(panels, panelWidth=8, panelHeight=8, startY=0):
    local sanitizePanels(ps) = std.map(
      function(p)
        local sanePanel = panelUtil.sanitizePanel(p);
        (
          if p.type == 'row'
          then sanePanel + {
            panels: sanitizePanels(sanePanel.panels),
          }
          else sanePanel + {
            gridPos+: {
              h: panelHeight,
              w: panelWidth,
            },
          }
        ),
      ps
    );

    local sanitizedPanels = sanitizePanels(panels);

    local grouped = panelUtil.groupPanelsInRows(sanitizedPanels);

    local panelsBeforeRows = panelUtil.getPanelsBeforeNextRow(grouped);
    local rowPanels =
      std.filter(
        function(p) p.type == 'row',
        grouped
      );

    local CalculateXforPanel(index, panel) =
      local panelsPerRow = std.floor(gridWidth / panelWidth);
      local col = std.mod(index, panelsPerRow);
      panel + { gridPos+: { x: panelWidth * col } };

    local panelsBeforeRowsWithX = std.mapWithIndex(CalculateXforPanel, panelsBeforeRows);

    local rowPanelsWithX =
      std.map(
        function(row)
          row + { panels: std.mapWithIndex(CalculateXforPanel, row.panels) },
        rowPanels
      );

    local uncollapsed = panelUtil.resolveCollapsedFlagOnRows(panelsBeforeRowsWithX + rowPanelsWithX);

    local normalized = panelUtil.normalizeY(uncollapsed);

    std.map(function(p) p + { gridPos+: { y+: startY } }, normalized),

  '#wrapPanels':: d.func.new(
    |||
      `wrapPanels` returns an array of `panels` organized in a grid, wrapping up to next 'row' if total width exceeds full grid of 24 columns.
      'panelHeight' and 'panelWidth' are used unless panels already have height and width defined.
    |||,
    args=[
      d.arg('panels', d.T.array),
      d.arg('panelWidth', d.T.number),
      d.arg('panelHeight', d.T.number),
      d.arg('startY', d.T.number),
    ],
  ),
  wrapPanels(panels, panelWidth=8, panelHeight=8, startY=0):

    local calculateGridPosForPanel(acc, panel) =
      local gridPos = std.get(panel, 'gridPos', {});
      local width = std.get(gridPos, 'w', panelWidth);
      local height = std.get(gridPos, 'h', panelHeight);
      if acc.cursor.x + width > gridWidth
      then
        // start new row as width exceeds gridWidth
        {
          panels+: [
            panel + {
              gridPos+:
                {
                  x: 0,
                  y: acc.cursor.y + height,
                  w: width,
                  h: height,
                },
            },
          ],
          cursor+:: {
            x: 0 + width,
            y: acc.cursor.y + height,
            maxH: if height > acc.cursor.maxH then height else acc.cursor.maxH,
          },
        }
      else
        // enough width, place panel on current row
        {
          panels+: [
            panel + {
              gridPos+:
                {
                  x: acc.cursor.x,
                  y: acc.cursor.y,
                  w: width,
                  h: height,
                },
            },
          ],
          cursor+:: {
            x: acc.cursor.x + width,
            y: acc.cursor.y,
            maxH: if height > acc.cursor.maxH then height else acc.cursor.maxH,
          },
        };

    std.foldl(
      function(acc, panel)
        if panel.type == 'row'
        then
          (
            if std.objectHas(panel, 'panels') && std.length(panel.panels) > 0
            then
              local rowPanels =
                std.foldl(
                  function(acc, panel)
                    acc + calculateGridPosForPanel(acc, panel),
                  panel.panels,
                  {
                    panels+: [],
                    // initial
                    cursor:: {
                      x: 0,
                      y: acc.cursor.y + acc.cursor.maxH + 1,
                      maxH: 0,
                    },
                  },
                );
              acc + {
                panels+: [
                  panel + {
                    //rows panels
                    panels: rowPanels.panels,
                    gridPos+: {
                      x: 0,
                      y: acc.cursor.y + acc.cursor.maxH,
                      w: 0,
                    },

                  },
                ],
                cursor:: rowPanels.cursor,
              }
            else
              acc + {
                panels+: [
                  panel + {
                    panels: [],
                    gridPos+:
                      {
                        x: acc.cursor.x,
                        y: acc.cursor.y + acc.cursor.maxH,
                        w: 0,
                        h: 1,
                      },
                  },
                ],
                cursor:: {
                  x: 0,
                  y: acc.cursor.y + acc.cursor.maxH + 1,
                  maxH: 0,
                },
              }
          )
        else
          // handle regular panel
          acc + calculateGridPosForPanel(acc, panel),
      panels,
      // Initial value for acc:
      {
        panels: [],
        cursor:: {
          x: 0,
          y: startY,
          maxH: 0,  // max height of current 'row'
        },
      }
    ).panels,
}

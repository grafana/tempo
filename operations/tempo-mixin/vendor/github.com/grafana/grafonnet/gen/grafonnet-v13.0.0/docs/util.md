# util

Helper functions that work well with Grafonnet.

## Index

* [`obj dashboard`](#obj-dashboard)
  * [`fn getOptionsForCustomQuery(query)`](#fn-dashboardgetoptionsforcustomquery)
* [`obj grid`](#obj-grid)
  * [`fn makeGrid(panels, panelWidth, panelHeight, startY)`](#fn-gridmakegrid)
  * [`fn wrapPanels(panels, panelWidth, panelHeight, startY)`](#fn-gridwrappanels)
* [`obj panel`](#obj-panel)
  * [`fn calculateLowestYforPanel(panel, panels)`](#fn-panelcalculatelowestyforpanel)
  * [`fn dedupeQueryTargets(panels)`](#fn-paneldedupequerytargets)
  * [`fn getPanelIDs(panels)`](#fn-panelgetpanelids)
  * [`fn getPanelsBeforeNextRow(panels)`](#fn-panelgetpanelsbeforenextrow)
  * [`fn groupPanelsInRows(panels)`](#fn-panelgrouppanelsinrows)
  * [`fn mapToRows(func, panels)`](#fn-panelmaptorows)
  * [`fn normalizeY(panels)`](#fn-panelnormalizey)
  * [`fn normalizeYInRow(rowPanel)`](#fn-panelnormalizeyinrow)
  * [`fn resolveCollapsedFlagOnRows(panels)`](#fn-panelresolvecollapsedflagonrows)
  * [`fn sanitizePanel(panel, defaultX=0, defaultY=0, defaultHeight=8, defaultWidth=8)`](#fn-panelsanitizepanel)
  * [`fn setPanelIDs(panels, overrideExistingIDs=true)`](#fn-panelsetpanelids)
  * [`fn setRefIDs(panel, overrideExistingIDs=true)`](#fn-panelsetrefids)
  * [`fn setRefIDsOnPanels(panels)`](#fn-panelsetrefidsonpanels)
  * [`fn sortPanelsByXY(panels)`](#fn-panelsortpanelsbyxy)
  * [`fn sortPanelsInRow(rowPanel)`](#fn-panelsortpanelsinrow)
  * [`fn validatePanelIDs(panels)`](#fn-panelvalidatepanelids)
* [`obj string`](#obj-string)
  * [`fn slugify(string)`](#fn-stringslugify)

## Fields

### obj dashboard


#### fn dashboard.getOptionsForCustomQuery

```jsonnet
dashboard.getOptionsForCustomQuery(query)
```

PARAMETERS:

* **query** (`string`)

`getOptionsForCustomQuery` provides values for the `options` and `current` fields.
These are required for template variables of type 'custom'but do not automatically
get populated by Grafana when importing a dashboard from JSON.

This is a bit of a hack and should always be called on functions that set `type` on
a template variable. Ideally Grafana populates these fields from the `query` value
but this provides a backwards compatible solution.

### obj grid


#### fn grid.makeGrid

```jsonnet
grid.makeGrid(panels, panelWidth, panelHeight, startY)
```

PARAMETERS:

* **panels** (`array`)
* **panelWidth** (`number`)
* **panelHeight** (`number`)
* **startY** (`number`)

`makeGrid` returns an array of `panels` organized in a grid with equal `panelWidth`
and `panelHeight`. Row panels are used as "linebreaks", if a Row panel is collapsed,
then all panels below it will be folded into the row.

This function will use the full grid of 24 columns, setting `panelWidth` to a value
that can divide 24 into equal parts will fill up the page nicely. (1, 2, 3, 4, 6, 8, 12)
Other value for `panelWidth` will leave a gap on the far right.

Optional `startY` can be provided to place generated grid above or below existing panels.

#### fn grid.wrapPanels

```jsonnet
grid.wrapPanels(panels, panelWidth, panelHeight, startY)
```

PARAMETERS:

* **panels** (`array`)
* **panelWidth** (`number`)
* **panelHeight** (`number`)
* **startY** (`number`)

`wrapPanels` returns an array of `panels` organized in a grid, wrapping up to next 'row' if total width exceeds full grid of 24 columns.
'panelHeight' and 'panelWidth' are used unless panels already have height and width defined.

### obj panel


#### fn panel.calculateLowestYforPanel

```jsonnet
panel.calculateLowestYforPanel(panel, panels)
```

PARAMETERS:

* **panel** (`object`)
* **panels** (`array`)

`calculateLowestYforPanel` calculates Y for a given `panel` from the `gridPos` of an array of `panels`. This function is used in `normalizeY`.

#### fn panel.dedupeQueryTargets

```jsonnet
panel.dedupeQueryTargets(panels)
```

PARAMETERS:

* **panels** (`array`)

`dedupeQueryTargets` dedupes the query targets in a set of panels and replaces the duplicates with a ['shared query'](https://grafana.com/docs/grafana/latest/panels-visualizations/query-transform-data/share-query/). Sharing query results across panels reduces the number of queries made to your data source, which can improve the performance of your dashboard.

This function requires that the query targets have `refId` set, `setRefIDs` and `setRefIDsOnPanels` can help with that.

#### fn panel.getPanelIDs

```jsonnet
panel.getPanelIDs(panels)
```

PARAMETERS:

* **panels** (`array`)

`getPanelIDs` returns an array with all panel IDs including IDs from panels in rows.

#### fn panel.getPanelsBeforeNextRow

```jsonnet
panel.getPanelsBeforeNextRow(panels)
```

PARAMETERS:

* **panels** (`array`)

`getPanelsBeforeNextRow` returns all panels in an array up until a row has been found. Used in `groupPanelsInRows`.

#### fn panel.groupPanelsInRows

```jsonnet
panel.groupPanelsInRows(panels)
```

PARAMETERS:

* **panels** (`array`)

`groupPanelsInRows` ensures that panels that come after a row panel in an array are added to the `row.panels` attribute. This can be useful to apply intermediate functions to only the panels that belong to a row. Finally the panel array should get processed by `resolveCollapsedFlagOnRows` to "unfold" the rows that are not collapsed into the main array.

#### fn panel.mapToRows

```jsonnet
panel.mapToRows(func, panels)
```

PARAMETERS:

* **func** (`function`)
* **panels** (`array`)

`mapToRows` is a little helper function that applies `func` to all row panels in an array. Other panels in that array are returned ad verbatim.

#### fn panel.normalizeY

```jsonnet
panel.normalizeY(panels)
```

PARAMETERS:

* **panels** (`array`)

`normalizeY` applies negative gravity on the inverted Y axis. This mimics the behavior of Grafana: when a panel is created without panel above it, then it'll float upward.

This is strictly not required as Grafana will do this on dashboard load, however it might be helpful when used when calculating the correct `gridPos`.

#### fn panel.normalizeYInRow

```jsonnet
panel.normalizeYInRow(rowPanel)
```

PARAMETERS:

* **rowPanel** (`object`)

`normalizeYInRow` applies `normalizeY` to the panels in a row panel.

#### fn panel.resolveCollapsedFlagOnRows

```jsonnet
panel.resolveCollapsedFlagOnRows(panels)
```

PARAMETERS:

* **panels** (`array`)

`resolveCollapsedFlagOnRows` should be applied to the final panel array to "unfold" the rows that are not collapsed into the main array.

#### fn panel.sanitizePanel

```jsonnet
panel.sanitizePanel(panel, defaultX=0, defaultY=0, defaultHeight=8, defaultWidth=8)
```

PARAMETERS:

* **panel** (`object`)
* **defaultX** (`number`)
   - default value: `0`
* **defaultY** (`number`)
   - default value: `0`
* **defaultHeight** (`number`)
   - default value: `8`
* **defaultWidth** (`number`)
   - default value: `8`

`sanitizePanel` ensures the panel has a valid `gridPos` and row panels have `collapsed` and `panels`. This function is recursively applied to panels inside row panels.

The default values for x,y,h,w are only applied if not already set.

#### fn panel.setPanelIDs

```jsonnet
panel.setPanelIDs(panels, overrideExistingIDs=true)
```

PARAMETERS:

* **panels** (`array`)
* **overrideExistingIDs** (`bool`)
   - default value: `true`

`setPanelIDs` ensures that all `panels` have a unique ID, this function is used in `dashboard.withPanels` and `dashboard.withPanelsMixin` to provide a consistent experience.

`overrideExistingIDs` can be set to not replace existing IDs, consider validating the IDs with `validatePanelIDs()` to ensure there are no duplicate IDs.

#### fn panel.setRefIDs

```jsonnet
panel.setRefIDs(panel, overrideExistingIDs=true)
```

PARAMETERS:

* **panel** (`object`)
* **overrideExistingIDs** (`bool`)
   - default value: `true`

`setRefIDs` calculates the `refId` field for each target on a panel.

#### fn panel.setRefIDsOnPanels

```jsonnet
panel.setRefIDsOnPanels(panels)
```

PARAMETERS:

* **panels** (`array`)

`setRefIDsOnPanels` applies `setRefIDs on all `panels`.

#### fn panel.sortPanelsByXY

```jsonnet
panel.sortPanelsByXY(panels)
```

PARAMETERS:

* **panels** (`array`)

`sortPanelsByXY` applies a simple sorting algorithm, first by x then again by y. This does not take width and height into account.

#### fn panel.sortPanelsInRow

```jsonnet
panel.sortPanelsInRow(rowPanel)
```

PARAMETERS:

* **rowPanel** (`object`)

`sortPanelsInRow` applies `sortPanelsByXY` on the panels in a rowPanel.

#### fn panel.validatePanelIDs

```jsonnet
panel.validatePanelIDs(panels)
```

PARAMETERS:

* **panels** (`array`)

`validatePanelIDs` validates returns `false` if there are duplicate panel IDs in `panels`.

### obj string


#### fn string.slugify

```jsonnet
string.slugify(string)
```

PARAMETERS:

* **string** (`string`)

`slugify` will create a simple slug from `string`, keeping only alphanumeric
characters and replacing spaces with dashes.

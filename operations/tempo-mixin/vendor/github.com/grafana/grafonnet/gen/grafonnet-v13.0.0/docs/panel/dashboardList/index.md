# dashboardList

grafonnet.panel.dashboardList

## Subpackages

* [panelOptions.link](panelOptions/link.md)
* [queryOptions.transformation](queryOptions/transformation.md)
* [standardOptions.mapping](standardOptions/mapping.md)
* [standardOptions.override](standardOptions/override.md)
* [standardOptions.threshold.step](standardOptions/threshold/step.md)

## Index

* [`fn new(title)`](#fn-new)
* [`obj libraryPanel`](#obj-librarypanel)
  * [`fn withName(value)`](#fn-librarypanelwithname)
  * [`fn withUid(value)`](#fn-librarypanelwithuid)
* [`obj options`](#obj-options)
  * [`fn withFolderId(value)`](#fn-optionswithfolderid)
  * [`fn withFolderUID(value)`](#fn-optionswithfolderuid)
  * [`fn withIncludeVars(value=true)`](#fn-optionswithincludevars)
  * [`fn withKeepTime(value=true)`](#fn-optionswithkeeptime)
  * [`fn withMaxItems(value=10)`](#fn-optionswithmaxitems)
  * [`fn withQuery(value="")`](#fn-optionswithquery)
  * [`fn withShowFolderNames(value=true)`](#fn-optionswithshowfoldernames)
  * [`fn withShowHeadings(value=true)`](#fn-optionswithshowheadings)
  * [`fn withShowRecentlyViewed(value=true)`](#fn-optionswithshowrecentlyviewed)
  * [`fn withShowSearch(value=true)`](#fn-optionswithshowsearch)
  * [`fn withShowStarred(value=true)`](#fn-optionswithshowstarred)
  * [`fn withTags(value)`](#fn-optionswithtags)
  * [`fn withTagsMixin(value)`](#fn-optionswithtagsmixin)
* [`obj panelOptions`](#obj-paneloptions)
  * [`fn withDescription(value)`](#fn-paneloptionswithdescription)
  * [`fn withGridPos(h="null", w="null", x="null", y="null")`](#fn-paneloptionswithgridpos)
  * [`fn withLinks(value)`](#fn-paneloptionswithlinks)
  * [`fn withLinksMixin(value)`](#fn-paneloptionswithlinksmixin)
  * [`fn withMaxPerRow(value)`](#fn-paneloptionswithmaxperrow)
  * [`fn withRepeat(value)`](#fn-paneloptionswithrepeat)
  * [`fn withRepeatDirection(value="h")`](#fn-paneloptionswithrepeatdirection)
  * [`fn withTitle(value)`](#fn-paneloptionswithtitle)
  * [`fn withTransparent(value=true)`](#fn-paneloptionswithtransparent)
* [`obj queryOptions`](#obj-queryoptions)
  * [`fn withDatasource(type, uid)`](#fn-queryoptionswithdatasource)
  * [`fn withDatasourceMixin(value)`](#fn-queryoptionswithdatasourcemixin)
  * [`fn withHideTimeOverride(value=true)`](#fn-queryoptionswithhidetimeoverride)
  * [`fn withInterval(value)`](#fn-queryoptionswithinterval)
  * [`fn withMaxDataPoints(value)`](#fn-queryoptionswithmaxdatapoints)
  * [`fn withQueryCachingTTL(value)`](#fn-queryoptionswithquerycachingttl)
  * [`fn withTargets(value)`](#fn-queryoptionswithtargets)
  * [`fn withTargetsMixin(value)`](#fn-queryoptionswithtargetsmixin)
  * [`fn withTimeFrom(value)`](#fn-queryoptionswithtimefrom)
  * [`fn withTimeShift(value)`](#fn-queryoptionswithtimeshift)
  * [`fn withTransformations(value)`](#fn-queryoptionswithtransformations)
  * [`fn withTransformationsMixin(value)`](#fn-queryoptionswithtransformationsmixin)
* [`obj standardOptions`](#obj-standardoptions)
  * [`fn withDecimals(value)`](#fn-standardoptionswithdecimals)
  * [`fn withDisplayName(value)`](#fn-standardoptionswithdisplayname)
  * [`fn withFilterable(value=true)`](#fn-standardoptionswithfilterable)
  * [`fn withLinks(value)`](#fn-standardoptionswithlinks)
  * [`fn withLinksMixin(value)`](#fn-standardoptionswithlinksmixin)
  * [`fn withMappings(value)`](#fn-standardoptionswithmappings)
  * [`fn withMappingsMixin(value)`](#fn-standardoptionswithmappingsmixin)
  * [`fn withMax(value)`](#fn-standardoptionswithmax)
  * [`fn withMin(value)`](#fn-standardoptionswithmin)
  * [`fn withNoValue(value)`](#fn-standardoptionswithnovalue)
  * [`fn withOverrides(value)`](#fn-standardoptionswithoverrides)
  * [`fn withOverridesMixin(value)`](#fn-standardoptionswithoverridesmixin)
  * [`fn withPath(value)`](#fn-standardoptionswithpath)
  * [`fn withUnit(value)`](#fn-standardoptionswithunit)
  * [`obj color`](#obj-standardoptionscolor)
    * [`fn withFixedColor(value)`](#fn-standardoptionscolorwithfixedcolor)
    * [`fn withMode(value)`](#fn-standardoptionscolorwithmode)
    * [`fn withSeriesBy(value)`](#fn-standardoptionscolorwithseriesby)
  * [`obj thresholds`](#obj-standardoptionsthresholds)
    * [`fn withMode(value)`](#fn-standardoptionsthresholdswithmode)
    * [`fn withSteps(value)`](#fn-standardoptionsthresholdswithsteps)
    * [`fn withStepsMixin(value)`](#fn-standardoptionsthresholdswithstepsmixin)

## Fields

### fn new

```jsonnet
new(title)
```

PARAMETERS:

* **title** (`string`)

Creates a new dashboardList panel with a title.
### obj libraryPanel


#### fn libraryPanel.withName

```jsonnet
libraryPanel.withName(value)
```

PARAMETERS:

* **value** (`string`)

Library panel name
#### fn libraryPanel.withUid

```jsonnet
libraryPanel.withUid(value)
```

PARAMETERS:

* **value** (`string`)

Library panel uid
### obj options


#### fn options.withFolderId

```jsonnet
options.withFolderId(value)
```

PARAMETERS:

* **value** (`integer`)

folderId is deprecated, and migrated to folderUid on panel init
#### fn options.withFolderUID

```jsonnet
options.withFolderUID(value)
```

PARAMETERS:

* **value** (`string`)


#### fn options.withIncludeVars

```jsonnet
options.withIncludeVars(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


#### fn options.withKeepTime

```jsonnet
options.withKeepTime(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


#### fn options.withMaxItems

```jsonnet
options.withMaxItems(value=10)
```

PARAMETERS:

* **value** (`integer`)
   - default value: `10`


#### fn options.withQuery

```jsonnet
options.withQuery(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`


#### fn options.withShowFolderNames

```jsonnet
options.withShowFolderNames(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


#### fn options.withShowHeadings

```jsonnet
options.withShowHeadings(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


#### fn options.withShowRecentlyViewed

```jsonnet
options.withShowRecentlyViewed(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


#### fn options.withShowSearch

```jsonnet
options.withShowSearch(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


#### fn options.withShowStarred

```jsonnet
options.withShowStarred(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


#### fn options.withTags

```jsonnet
options.withTags(value)
```

PARAMETERS:

* **value** (`array`)


#### fn options.withTagsMixin

```jsonnet
options.withTagsMixin(value)
```

PARAMETERS:

* **value** (`array`)


### obj panelOptions


#### fn panelOptions.withDescription

```jsonnet
panelOptions.withDescription(value)
```

PARAMETERS:

* **value** (`string`)

Panel description.
#### fn panelOptions.withGridPos

```jsonnet
panelOptions.withGridPos(h="null", w="null", x="null", y="null")
```

PARAMETERS:

* **h** (`number`)
   - default value: `"null"`
* **w** (`number`)
   - default value: `"null"`
* **x** (`number`)
   - default value: `"null"`
* **y** (`number`)
   - default value: `"null"`

`withGridPos` configures the height, width and xy coordinates of the panel. Also see `grafonnet.util.grid` for helper functions to calculate these fields.

All arguments default to `null`, which means they will remain unchanged or unset.

#### fn panelOptions.withLinks

```jsonnet
panelOptions.withLinks(value)
```

PARAMETERS:

* **value** (`array`)

Panel links.
#### fn panelOptions.withLinksMixin

```jsonnet
panelOptions.withLinksMixin(value)
```

PARAMETERS:

* **value** (`array`)

Panel links.
#### fn panelOptions.withMaxPerRow

```jsonnet
panelOptions.withMaxPerRow(value)
```

PARAMETERS:

* **value** (`number`)

Option for repeated panels that controls max items per row
Only relevant for horizontally repeated panels
#### fn panelOptions.withRepeat

```jsonnet
panelOptions.withRepeat(value)
```

PARAMETERS:

* **value** (`string`)

Name of template variable to repeat for.
#### fn panelOptions.withRepeatDirection

```jsonnet
panelOptions.withRepeatDirection(value="h")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"h"`
   - valid values: `"h"`, `"v"`

Direction to repeat in if 'repeat' is set.
`h` for horizontal, `v` for vertical.
#### fn panelOptions.withTitle

```jsonnet
panelOptions.withTitle(value)
```

PARAMETERS:

* **value** (`string`)

Panel title.
#### fn panelOptions.withTransparent

```jsonnet
panelOptions.withTransparent(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Whether to display the panel without a background.
### obj queryOptions


#### fn queryOptions.withDatasource

```jsonnet
queryOptions.withDatasource(type, uid)
```

PARAMETERS:

* **type** (`string`)
* **uid** (`string`)

`withDatasource` sets the datasource for all queries in a panel.

The default datasource for a panel is set to 'Mixed datasource' so panels can be datasource agnostic, which is a lot more interesting from a reusability standpoint. Note that this requires query targets to explicitly set datasource for the same reason.

#### fn queryOptions.withDatasourceMixin

```jsonnet
queryOptions.withDatasourceMixin(value)
```

PARAMETERS:

* **value** (`object`)

The datasource used in all targets.
#### fn queryOptions.withHideTimeOverride

```jsonnet
queryOptions.withHideTimeOverride(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Controls if the timeFrom or timeShift overrides are shown in the panel header
#### fn queryOptions.withInterval

```jsonnet
queryOptions.withInterval(value)
```

PARAMETERS:

* **value** (`string`)

The min time interval setting defines a lower limit for the $__interval and $__interval_ms variables.
This value must be formatted as a number followed by a valid time
identifier like: "40s", "3d", etc.
See: https://grafana.com/docs/grafana/latest/panels-visualizations/query-transform-data/#query-options
#### fn queryOptions.withMaxDataPoints

```jsonnet
queryOptions.withMaxDataPoints(value)
```

PARAMETERS:

* **value** (`number`)

The maximum number of data points that the panel queries are retrieving.
#### fn queryOptions.withQueryCachingTTL

```jsonnet
queryOptions.withQueryCachingTTL(value)
```

PARAMETERS:

* **value** (`number`)

Overrides the data source configured time-to-live for a query cache item in milliseconds
#### fn queryOptions.withTargets

```jsonnet
queryOptions.withTargets(value)
```

PARAMETERS:

* **value** (`array`)

Depends on the panel plugin. See the plugin documentation for details.
#### fn queryOptions.withTargetsMixin

```jsonnet
queryOptions.withTargetsMixin(value)
```

PARAMETERS:

* **value** (`array`)

Depends on the panel plugin. See the plugin documentation for details.
#### fn queryOptions.withTimeFrom

```jsonnet
queryOptions.withTimeFrom(value)
```

PARAMETERS:

* **value** (`string`)

Overrides the relative time range for individual panels,
which causes them to be different than what is selected in
the dashboard time picker in the top-right corner of the dashboard. You can use this to show metrics from different
time periods or days on the same dashboard.
The value is formatted as time operation like: `now-5m` (Last 5 minutes), `now/d` (the day so far),
`now-5d/d`(Last 5 days), `now/w` (This week so far), `now-2y/y` (Last 2 years).
Note: Panel time overrides have no effect when the dashboard’s time range is absolute.
See: https://grafana.com/docs/grafana/latest/panels-visualizations/query-transform-data/#query-options
#### fn queryOptions.withTimeShift

```jsonnet
queryOptions.withTimeShift(value)
```

PARAMETERS:

* **value** (`string`)

Overrides the time range for individual panels by shifting its start and end relative to the time picker.
For example, you can shift the time range for the panel to be two hours earlier than the dashboard time picker setting `2h`.
Note: Panel time overrides have no effect when the dashboard’s time range is absolute.
See: https://grafana.com/docs/grafana/latest/panels-visualizations/query-transform-data/#query-options
#### fn queryOptions.withTransformations

```jsonnet
queryOptions.withTransformations(value)
```

PARAMETERS:

* **value** (`array`)

List of transformations that are applied to the panel data before rendering.
When there are multiple transformations, Grafana applies them in the order they are listed.
Each transformation creates a result set that then passes on to the next transformation in the processing pipeline.
#### fn queryOptions.withTransformationsMixin

```jsonnet
queryOptions.withTransformationsMixin(value)
```

PARAMETERS:

* **value** (`array`)

List of transformations that are applied to the panel data before rendering.
When there are multiple transformations, Grafana applies them in the order they are listed.
Each transformation creates a result set that then passes on to the next transformation in the processing pipeline.
### obj standardOptions


#### fn standardOptions.withDecimals

```jsonnet
standardOptions.withDecimals(value)
```

PARAMETERS:

* **value** (`number`)

Specify the number of decimals Grafana includes in the rendered value.
If you leave this field blank, Grafana automatically truncates the number of decimals based on the value.
For example 1.1234 will display as 1.12 and 100.456 will display as 100.
To display all decimals, set the unit to `String`.
#### fn standardOptions.withDisplayName

```jsonnet
standardOptions.withDisplayName(value)
```

PARAMETERS:

* **value** (`string`)

The display value for this field.  This supports template variables blank is auto
#### fn standardOptions.withFilterable

```jsonnet
standardOptions.withFilterable(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

True if data source field supports ad-hoc filters
#### fn standardOptions.withLinks

```jsonnet
standardOptions.withLinks(value)
```

PARAMETERS:

* **value** (`array`)

The behavior when clicking on a result
#### fn standardOptions.withLinksMixin

```jsonnet
standardOptions.withLinksMixin(value)
```

PARAMETERS:

* **value** (`array`)

The behavior when clicking on a result
#### fn standardOptions.withMappings

```jsonnet
standardOptions.withMappings(value)
```

PARAMETERS:

* **value** (`array`)

Convert input values into a display string
#### fn standardOptions.withMappingsMixin

```jsonnet
standardOptions.withMappingsMixin(value)
```

PARAMETERS:

* **value** (`array`)

Convert input values into a display string
#### fn standardOptions.withMax

```jsonnet
standardOptions.withMax(value)
```

PARAMETERS:

* **value** (`number`)

The maximum value used in percentage threshold calculations. Leave blank for auto calculation based on all series and fields.
#### fn standardOptions.withMin

```jsonnet
standardOptions.withMin(value)
```

PARAMETERS:

* **value** (`number`)

The minimum value used in percentage threshold calculations. Leave blank for auto calculation based on all series and fields.
#### fn standardOptions.withNoValue

```jsonnet
standardOptions.withNoValue(value)
```

PARAMETERS:

* **value** (`string`)

Alternative to empty string
#### fn standardOptions.withOverrides

```jsonnet
standardOptions.withOverrides(value)
```

PARAMETERS:

* **value** (`array`)

Overrides are the options applied to specific fields overriding the defaults.
#### fn standardOptions.withOverridesMixin

```jsonnet
standardOptions.withOverridesMixin(value)
```

PARAMETERS:

* **value** (`array`)

Overrides are the options applied to specific fields overriding the defaults.
#### fn standardOptions.withPath

```jsonnet
standardOptions.withPath(value)
```

PARAMETERS:

* **value** (`string`)

An explicit path to the field in the datasource.  When the frame meta includes a path,
This will default to `${frame.meta.path}/${field.name}

When defined, this value can be used as an identifier within the datasource scope, and
may be used to update the results
#### fn standardOptions.withUnit

```jsonnet
standardOptions.withUnit(value)
```

PARAMETERS:

* **value** (`string`)

Unit a field should use. The unit you select is applied to all fields except time.
You can use the units ID availables in Grafana or a custom unit.
Available units in Grafana: https://github.com/grafana/grafana/blob/main/packages/grafana-data/src/valueFormats/categories.ts
As custom unit, you can use the following formats:
`suffix:<suffix>` for custom unit that should go after value.
`prefix:<prefix>` for custom unit that should go before value.
`time:<format>` For custom date time formats type for example `time:YYYY-MM-DD`.
`si:<base scale><unit characters>` for custom SI units. For example: `si: mF`. This one is a bit more advanced as you can specify both a unit and the source data scale. So if your source data is represented as milli (thousands of) something prefix the unit with that SI scale character.
`count:<unit>` for a custom count unit.
`currency:<unit>` for custom a currency unit.
#### obj standardOptions.color


##### fn standardOptions.color.withFixedColor

```jsonnet
standardOptions.color.withFixedColor(value)
```

PARAMETERS:

* **value** (`string`)

The fixed color value for fixed or shades color modes.
##### fn standardOptions.color.withMode

```jsonnet
standardOptions.color.withMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"thresholds"`, `"palette-classic"`, `"palette-classic-by-name"`, `"continuous-GrYlRd"`, `"continuous-RdYlGr"`, `"continuous-BlYlRd"`, `"continuous-YlRd"`, `"continuous-BlPu"`, `"continuous-YlBl"`, `"continuous-blues"`, `"continuous-reds"`, `"continuous-greens"`, `"continuous-purples"`, `"fixed"`, `"shades"`

Color mode for a field. You can specify a single color, or select a continuous (gradient) color schemes, based on a value.
Continuous color interpolates a color using the percentage of a value relative to min and max.
Accepted values are:
`thresholds`: From thresholds. Informs Grafana to take the color from the matching threshold
`palette-classic`: Classic palette. Grafana will assign color by looking up a color in a palette by series index. Useful for Graphs and pie charts and other categorical data visualizations
`palette-classic-by-name`: Classic palette (by name). Grafana will assign color by looking up a color in a palette by series name. Useful for Graphs and pie charts and other categorical data visualizations
`continuous-GrYlRd`: ontinuous Green-Yellow-Red palette mode
`continuous-RdYlGr`: Continuous Red-Yellow-Green palette mode
`continuous-BlYlRd`: Continuous Blue-Yellow-Red palette mode
`continuous-YlRd`: Continuous Yellow-Red palette mode
`continuous-BlPu`: Continuous Blue-Purple palette mode
`continuous-YlBl`: Continuous Yellow-Blue palette mode
`continuous-blues`: Continuous Blue palette mode
`continuous-reds`: Continuous Red palette mode
`continuous-greens`: Continuous Green palette mode
`continuous-purples`: Continuous Purple palette mode
`shades`: Shades of a single color. Specify a single color, useful in an override rule.
`fixed`: Fixed color mode. Specify a single color, useful in an override rule.
##### fn standardOptions.color.withSeriesBy

```jsonnet
standardOptions.color.withSeriesBy(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"min"`, `"max"`, `"last"`

Defines how to assign a series color from "by value" color schemes. For example for an aggregated data points like a timeseries, the color can be assigned by the min, max or last value.
#### obj standardOptions.thresholds


##### fn standardOptions.thresholds.withMode

```jsonnet
standardOptions.thresholds.withMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"absolute"`, `"percentage"`

Thresholds can either be `absolute` (specific number) or `percentage` (relative to min or max, it will be values between 0 and 1).
##### fn standardOptions.thresholds.withSteps

```jsonnet
standardOptions.thresholds.withSteps(value)
```

PARAMETERS:

* **value** (`array`)

Must be sorted by 'value', first value is always -Infinity
##### fn standardOptions.thresholds.withStepsMixin

```jsonnet
standardOptions.thresholds.withStepsMixin(value)
```

PARAMETERS:

* **value** (`array`)

Must be sorted by 'value', first value is always -Infinity
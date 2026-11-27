# trend

grafonnet.panel.trend

## Subpackages

* [panelOptions.link](panelOptions/link.md)
* [queryOptions.transformation](queryOptions/transformation.md)
* [standardOptions.mapping](standardOptions/mapping.md)
* [standardOptions.override](standardOptions/override.md)
* [standardOptions.threshold.step](standardOptions/threshold/step.md)

## Index

* [`fn new(title)`](#fn-new)
* [`obj fieldConfig`](#obj-fieldconfig)
  * [`obj defaults`](#obj-fieldconfigdefaults)
    * [`obj custom`](#obj-fieldconfigdefaultscustom)
      * [`fn withAxisBorderShow(value=true)`](#fn-fieldconfigdefaultscustomwithaxisbordershow)
      * [`fn withAxisCenteredZero(value=true)`](#fn-fieldconfigdefaultscustomwithaxiscenteredzero)
      * [`fn withAxisColorMode(value)`](#fn-fieldconfigdefaultscustomwithaxiscolormode)
      * [`fn withAxisGridShow(value=true)`](#fn-fieldconfigdefaultscustomwithaxisgridshow)
      * [`fn withAxisLabel(value)`](#fn-fieldconfigdefaultscustomwithaxislabel)
      * [`fn withAxisPlacement(value)`](#fn-fieldconfigdefaultscustomwithaxisplacement)
      * [`fn withAxisSoftMax(value)`](#fn-fieldconfigdefaultscustomwithaxissoftmax)
      * [`fn withAxisSoftMin(value)`](#fn-fieldconfigdefaultscustomwithaxissoftmin)
      * [`fn withAxisWidth(value)`](#fn-fieldconfigdefaultscustomwithaxiswidth)
      * [`fn withBarAlignment(value)`](#fn-fieldconfigdefaultscustomwithbaralignment)
      * [`fn withBarMaxWidth(value)`](#fn-fieldconfigdefaultscustomwithbarmaxwidth)
      * [`fn withBarWidthFactor(value)`](#fn-fieldconfigdefaultscustomwithbarwidthfactor)
      * [`fn withDrawStyle(value)`](#fn-fieldconfigdefaultscustomwithdrawstyle)
      * [`fn withFillBelowTo(value)`](#fn-fieldconfigdefaultscustomwithfillbelowto)
      * [`fn withFillColor(value)`](#fn-fieldconfigdefaultscustomwithfillcolor)
      * [`fn withFillOpacity(value)`](#fn-fieldconfigdefaultscustomwithfillopacity)
      * [`fn withGradientMode(value)`](#fn-fieldconfigdefaultscustomwithgradientmode)
      * [`fn withHideFrom(value)`](#fn-fieldconfigdefaultscustomwithhidefrom)
      * [`fn withHideFromMixin(value)`](#fn-fieldconfigdefaultscustomwithhidefrommixin)
      * [`fn withInsertNulls(value)`](#fn-fieldconfigdefaultscustomwithinsertnulls)
      * [`fn withInsertNullsMixin(value)`](#fn-fieldconfigdefaultscustomwithinsertnullsmixin)
      * [`fn withLineColor(value)`](#fn-fieldconfigdefaultscustomwithlinecolor)
      * [`fn withLineInterpolation(value)`](#fn-fieldconfigdefaultscustomwithlineinterpolation)
      * [`fn withLineStyle(value)`](#fn-fieldconfigdefaultscustomwithlinestyle)
      * [`fn withLineStyleMixin(value)`](#fn-fieldconfigdefaultscustomwithlinestylemixin)
      * [`fn withLineWidth(value)`](#fn-fieldconfigdefaultscustomwithlinewidth)
      * [`fn withPointColor(value)`](#fn-fieldconfigdefaultscustomwithpointcolor)
      * [`fn withPointSize(value)`](#fn-fieldconfigdefaultscustomwithpointsize)
      * [`fn withPointSymbol(value)`](#fn-fieldconfigdefaultscustomwithpointsymbol)
      * [`fn withScaleDistribution(value)`](#fn-fieldconfigdefaultscustomwithscaledistribution)
      * [`fn withScaleDistributionMixin(value)`](#fn-fieldconfigdefaultscustomwithscaledistributionmixin)
      * [`fn withShowPoints(value)`](#fn-fieldconfigdefaultscustomwithshowpoints)
      * [`fn withSpanNulls(value)`](#fn-fieldconfigdefaultscustomwithspannulls)
      * [`fn withSpanNullsMixin(value)`](#fn-fieldconfigdefaultscustomwithspannullsmixin)
      * [`fn withStacking(value)`](#fn-fieldconfigdefaultscustomwithstacking)
      * [`fn withStackingMixin(value)`](#fn-fieldconfigdefaultscustomwithstackingmixin)
      * [`fn withThresholdsStyle(value)`](#fn-fieldconfigdefaultscustomwiththresholdsstyle)
      * [`fn withThresholdsStyleMixin(value)`](#fn-fieldconfigdefaultscustomwiththresholdsstylemixin)
      * [`fn withTransform(value)`](#fn-fieldconfigdefaultscustomwithtransform)
      * [`obj hideFrom`](#obj-fieldconfigdefaultscustomhidefrom)
        * [`fn withLegend(value=true)`](#fn-fieldconfigdefaultscustomhidefromwithlegend)
        * [`fn withTooltip(value=true)`](#fn-fieldconfigdefaultscustomhidefromwithtooltip)
        * [`fn withViz(value=true)`](#fn-fieldconfigdefaultscustomhidefromwithviz)
      * [`obj lineStyle`](#obj-fieldconfigdefaultscustomlinestyle)
        * [`fn withDash(value)`](#fn-fieldconfigdefaultscustomlinestylewithdash)
        * [`fn withDashMixin(value)`](#fn-fieldconfigdefaultscustomlinestylewithdashmixin)
        * [`fn withFill(value)`](#fn-fieldconfigdefaultscustomlinestylewithfill)
      * [`obj scaleDistribution`](#obj-fieldconfigdefaultscustomscaledistribution)
        * [`fn withLinearThreshold(value)`](#fn-fieldconfigdefaultscustomscaledistributionwithlinearthreshold)
        * [`fn withLog(value)`](#fn-fieldconfigdefaultscustomscaledistributionwithlog)
        * [`fn withType(value)`](#fn-fieldconfigdefaultscustomscaledistributionwithtype)
      * [`obj stacking`](#obj-fieldconfigdefaultscustomstacking)
        * [`fn withGroup(value)`](#fn-fieldconfigdefaultscustomstackingwithgroup)
        * [`fn withMode(value)`](#fn-fieldconfigdefaultscustomstackingwithmode)
      * [`obj thresholdsStyle`](#obj-fieldconfigdefaultscustomthresholdsstyle)
        * [`fn withMode(value)`](#fn-fieldconfigdefaultscustomthresholdsstylewithmode)
* [`obj libraryPanel`](#obj-librarypanel)
  * [`fn withName(value)`](#fn-librarypanelwithname)
  * [`fn withUid(value)`](#fn-librarypanelwithuid)
* [`obj options`](#obj-options)
  * [`fn withLegend(value)`](#fn-optionswithlegend)
  * [`fn withLegendMixin(value)`](#fn-optionswithlegendmixin)
  * [`fn withTooltip(value)`](#fn-optionswithtooltip)
  * [`fn withTooltipMixin(value)`](#fn-optionswithtooltipmixin)
  * [`fn withXField(value)`](#fn-optionswithxfield)
  * [`obj legend`](#obj-optionslegend)
    * [`fn withAsTable(value=true)`](#fn-optionslegendwithastable)
    * [`fn withCalcs(value=[])`](#fn-optionslegendwithcalcs)
    * [`fn withCalcsMixin(value=[])`](#fn-optionslegendwithcalcsmixin)
    * [`fn withDisplayMode(value="list")`](#fn-optionslegendwithdisplaymode)
    * [`fn withIsVisible(value=true)`](#fn-optionslegendwithisvisible)
    * [`fn withPlacement(value="bottom")`](#fn-optionslegendwithplacement)
    * [`fn withShowLegend(value=true)`](#fn-optionslegendwithshowlegend)
    * [`fn withSortBy(value)`](#fn-optionslegendwithsortby)
    * [`fn withSortDesc(value=true)`](#fn-optionslegendwithsortdesc)
    * [`fn withWidth(value)`](#fn-optionslegendwithwidth)
  * [`obj tooltip`](#obj-optionstooltip)
    * [`fn withHideZeros(value=true)`](#fn-optionstooltipwithhidezeros)
    * [`fn withMaxHeight(value)`](#fn-optionstooltipwithmaxheight)
    * [`fn withMaxWidth(value)`](#fn-optionstooltipwithmaxwidth)
    * [`fn withMode(value)`](#fn-optionstooltipwithmode)
    * [`fn withSort(value)`](#fn-optionstooltipwithsort)
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

Creates a new trend panel with a title.
### obj fieldConfig


#### obj fieldConfig.defaults


##### obj fieldConfig.defaults.custom


###### fn fieldConfig.defaults.custom.withAxisBorderShow

```jsonnet
fieldConfig.defaults.custom.withAxisBorderShow(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


###### fn fieldConfig.defaults.custom.withAxisCenteredZero

```jsonnet
fieldConfig.defaults.custom.withAxisCenteredZero(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


###### fn fieldConfig.defaults.custom.withAxisColorMode

```jsonnet
fieldConfig.defaults.custom.withAxisColorMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"text"`, `"series"`

TODO docs
###### fn fieldConfig.defaults.custom.withAxisGridShow

```jsonnet
fieldConfig.defaults.custom.withAxisGridShow(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


###### fn fieldConfig.defaults.custom.withAxisLabel

```jsonnet
fieldConfig.defaults.custom.withAxisLabel(value)
```

PARAMETERS:

* **value** (`string`)


###### fn fieldConfig.defaults.custom.withAxisPlacement

```jsonnet
fieldConfig.defaults.custom.withAxisPlacement(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"auto"`, `"top"`, `"right"`, `"bottom"`, `"left"`, `"hidden"`

TODO docs
###### fn fieldConfig.defaults.custom.withAxisSoftMax

```jsonnet
fieldConfig.defaults.custom.withAxisSoftMax(value)
```

PARAMETERS:

* **value** (`number`)


###### fn fieldConfig.defaults.custom.withAxisSoftMin

```jsonnet
fieldConfig.defaults.custom.withAxisSoftMin(value)
```

PARAMETERS:

* **value** (`number`)


###### fn fieldConfig.defaults.custom.withAxisWidth

```jsonnet
fieldConfig.defaults.custom.withAxisWidth(value)
```

PARAMETERS:

* **value** (`number`)


###### fn fieldConfig.defaults.custom.withBarAlignment

```jsonnet
fieldConfig.defaults.custom.withBarAlignment(value)
```

PARAMETERS:

* **value** (`integer`)
   - valid values: `-1`, `0`, `1`

TODO docs
###### fn fieldConfig.defaults.custom.withBarMaxWidth

```jsonnet
fieldConfig.defaults.custom.withBarMaxWidth(value)
```

PARAMETERS:

* **value** (`number`)


###### fn fieldConfig.defaults.custom.withBarWidthFactor

```jsonnet
fieldConfig.defaults.custom.withBarWidthFactor(value)
```

PARAMETERS:

* **value** (`number`)


###### fn fieldConfig.defaults.custom.withDrawStyle

```jsonnet
fieldConfig.defaults.custom.withDrawStyle(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"line"`, `"bars"`, `"points"`

TODO docs
###### fn fieldConfig.defaults.custom.withFillBelowTo

```jsonnet
fieldConfig.defaults.custom.withFillBelowTo(value)
```

PARAMETERS:

* **value** (`string`)


###### fn fieldConfig.defaults.custom.withFillColor

```jsonnet
fieldConfig.defaults.custom.withFillColor(value)
```

PARAMETERS:

* **value** (`string`)


###### fn fieldConfig.defaults.custom.withFillOpacity

```jsonnet
fieldConfig.defaults.custom.withFillOpacity(value)
```

PARAMETERS:

* **value** (`number`)


###### fn fieldConfig.defaults.custom.withGradientMode

```jsonnet
fieldConfig.defaults.custom.withGradientMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"none"`, `"opacity"`, `"hue"`, `"scheme"`

TODO docs
###### fn fieldConfig.defaults.custom.withHideFrom

```jsonnet
fieldConfig.defaults.custom.withHideFrom(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
###### fn fieldConfig.defaults.custom.withHideFromMixin

```jsonnet
fieldConfig.defaults.custom.withHideFromMixin(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
###### fn fieldConfig.defaults.custom.withInsertNulls

```jsonnet
fieldConfig.defaults.custom.withInsertNulls(value)
```

PARAMETERS:

* **value** (`boolean`,`number`)


###### fn fieldConfig.defaults.custom.withInsertNullsMixin

```jsonnet
fieldConfig.defaults.custom.withInsertNullsMixin(value)
```

PARAMETERS:

* **value** (`boolean`,`number`)


###### fn fieldConfig.defaults.custom.withLineColor

```jsonnet
fieldConfig.defaults.custom.withLineColor(value)
```

PARAMETERS:

* **value** (`string`)


###### fn fieldConfig.defaults.custom.withLineInterpolation

```jsonnet
fieldConfig.defaults.custom.withLineInterpolation(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"linear"`, `"smooth"`, `"stepBefore"`, `"stepAfter"`

TODO docs
###### fn fieldConfig.defaults.custom.withLineStyle

```jsonnet
fieldConfig.defaults.custom.withLineStyle(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
###### fn fieldConfig.defaults.custom.withLineStyleMixin

```jsonnet
fieldConfig.defaults.custom.withLineStyleMixin(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
###### fn fieldConfig.defaults.custom.withLineWidth

```jsonnet
fieldConfig.defaults.custom.withLineWidth(value)
```

PARAMETERS:

* **value** (`number`)


###### fn fieldConfig.defaults.custom.withPointColor

```jsonnet
fieldConfig.defaults.custom.withPointColor(value)
```

PARAMETERS:

* **value** (`string`)


###### fn fieldConfig.defaults.custom.withPointSize

```jsonnet
fieldConfig.defaults.custom.withPointSize(value)
```

PARAMETERS:

* **value** (`number`)


###### fn fieldConfig.defaults.custom.withPointSymbol

```jsonnet
fieldConfig.defaults.custom.withPointSymbol(value)
```

PARAMETERS:

* **value** (`string`)


###### fn fieldConfig.defaults.custom.withScaleDistribution

```jsonnet
fieldConfig.defaults.custom.withScaleDistribution(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
###### fn fieldConfig.defaults.custom.withScaleDistributionMixin

```jsonnet
fieldConfig.defaults.custom.withScaleDistributionMixin(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
###### fn fieldConfig.defaults.custom.withShowPoints

```jsonnet
fieldConfig.defaults.custom.withShowPoints(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"auto"`, `"never"`, `"always"`

TODO docs
###### fn fieldConfig.defaults.custom.withSpanNulls

```jsonnet
fieldConfig.defaults.custom.withSpanNulls(value)
```

PARAMETERS:

* **value** (`boolean`,`number`)

Indicate if null values should be treated as gaps or connected.
When the value is a number, it represents the maximum delta in the
X axis that should be considered connected.  For timeseries, this is milliseconds
###### fn fieldConfig.defaults.custom.withSpanNullsMixin

```jsonnet
fieldConfig.defaults.custom.withSpanNullsMixin(value)
```

PARAMETERS:

* **value** (`boolean`,`number`)

Indicate if null values should be treated as gaps or connected.
When the value is a number, it represents the maximum delta in the
X axis that should be considered connected.  For timeseries, this is milliseconds
###### fn fieldConfig.defaults.custom.withStacking

```jsonnet
fieldConfig.defaults.custom.withStacking(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
###### fn fieldConfig.defaults.custom.withStackingMixin

```jsonnet
fieldConfig.defaults.custom.withStackingMixin(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
###### fn fieldConfig.defaults.custom.withThresholdsStyle

```jsonnet
fieldConfig.defaults.custom.withThresholdsStyle(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
###### fn fieldConfig.defaults.custom.withThresholdsStyleMixin

```jsonnet
fieldConfig.defaults.custom.withThresholdsStyleMixin(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
###### fn fieldConfig.defaults.custom.withTransform

```jsonnet
fieldConfig.defaults.custom.withTransform(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"constant"`, `"negative-Y"`

TODO docs
###### obj fieldConfig.defaults.custom.hideFrom


####### fn fieldConfig.defaults.custom.hideFrom.withLegend

```jsonnet
fieldConfig.defaults.custom.hideFrom.withLegend(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### fn fieldConfig.defaults.custom.hideFrom.withTooltip

```jsonnet
fieldConfig.defaults.custom.hideFrom.withTooltip(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


####### fn fieldConfig.defaults.custom.hideFrom.withViz

```jsonnet
fieldConfig.defaults.custom.hideFrom.withViz(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


###### obj fieldConfig.defaults.custom.lineStyle


####### fn fieldConfig.defaults.custom.lineStyle.withDash

```jsonnet
fieldConfig.defaults.custom.lineStyle.withDash(value)
```

PARAMETERS:

* **value** (`array`)


####### fn fieldConfig.defaults.custom.lineStyle.withDashMixin

```jsonnet
fieldConfig.defaults.custom.lineStyle.withDashMixin(value)
```

PARAMETERS:

* **value** (`array`)


####### fn fieldConfig.defaults.custom.lineStyle.withFill

```jsonnet
fieldConfig.defaults.custom.lineStyle.withFill(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"solid"`, `"dash"`, `"dot"`, `"square"`


###### obj fieldConfig.defaults.custom.scaleDistribution


####### fn fieldConfig.defaults.custom.scaleDistribution.withLinearThreshold

```jsonnet
fieldConfig.defaults.custom.scaleDistribution.withLinearThreshold(value)
```

PARAMETERS:

* **value** (`number`)


####### fn fieldConfig.defaults.custom.scaleDistribution.withLog

```jsonnet
fieldConfig.defaults.custom.scaleDistribution.withLog(value)
```

PARAMETERS:

* **value** (`number`)


####### fn fieldConfig.defaults.custom.scaleDistribution.withType

```jsonnet
fieldConfig.defaults.custom.scaleDistribution.withType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"linear"`, `"log"`, `"ordinal"`, `"symlog"`

TODO docs
###### obj fieldConfig.defaults.custom.stacking


####### fn fieldConfig.defaults.custom.stacking.withGroup

```jsonnet
fieldConfig.defaults.custom.stacking.withGroup(value)
```

PARAMETERS:

* **value** (`string`)


####### fn fieldConfig.defaults.custom.stacking.withMode

```jsonnet
fieldConfig.defaults.custom.stacking.withMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"none"`, `"normal"`, `"percent"`

TODO docs
###### obj fieldConfig.defaults.custom.thresholdsStyle


####### fn fieldConfig.defaults.custom.thresholdsStyle.withMode

```jsonnet
fieldConfig.defaults.custom.thresholdsStyle.withMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"off"`, `"line"`, `"dashed"`, `"area"`, `"line+area"`, `"dashed+area"`, `"series"`

TODO docs
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


#### fn options.withLegend

```jsonnet
options.withLegend(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
#### fn options.withLegendMixin

```jsonnet
options.withLegendMixin(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
#### fn options.withTooltip

```jsonnet
options.withTooltip(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
#### fn options.withTooltipMixin

```jsonnet
options.withTooltipMixin(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
#### fn options.withXField

```jsonnet
options.withXField(value)
```

PARAMETERS:

* **value** (`string`)

Name of the x field to use (defaults to first number)
#### obj options.legend


##### fn options.legend.withAsTable

```jsonnet
options.legend.withAsTable(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn options.legend.withCalcs

```jsonnet
options.legend.withCalcs(value=[])
```

PARAMETERS:

* **value** (`array`)
   - default value: `[]`


##### fn options.legend.withCalcsMixin

```jsonnet
options.legend.withCalcsMixin(value=[])
```

PARAMETERS:

* **value** (`array`)
   - default value: `[]`


##### fn options.legend.withDisplayMode

```jsonnet
options.legend.withDisplayMode(value="list")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"list"`
   - valid values: `"list"`, `"table"`, `"hidden"`

TODO docs
Note: "hidden" needs to remain as an option for plugins compatibility
##### fn options.legend.withIsVisible

```jsonnet
options.legend.withIsVisible(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn options.legend.withPlacement

```jsonnet
options.legend.withPlacement(value="bottom")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"bottom"`
   - valid values: `"bottom"`, `"right"`

TODO docs
##### fn options.legend.withShowLegend

```jsonnet
options.legend.withShowLegend(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn options.legend.withSortBy

```jsonnet
options.legend.withSortBy(value)
```

PARAMETERS:

* **value** (`string`)


##### fn options.legend.withSortDesc

```jsonnet
options.legend.withSortDesc(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn options.legend.withWidth

```jsonnet
options.legend.withWidth(value)
```

PARAMETERS:

* **value** (`number`)


#### obj options.tooltip


##### fn options.tooltip.withHideZeros

```jsonnet
options.tooltip.withHideZeros(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn options.tooltip.withMaxHeight

```jsonnet
options.tooltip.withMaxHeight(value)
```

PARAMETERS:

* **value** (`number`)


##### fn options.tooltip.withMaxWidth

```jsonnet
options.tooltip.withMaxWidth(value)
```

PARAMETERS:

* **value** (`number`)


##### fn options.tooltip.withMode

```jsonnet
options.tooltip.withMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"single"`, `"multi"`, `"none"`

TODO docs
##### fn options.tooltip.withSort

```jsonnet
options.tooltip.withSort(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"asc"`, `"desc"`, `"none"`

TODO docs
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
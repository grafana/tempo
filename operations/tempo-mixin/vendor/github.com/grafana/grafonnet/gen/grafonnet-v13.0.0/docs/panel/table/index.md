# table

grafonnet.panel.table

## Subpackages

* [options.sortBy](options/sortBy.md)
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
      * [`fn withAlign(value="auto")`](#fn-fieldconfigdefaultscustomwithalign)
      * [`fn withCellOptions(value)`](#fn-fieldconfigdefaultscustomwithcelloptions)
      * [`fn withCellOptionsMixin(value)`](#fn-fieldconfigdefaultscustomwithcelloptionsmixin)
      * [`fn withDisplayMode(value)`](#fn-fieldconfigdefaultscustomwithdisplaymode)
      * [`fn withFilterable(value=true)`](#fn-fieldconfigdefaultscustomwithfilterable)
      * [`fn withHidden(value=true)`](#fn-fieldconfigdefaultscustomwithhidden)
      * [`fn withHideHeader(value=true)`](#fn-fieldconfigdefaultscustomwithhideheader)
      * [`fn withInspect(value=true)`](#fn-fieldconfigdefaultscustomwithinspect)
      * [`fn withMinWidth(value)`](#fn-fieldconfigdefaultscustomwithminwidth)
      * [`fn withWidth(value)`](#fn-fieldconfigdefaultscustomwithwidth)
      * [`obj cellOptions`](#obj-fieldconfigdefaultscustomcelloptions)
        * [`fn withTableCellOptions(value)`](#fn-fieldconfigdefaultscustomcelloptionswithtablecelloptions)
        * [`fn withTableCellOptionsMixin(value)`](#fn-fieldconfigdefaultscustomcelloptionswithtablecelloptionsmixin)
        * [`obj TableCellOptions`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptions)
          * [`fn withTableActionsCellOptions(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionswithtableactionscelloptions)
          * [`fn withTableActionsCellOptionsMixin(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionswithtableactionscelloptionsmixin)
          * [`fn withTableAutoCellOptions(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionswithtableautocelloptions)
          * [`fn withTableAutoCellOptionsMixin(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionswithtableautocelloptionsmixin)
          * [`fn withTableBarGaugeCellOptions(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionswithtablebargaugecelloptions)
          * [`fn withTableBarGaugeCellOptionsMixin(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionswithtablebargaugecelloptionsmixin)
          * [`fn withTableColorTextCellOptions(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionswithtablecolortextcelloptions)
          * [`fn withTableColorTextCellOptionsMixin(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionswithtablecolortextcelloptionsmixin)
          * [`fn withTableColoredBackgroundCellOptions(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionswithtablecoloredbackgroundcelloptions)
          * [`fn withTableColoredBackgroundCellOptionsMixin(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionswithtablecoloredbackgroundcelloptionsmixin)
          * [`fn withTableDataLinksCellOptions(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionswithtabledatalinkscelloptions)
          * [`fn withTableDataLinksCellOptionsMixin(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionswithtabledatalinkscelloptionsmixin)
          * [`fn withTableImageCellOptions(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionswithtableimagecelloptions)
          * [`fn withTableImageCellOptionsMixin(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionswithtableimagecelloptionsmixin)
          * [`fn withTableJsonViewCellOptions(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionswithtablejsonviewcelloptions)
          * [`fn withTableJsonViewCellOptionsMixin(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionswithtablejsonviewcelloptionsmixin)
          * [`fn withTableSparklineCellOptions(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionswithtablesparklinecelloptions)
          * [`fn withTableSparklineCellOptionsMixin(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionswithtablesparklinecelloptionsmixin)
          * [`obj TableActionsCellOptions`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstableactionscelloptions)
            * [`fn withType(value="actions")`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstableactionscelloptionswithtype)
            * [`fn withTypeMixin(value="actions")`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstableactionscelloptionswithtypemixin)
            * [`obj type`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstableactionscelloptionstype)
              * [`fn withTableCellDisplayMode(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstableactionscelloptionstypewithtablecelldisplaymode)
          * [`obj TableAutoCellOptions`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstableautocelloptions)
            * [`fn withType(value="auto")`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstableautocelloptionswithtype)
            * [`fn withTypeMixin(value="auto")`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstableautocelloptionswithtypemixin)
            * [`fn withWrapText(value=true)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstableautocelloptionswithwraptext)
            * [`obj type`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstableautocelloptionstype)
              * [`fn withTableCellDisplayMode(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstableautocelloptionstypewithtablecelldisplaymode)
          * [`obj TableBarGaugeCellOptions`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstablebargaugecelloptions)
            * [`fn withMode(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablebargaugecelloptionswithmode)
            * [`fn withType(value="gauge")`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablebargaugecelloptionswithtype)
            * [`fn withTypeMixin(value="gauge")`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablebargaugecelloptionswithtypemixin)
            * [`fn withValueDisplayMode(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablebargaugecelloptionswithvaluedisplaymode)
            * [`obj type`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstablebargaugecelloptionstype)
              * [`fn withTableCellDisplayMode(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablebargaugecelloptionstypewithtablecelldisplaymode)
          * [`obj TableColorTextCellOptions`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstablecolortextcelloptions)
            * [`fn withType(value="color-text")`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablecolortextcelloptionswithtype)
            * [`fn withTypeMixin(value="color-text")`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablecolortextcelloptionswithtypemixin)
            * [`fn withWrapText(value=true)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablecolortextcelloptionswithwraptext)
            * [`obj type`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstablecolortextcelloptionstype)
              * [`fn withTableCellDisplayMode(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablecolortextcelloptionstypewithtablecelldisplaymode)
          * [`obj TableColoredBackgroundCellOptions`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstablecoloredbackgroundcelloptions)
            * [`fn withApplyToRow(value=true)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablecoloredbackgroundcelloptionswithapplytorow)
            * [`fn withMode(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablecoloredbackgroundcelloptionswithmode)
            * [`fn withType(value="color-background")`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablecoloredbackgroundcelloptionswithtype)
            * [`fn withTypeMixin(value="color-background")`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablecoloredbackgroundcelloptionswithtypemixin)
            * [`fn withWrapText(value=true)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablecoloredbackgroundcelloptionswithwraptext)
            * [`obj type`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstablecoloredbackgroundcelloptionstype)
              * [`fn withTableCellDisplayMode(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablecoloredbackgroundcelloptionstypewithtablecelldisplaymode)
          * [`obj TableDataLinksCellOptions`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstabledatalinkscelloptions)
            * [`fn withType(value="data-links")`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstabledatalinkscelloptionswithtype)
            * [`fn withTypeMixin(value="data-links")`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstabledatalinkscelloptionswithtypemixin)
            * [`obj type`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstabledatalinkscelloptionstype)
              * [`fn withTableCellDisplayMode(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstabledatalinkscelloptionstypewithtablecelldisplaymode)
          * [`obj TableImageCellOptions`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstableimagecelloptions)
            * [`fn withAlt(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstableimagecelloptionswithalt)
            * [`fn withTitle(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstableimagecelloptionswithtitle)
            * [`fn withType(value="image")`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstableimagecelloptionswithtype)
            * [`fn withTypeMixin(value="image")`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstableimagecelloptionswithtypemixin)
            * [`obj type`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstableimagecelloptionstype)
              * [`fn withTableCellDisplayMode(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstableimagecelloptionstypewithtablecelldisplaymode)
          * [`obj TableJsonViewCellOptions`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstablejsonviewcelloptions)
            * [`fn withType(value="json-view")`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablejsonviewcelloptionswithtype)
            * [`fn withTypeMixin(value="json-view")`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablejsonviewcelloptionswithtypemixin)
            * [`obj type`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstablejsonviewcelloptionstype)
              * [`fn withTableCellDisplayMode(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablejsonviewcelloptionstypewithtablecelldisplaymode)
          * [`obj TableSparklineCellOptions`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptions)
            * [`fn withAxisBorderShow(value=true)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithaxisbordershow)
            * [`fn withAxisCenteredZero(value=true)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithaxiscenteredzero)
            * [`fn withAxisColorMode(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithaxiscolormode)
            * [`fn withAxisGridShow(value=true)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithaxisgridshow)
            * [`fn withAxisLabel(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithaxislabel)
            * [`fn withAxisPlacement(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithaxisplacement)
            * [`fn withAxisSoftMax(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithaxissoftmax)
            * [`fn withAxisSoftMin(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithaxissoftmin)
            * [`fn withAxisWidth(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithaxiswidth)
            * [`fn withBarAlignment(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithbaralignment)
            * [`fn withBarMaxWidth(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithbarmaxwidth)
            * [`fn withBarWidthFactor(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithbarwidthfactor)
            * [`fn withDrawStyle(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithdrawstyle)
            * [`fn withFillBelowTo(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithfillbelowto)
            * [`fn withFillColor(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithfillcolor)
            * [`fn withFillOpacity(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithfillopacity)
            * [`fn withGradientMode(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithgradientmode)
            * [`fn withHideFrom(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithhidefrom)
            * [`fn withHideFromMixin(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithhidefrommixin)
            * [`fn withHideValue(value=true)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithhidevalue)
            * [`fn withInsertNulls(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithinsertnulls)
            * [`fn withInsertNullsMixin(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithinsertnullsmixin)
            * [`fn withLineColor(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithlinecolor)
            * [`fn withLineInterpolation(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithlineinterpolation)
            * [`fn withLineStyle(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithlinestyle)
            * [`fn withLineStyleMixin(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithlinestylemixin)
            * [`fn withLineWidth(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithlinewidth)
            * [`fn withPointColor(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithpointcolor)
            * [`fn withPointSize(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithpointsize)
            * [`fn withPointSymbol(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithpointsymbol)
            * [`fn withScaleDistribution(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithscaledistribution)
            * [`fn withScaleDistributionMixin(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithscaledistributionmixin)
            * [`fn withShowPoints(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithshowpoints)
            * [`fn withSpanNulls(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithspannulls)
            * [`fn withSpanNullsMixin(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithspannullsmixin)
            * [`fn withStacking(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithstacking)
            * [`fn withStackingMixin(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithstackingmixin)
            * [`fn withThresholdsStyle(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswiththresholdsstyle)
            * [`fn withThresholdsStyleMixin(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswiththresholdsstylemixin)
            * [`fn withTransform(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithtransform)
            * [`fn withType(value="sparkline")`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithtype)
            * [`fn withTypeMixin(value="sparkline")`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionswithtypemixin)
            * [`obj hideFrom`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionshidefrom)
              * [`fn withLegend(value=true)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionshidefromwithlegend)
              * [`fn withTooltip(value=true)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionshidefromwithtooltip)
              * [`fn withViz(value=true)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionshidefromwithviz)
            * [`obj lineStyle`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionslinestyle)
              * [`fn withDash(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionslinestylewithdash)
              * [`fn withDashMixin(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionslinestylewithdashmixin)
              * [`fn withFill(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionslinestylewithfill)
            * [`obj scaleDistribution`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionsscaledistribution)
              * [`fn withLinearThreshold(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionsscaledistributionwithlinearthreshold)
              * [`fn withLog(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionsscaledistributionwithlog)
              * [`fn withType(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionsscaledistributionwithtype)
            * [`obj stacking`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionsstacking)
              * [`fn withGroup(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionsstackingwithgroup)
              * [`fn withMode(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionsstackingwithmode)
            * [`obj thresholdsStyle`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionsthresholdsstyle)
              * [`fn withMode(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionsthresholdsstylewithmode)
            * [`obj type`](#obj-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionstype)
              * [`fn withTableCellDisplayMode(value)`](#fn-fieldconfigdefaultscustomcelloptionstablecelloptionstablesparklinecelloptionstypewithtablecelldisplaymode)
* [`obj libraryPanel`](#obj-librarypanel)
  * [`fn withName(value)`](#fn-librarypanelwithname)
  * [`fn withUid(value)`](#fn-librarypanelwithuid)
* [`obj options`](#obj-options)
  * [`fn withCellHeight(value="sm")`](#fn-optionswithcellheight)
  * [`fn withFooter(value={"countRows": false,"reducer": null,"show": false})`](#fn-optionswithfooter)
  * [`fn withFooterMixin(value={"countRows": false,"reducer": null,"show": false})`](#fn-optionswithfootermixin)
  * [`fn withFrameIndex(value=0)`](#fn-optionswithframeindex)
  * [`fn withShowHeader(value=true)`](#fn-optionswithshowheader)
  * [`fn withShowTypeIcons(value=true)`](#fn-optionswithshowtypeicons)
  * [`fn withSortBy(value)`](#fn-optionswithsortby)
  * [`fn withSortByMixin(value)`](#fn-optionswithsortbymixin)
  * [`obj footer`](#obj-optionsfooter)
    * [`fn withCountRows(value=true)`](#fn-optionsfooterwithcountrows)
    * [`fn withEnablePagination(value=true)`](#fn-optionsfooterwithenablepagination)
    * [`fn withFields(value)`](#fn-optionsfooterwithfields)
    * [`fn withFieldsMixin(value)`](#fn-optionsfooterwithfieldsmixin)
    * [`fn withReducer(value)`](#fn-optionsfooterwithreducer)
    * [`fn withReducerMixin(value)`](#fn-optionsfooterwithreducermixin)
    * [`fn withShow(value=true)`](#fn-optionsfooterwithshow)
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

Creates a new table panel with a title.
### obj fieldConfig


#### obj fieldConfig.defaults


##### obj fieldConfig.defaults.custom


###### fn fieldConfig.defaults.custom.withAlign

```jsonnet
fieldConfig.defaults.custom.withAlign(value="auto")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"auto"`
   - valid values: `"auto"`, `"left"`, `"right"`, `"center"`

TODO -- should not be table specific!
TODO docs
###### fn fieldConfig.defaults.custom.withCellOptions

```jsonnet
fieldConfig.defaults.custom.withCellOptions(value)
```

PARAMETERS:

* **value** (`object`)


###### fn fieldConfig.defaults.custom.withCellOptionsMixin

```jsonnet
fieldConfig.defaults.custom.withCellOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)


###### fn fieldConfig.defaults.custom.withDisplayMode

```jsonnet
fieldConfig.defaults.custom.withDisplayMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"auto"`, `"color-text"`, `"color-background"`, `"color-background-solid"`, `"gradient-gauge"`, `"lcd-gauge"`, `"json-view"`, `"basic"`, `"image"`, `"gauge"`, `"sparkline"`, `"data-links"`, `"custom"`, `"actions"`

Internally, this is the "type" of cell that's being displayed
in the table such as colored text, JSON, gauge, etc.
The color-background-solid, gradient-gauge, and lcd-gauge
modes are deprecated in favor of new cell subOptions
###### fn fieldConfig.defaults.custom.withFilterable

```jsonnet
fieldConfig.defaults.custom.withFilterable(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


###### fn fieldConfig.defaults.custom.withHidden

```jsonnet
fieldConfig.defaults.custom.withHidden(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

?? default is missing or false ??
###### fn fieldConfig.defaults.custom.withHideHeader

```jsonnet
fieldConfig.defaults.custom.withHideHeader(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Hides any header for a column, useful for columns that show some static content or buttons.
###### fn fieldConfig.defaults.custom.withInspect

```jsonnet
fieldConfig.defaults.custom.withInspect(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


###### fn fieldConfig.defaults.custom.withMinWidth

```jsonnet
fieldConfig.defaults.custom.withMinWidth(value)
```

PARAMETERS:

* **value** (`number`)


###### fn fieldConfig.defaults.custom.withWidth

```jsonnet
fieldConfig.defaults.custom.withWidth(value)
```

PARAMETERS:

* **value** (`number`)


###### obj fieldConfig.defaults.custom.cellOptions


####### fn fieldConfig.defaults.custom.cellOptions.withTableCellOptions

```jsonnet
fieldConfig.defaults.custom.cellOptions.withTableCellOptions(value)
```

PARAMETERS:

* **value** (`object`)

Table cell options. Each cell has a display mode
and other potential options for that display.
####### fn fieldConfig.defaults.custom.cellOptions.withTableCellOptionsMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.withTableCellOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Table cell options. Each cell has a display mode
and other potential options for that display.
####### obj fieldConfig.defaults.custom.cellOptions.TableCellOptions


######## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableActionsCellOptions

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableActionsCellOptions(value)
```

PARAMETERS:

* **value** (`object`)

Show actions in the cell
######## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableActionsCellOptionsMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableActionsCellOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Show actions in the cell
######## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableAutoCellOptions

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableAutoCellOptions(value)
```

PARAMETERS:

* **value** (`object`)

Auto mode table cell options
######## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableAutoCellOptionsMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableAutoCellOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Auto mode table cell options
######## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableBarGaugeCellOptions

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableBarGaugeCellOptions(value)
```

PARAMETERS:

* **value** (`object`)

Gauge cell options
######## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableBarGaugeCellOptionsMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableBarGaugeCellOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Gauge cell options
######## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableColorTextCellOptions

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableColorTextCellOptions(value)
```

PARAMETERS:

* **value** (`object`)

Colored text cell options
######## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableColorTextCellOptionsMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableColorTextCellOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Colored text cell options
######## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableColoredBackgroundCellOptions

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableColoredBackgroundCellOptions(value)
```

PARAMETERS:

* **value** (`object`)

Colored background cell options
######## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableColoredBackgroundCellOptionsMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableColoredBackgroundCellOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Colored background cell options
######## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableDataLinksCellOptions

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableDataLinksCellOptions(value)
```

PARAMETERS:

* **value** (`object`)

Show data links in the cell
######## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableDataLinksCellOptionsMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableDataLinksCellOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Show data links in the cell
######## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableImageCellOptions

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableImageCellOptions(value)
```

PARAMETERS:

* **value** (`object`)

Json view cell options
######## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableImageCellOptionsMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableImageCellOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Json view cell options
######## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableJsonViewCellOptions

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableJsonViewCellOptions(value)
```

PARAMETERS:

* **value** (`object`)

Json view cell options
######## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableJsonViewCellOptionsMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableJsonViewCellOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Json view cell options
######## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableSparklineCellOptions

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableSparklineCellOptions(value)
```

PARAMETERS:

* **value** (`object`)

Sparkline cell options
######## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableSparklineCellOptionsMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.withTableSparklineCellOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Sparkline cell options
######## obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableActionsCellOptions


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableActionsCellOptions.withType

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableActionsCellOptions.withType(value="actions")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"actions"`


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableActionsCellOptions.withTypeMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableActionsCellOptions.withTypeMixin(value="actions")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"actions"`


######### obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableActionsCellOptions.type


########## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableActionsCellOptions.type.withTableCellDisplayMode

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableActionsCellOptions.type.withTableCellDisplayMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"auto"`, `"color-text"`, `"color-background"`, `"color-background-solid"`, `"gradient-gauge"`, `"lcd-gauge"`, `"json-view"`, `"basic"`, `"image"`, `"gauge"`, `"sparkline"`, `"data-links"`, `"custom"`, `"actions"`

Internally, this is the "type" of cell that's being displayed
in the table such as colored text, JSON, gauge, etc.
The color-background-solid, gradient-gauge, and lcd-gauge
modes are deprecated in favor of new cell subOptions
######## obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableAutoCellOptions


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableAutoCellOptions.withType

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableAutoCellOptions.withType(value="auto")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"auto"`


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableAutoCellOptions.withTypeMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableAutoCellOptions.withTypeMixin(value="auto")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"auto"`


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableAutoCellOptions.withWrapText

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableAutoCellOptions.withWrapText(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


######### obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableAutoCellOptions.type


########## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableAutoCellOptions.type.withTableCellDisplayMode

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableAutoCellOptions.type.withTableCellDisplayMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"auto"`, `"color-text"`, `"color-background"`, `"color-background-solid"`, `"gradient-gauge"`, `"lcd-gauge"`, `"json-view"`, `"basic"`, `"image"`, `"gauge"`, `"sparkline"`, `"data-links"`, `"custom"`, `"actions"`

Internally, this is the "type" of cell that's being displayed
in the table such as colored text, JSON, gauge, etc.
The color-background-solid, gradient-gauge, and lcd-gauge
modes are deprecated in favor of new cell subOptions
######## obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableBarGaugeCellOptions


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableBarGaugeCellOptions.withMode

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableBarGaugeCellOptions.withMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"basic"`, `"lcd"`, `"gradient"`

Enum expressing the possible display modes
for the bar gauge component of Grafana UI
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableBarGaugeCellOptions.withType

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableBarGaugeCellOptions.withType(value="gauge")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"gauge"`


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableBarGaugeCellOptions.withTypeMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableBarGaugeCellOptions.withTypeMixin(value="gauge")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"gauge"`


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableBarGaugeCellOptions.withValueDisplayMode

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableBarGaugeCellOptions.withValueDisplayMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"color"`, `"text"`, `"hidden"`

Allows for the table cell gauge display type to set the gauge mode.
######### obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableBarGaugeCellOptions.type


########## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableBarGaugeCellOptions.type.withTableCellDisplayMode

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableBarGaugeCellOptions.type.withTableCellDisplayMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"auto"`, `"color-text"`, `"color-background"`, `"color-background-solid"`, `"gradient-gauge"`, `"lcd-gauge"`, `"json-view"`, `"basic"`, `"image"`, `"gauge"`, `"sparkline"`, `"data-links"`, `"custom"`, `"actions"`

Internally, this is the "type" of cell that's being displayed
in the table such as colored text, JSON, gauge, etc.
The color-background-solid, gradient-gauge, and lcd-gauge
modes are deprecated in favor of new cell subOptions
######## obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColorTextCellOptions


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColorTextCellOptions.withType

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColorTextCellOptions.withType(value="color-text")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"color-text"`


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColorTextCellOptions.withTypeMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColorTextCellOptions.withTypeMixin(value="color-text")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"color-text"`


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColorTextCellOptions.withWrapText

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColorTextCellOptions.withWrapText(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


######### obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColorTextCellOptions.type


########## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColorTextCellOptions.type.withTableCellDisplayMode

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColorTextCellOptions.type.withTableCellDisplayMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"auto"`, `"color-text"`, `"color-background"`, `"color-background-solid"`, `"gradient-gauge"`, `"lcd-gauge"`, `"json-view"`, `"basic"`, `"image"`, `"gauge"`, `"sparkline"`, `"data-links"`, `"custom"`, `"actions"`

Internally, this is the "type" of cell that's being displayed
in the table such as colored text, JSON, gauge, etc.
The color-background-solid, gradient-gauge, and lcd-gauge
modes are deprecated in favor of new cell subOptions
######## obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColoredBackgroundCellOptions


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColoredBackgroundCellOptions.withApplyToRow

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColoredBackgroundCellOptions.withApplyToRow(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColoredBackgroundCellOptions.withMode

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColoredBackgroundCellOptions.withMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"basic"`, `"gradient"`

Display mode to the "Colored Background" display
mode for table cells. Either displays a solid color (basic mode)
or a gradient.
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColoredBackgroundCellOptions.withType

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColoredBackgroundCellOptions.withType(value="color-background")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"color-background"`


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColoredBackgroundCellOptions.withTypeMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColoredBackgroundCellOptions.withTypeMixin(value="color-background")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"color-background"`


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColoredBackgroundCellOptions.withWrapText

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColoredBackgroundCellOptions.withWrapText(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


######### obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColoredBackgroundCellOptions.type


########## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColoredBackgroundCellOptions.type.withTableCellDisplayMode

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableColoredBackgroundCellOptions.type.withTableCellDisplayMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"auto"`, `"color-text"`, `"color-background"`, `"color-background-solid"`, `"gradient-gauge"`, `"lcd-gauge"`, `"json-view"`, `"basic"`, `"image"`, `"gauge"`, `"sparkline"`, `"data-links"`, `"custom"`, `"actions"`

Internally, this is the "type" of cell that's being displayed
in the table such as colored text, JSON, gauge, etc.
The color-background-solid, gradient-gauge, and lcd-gauge
modes are deprecated in favor of new cell subOptions
######## obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableDataLinksCellOptions


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableDataLinksCellOptions.withType

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableDataLinksCellOptions.withType(value="data-links")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"data-links"`


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableDataLinksCellOptions.withTypeMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableDataLinksCellOptions.withTypeMixin(value="data-links")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"data-links"`


######### obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableDataLinksCellOptions.type


########## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableDataLinksCellOptions.type.withTableCellDisplayMode

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableDataLinksCellOptions.type.withTableCellDisplayMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"auto"`, `"color-text"`, `"color-background"`, `"color-background-solid"`, `"gradient-gauge"`, `"lcd-gauge"`, `"json-view"`, `"basic"`, `"image"`, `"gauge"`, `"sparkline"`, `"data-links"`, `"custom"`, `"actions"`

Internally, this is the "type" of cell that's being displayed
in the table such as colored text, JSON, gauge, etc.
The color-background-solid, gradient-gauge, and lcd-gauge
modes are deprecated in favor of new cell subOptions
######## obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableImageCellOptions


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableImageCellOptions.withAlt

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableImageCellOptions.withAlt(value)
```

PARAMETERS:

* **value** (`string`)


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableImageCellOptions.withTitle

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableImageCellOptions.withTitle(value)
```

PARAMETERS:

* **value** (`string`)


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableImageCellOptions.withType

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableImageCellOptions.withType(value="image")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"image"`


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableImageCellOptions.withTypeMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableImageCellOptions.withTypeMixin(value="image")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"image"`


######### obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableImageCellOptions.type


########## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableImageCellOptions.type.withTableCellDisplayMode

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableImageCellOptions.type.withTableCellDisplayMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"auto"`, `"color-text"`, `"color-background"`, `"color-background-solid"`, `"gradient-gauge"`, `"lcd-gauge"`, `"json-view"`, `"basic"`, `"image"`, `"gauge"`, `"sparkline"`, `"data-links"`, `"custom"`, `"actions"`

Internally, this is the "type" of cell that's being displayed
in the table such as colored text, JSON, gauge, etc.
The color-background-solid, gradient-gauge, and lcd-gauge
modes are deprecated in favor of new cell subOptions
######## obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableJsonViewCellOptions


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableJsonViewCellOptions.withType

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableJsonViewCellOptions.withType(value="json-view")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"json-view"`


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableJsonViewCellOptions.withTypeMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableJsonViewCellOptions.withTypeMixin(value="json-view")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"json-view"`


######### obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableJsonViewCellOptions.type


########## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableJsonViewCellOptions.type.withTableCellDisplayMode

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableJsonViewCellOptions.type.withTableCellDisplayMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"auto"`, `"color-text"`, `"color-background"`, `"color-background-solid"`, `"gradient-gauge"`, `"lcd-gauge"`, `"json-view"`, `"basic"`, `"image"`, `"gauge"`, `"sparkline"`, `"data-links"`, `"custom"`, `"actions"`

Internally, this is the "type" of cell that's being displayed
in the table such as colored text, JSON, gauge, etc.
The color-background-solid, gradient-gauge, and lcd-gauge
modes are deprecated in favor of new cell subOptions
######## obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withAxisBorderShow

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withAxisBorderShow(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withAxisCenteredZero

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withAxisCenteredZero(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withAxisColorMode

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withAxisColorMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"text"`, `"series"`

TODO docs
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withAxisGridShow

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withAxisGridShow(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withAxisLabel

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withAxisLabel(value)
```

PARAMETERS:

* **value** (`string`)


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withAxisPlacement

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withAxisPlacement(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"auto"`, `"top"`, `"right"`, `"bottom"`, `"left"`, `"hidden"`

TODO docs
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withAxisSoftMax

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withAxisSoftMax(value)
```

PARAMETERS:

* **value** (`number`)


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withAxisSoftMin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withAxisSoftMin(value)
```

PARAMETERS:

* **value** (`number`)


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withAxisWidth

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withAxisWidth(value)
```

PARAMETERS:

* **value** (`number`)


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withBarAlignment

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withBarAlignment(value)
```

PARAMETERS:

* **value** (`integer`)
   - valid values: `-1`, `0`, `1`

TODO docs
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withBarMaxWidth

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withBarMaxWidth(value)
```

PARAMETERS:

* **value** (`number`)


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withBarWidthFactor

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withBarWidthFactor(value)
```

PARAMETERS:

* **value** (`number`)


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withDrawStyle

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withDrawStyle(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"line"`, `"bars"`, `"points"`

TODO docs
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withFillBelowTo

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withFillBelowTo(value)
```

PARAMETERS:

* **value** (`string`)


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withFillColor

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withFillColor(value)
```

PARAMETERS:

* **value** (`string`)


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withFillOpacity

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withFillOpacity(value)
```

PARAMETERS:

* **value** (`number`)


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withGradientMode

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withGradientMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"none"`, `"opacity"`, `"hue"`, `"scheme"`

TODO docs
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withHideFrom

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withHideFrom(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withHideFromMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withHideFromMixin(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withHideValue

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withHideValue(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withInsertNulls

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withInsertNulls(value)
```

PARAMETERS:

* **value** (`boolean`,`number`)


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withInsertNullsMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withInsertNullsMixin(value)
```

PARAMETERS:

* **value** (`boolean`,`number`)


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withLineColor

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withLineColor(value)
```

PARAMETERS:

* **value** (`string`)


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withLineInterpolation

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withLineInterpolation(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"linear"`, `"smooth"`, `"stepBefore"`, `"stepAfter"`

TODO docs
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withLineStyle

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withLineStyle(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withLineStyleMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withLineStyleMixin(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withLineWidth

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withLineWidth(value)
```

PARAMETERS:

* **value** (`number`)


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withPointColor

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withPointColor(value)
```

PARAMETERS:

* **value** (`string`)


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withPointSize

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withPointSize(value)
```

PARAMETERS:

* **value** (`number`)


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withPointSymbol

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withPointSymbol(value)
```

PARAMETERS:

* **value** (`string`)


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withScaleDistribution

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withScaleDistribution(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withScaleDistributionMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withScaleDistributionMixin(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withShowPoints

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withShowPoints(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"auto"`, `"never"`, `"always"`

TODO docs
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withSpanNulls

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withSpanNulls(value)
```

PARAMETERS:

* **value** (`boolean`,`number`)

Indicate if null values should be treated as gaps or connected.
When the value is a number, it represents the maximum delta in the
X axis that should be considered connected.  For timeseries, this is milliseconds
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withSpanNullsMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withSpanNullsMixin(value)
```

PARAMETERS:

* **value** (`boolean`,`number`)

Indicate if null values should be treated as gaps or connected.
When the value is a number, it represents the maximum delta in the
X axis that should be considered connected.  For timeseries, this is milliseconds
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withStacking

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withStacking(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withStackingMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withStackingMixin(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withThresholdsStyle

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withThresholdsStyle(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withThresholdsStyleMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withThresholdsStyleMixin(value)
```

PARAMETERS:

* **value** (`object`)

TODO docs
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withTransform

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withTransform(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"constant"`, `"negative-Y"`

TODO docs
######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withType

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withType(value="sparkline")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"sparkline"`


######### fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withTypeMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.withTypeMixin(value="sparkline")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"sparkline"`


######### obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.hideFrom


########## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.hideFrom.withLegend

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.hideFrom.withLegend(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


########## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.hideFrom.withTooltip

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.hideFrom.withTooltip(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


########## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.hideFrom.withViz

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.hideFrom.withViz(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


######### obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.lineStyle


########## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.lineStyle.withDash

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.lineStyle.withDash(value)
```

PARAMETERS:

* **value** (`array`)


########## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.lineStyle.withDashMixin

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.lineStyle.withDashMixin(value)
```

PARAMETERS:

* **value** (`array`)


########## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.lineStyle.withFill

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.lineStyle.withFill(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"solid"`, `"dash"`, `"dot"`, `"square"`


######### obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.scaleDistribution


########## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.scaleDistribution.withLinearThreshold

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.scaleDistribution.withLinearThreshold(value)
```

PARAMETERS:

* **value** (`number`)


########## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.scaleDistribution.withLog

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.scaleDistribution.withLog(value)
```

PARAMETERS:

* **value** (`number`)


########## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.scaleDistribution.withType

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.scaleDistribution.withType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"linear"`, `"log"`, `"ordinal"`, `"symlog"`

TODO docs
######### obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.stacking


########## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.stacking.withGroup

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.stacking.withGroup(value)
```

PARAMETERS:

* **value** (`string`)


########## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.stacking.withMode

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.stacking.withMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"none"`, `"normal"`, `"percent"`

TODO docs
######### obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.thresholdsStyle


########## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.thresholdsStyle.withMode

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.thresholdsStyle.withMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"off"`, `"line"`, `"dashed"`, `"area"`, `"line+area"`, `"dashed+area"`, `"series"`

TODO docs
######### obj fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.type


########## fn fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.type.withTableCellDisplayMode

```jsonnet
fieldConfig.defaults.custom.cellOptions.TableCellOptions.TableSparklineCellOptions.type.withTableCellDisplayMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"auto"`, `"color-text"`, `"color-background"`, `"color-background-solid"`, `"gradient-gauge"`, `"lcd-gauge"`, `"json-view"`, `"basic"`, `"image"`, `"gauge"`, `"sparkline"`, `"data-links"`, `"custom"`, `"actions"`

Internally, this is the "type" of cell that's being displayed
in the table such as colored text, JSON, gauge, etc.
The color-background-solid, gradient-gauge, and lcd-gauge
modes are deprecated in favor of new cell subOptions
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


#### fn options.withCellHeight

```jsonnet
options.withCellHeight(value="sm")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"sm"`
   - valid values: `"sm"`, `"md"`, `"lg"`, `"auto"`

Height of a table cell
#### fn options.withFooter

```jsonnet
options.withFooter(value={"countRows": false,"reducer": null,"show": false})
```

PARAMETERS:

* **value** (`object`)
   - default value: `{"countRows": false,"reducer": null,"show": false}`

Footer options
#### fn options.withFooterMixin

```jsonnet
options.withFooterMixin(value={"countRows": false,"reducer": null,"show": false})
```

PARAMETERS:

* **value** (`object`)
   - default value: `{"countRows": false,"reducer": null,"show": false}`

Footer options
#### fn options.withFrameIndex

```jsonnet
options.withFrameIndex(value=0)
```

PARAMETERS:

* **value** (`number`)
   - default value: `0`

Represents the index of the selected frame
#### fn options.withShowHeader

```jsonnet
options.withShowHeader(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Controls whether the panel should show the header
#### fn options.withShowTypeIcons

```jsonnet
options.withShowTypeIcons(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Controls whether the header should show icons for the column types
#### fn options.withSortBy

```jsonnet
options.withSortBy(value)
```

PARAMETERS:

* **value** (`array`)

Used to control row sorting
#### fn options.withSortByMixin

```jsonnet
options.withSortByMixin(value)
```

PARAMETERS:

* **value** (`array`)

Used to control row sorting
#### obj options.footer


##### fn options.footer.withCountRows

```jsonnet
options.footer.withCountRows(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn options.footer.withEnablePagination

```jsonnet
options.footer.withEnablePagination(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


##### fn options.footer.withFields

```jsonnet
options.footer.withFields(value)
```

PARAMETERS:

* **value** (`array`)


##### fn options.footer.withFieldsMixin

```jsonnet
options.footer.withFieldsMixin(value)
```

PARAMETERS:

* **value** (`array`)


##### fn options.footer.withReducer

```jsonnet
options.footer.withReducer(value)
```

PARAMETERS:

* **value** (`array`)

actually 1 value
##### fn options.footer.withReducerMixin

```jsonnet
options.footer.withReducerMixin(value)
```

PARAMETERS:

* **value** (`array`)

actually 1 value
##### fn options.footer.withShow

```jsonnet
options.footer.withShow(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


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
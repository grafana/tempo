# librarypanel

grafonnet.librarypanel

## Subpackages

* [model.fieldConfig.defaults.actions](model/fieldConfig/defaults/actions/index.md)
* [model.fieldConfig.defaults.links](model/fieldConfig/defaults/links.md)
* [model.fieldConfig.defaults.thresholds.steps](model/fieldConfig/defaults/thresholds/steps.md)
* [model.fieldConfig.overrides](model/fieldConfig/overrides/index.md)
* [model.links](model/links.md)
* [model.transformations](model/transformations.md)

## Index

* [`fn withDescription(value)`](#fn-withdescription)
* [`fn withFolderUid(value)`](#fn-withfolderuid)
* [`fn withMeta(value)`](#fn-withmeta)
* [`fn withMetaMixin(value)`](#fn-withmetamixin)
* [`fn withModel(value)`](#fn-withmodel)
* [`fn withModelMixin(value)`](#fn-withmodelmixin)
* [`fn withName(value)`](#fn-withname)
* [`fn withSchemaVersion(value)`](#fn-withschemaversion)
* [`fn withType(value)`](#fn-withtype)
* [`fn withUid(value)`](#fn-withuid)
* [`fn withVersion(value)`](#fn-withversion)
* [`obj meta`](#obj-meta)
  * [`fn withConnectedDashboards(value)`](#fn-metawithconnecteddashboards)
  * [`fn withCreated(value)`](#fn-metawithcreated)
  * [`fn withCreatedBy(value)`](#fn-metawithcreatedby)
  * [`fn withCreatedByMixin(value)`](#fn-metawithcreatedbymixin)
  * [`fn withFolderName(value)`](#fn-metawithfoldername)
  * [`fn withFolderUid(value)`](#fn-metawithfolderuid)
  * [`fn withUpdated(value)`](#fn-metawithupdated)
  * [`fn withUpdatedBy(value)`](#fn-metawithupdatedby)
  * [`fn withUpdatedByMixin(value)`](#fn-metawithupdatedbymixin)
  * [`obj createdBy`](#obj-metacreatedby)
    * [`fn withAvatarUrl(value)`](#fn-metacreatedbywithavatarurl)
    * [`fn withId(value)`](#fn-metacreatedbywithid)
    * [`fn withName(value)`](#fn-metacreatedbywithname)
  * [`obj updatedBy`](#obj-metaupdatedby)
    * [`fn withAvatarUrl(value)`](#fn-metaupdatedbywithavatarurl)
    * [`fn withId(value)`](#fn-metaupdatedbywithid)
    * [`fn withName(value)`](#fn-metaupdatedbywithname)
* [`obj model`](#obj-model)
  * [`fn withCacheTimeout(value)`](#fn-modelwithcachetimeout)
  * [`fn withDatasource(value)`](#fn-modelwithdatasource)
  * [`fn withDatasourceMixin(value)`](#fn-modelwithdatasourcemixin)
  * [`fn withDescription(value)`](#fn-modelwithdescription)
  * [`fn withFieldConfig(value)`](#fn-modelwithfieldconfig)
  * [`fn withFieldConfigMixin(value)`](#fn-modelwithfieldconfigmixin)
  * [`fn withHideTimeOverride(value=true)`](#fn-modelwithhidetimeoverride)
  * [`fn withInterval(value)`](#fn-modelwithinterval)
  * [`fn withLinks(value)`](#fn-modelwithlinks)
  * [`fn withLinksMixin(value)`](#fn-modelwithlinksmixin)
  * [`fn withMaxDataPoints(value)`](#fn-modelwithmaxdatapoints)
  * [`fn withMaxPerRow(value)`](#fn-modelwithmaxperrow)
  * [`fn withOptions(value)`](#fn-modelwithoptions)
  * [`fn withOptionsMixin(value)`](#fn-modelwithoptionsmixin)
  * [`fn withPluginVersion(value)`](#fn-modelwithpluginversion)
  * [`fn withQueryCachingTTL(value)`](#fn-modelwithquerycachingttl)
  * [`fn withRepeat(value)`](#fn-modelwithrepeat)
  * [`fn withRepeatDirection(value="h")`](#fn-modelwithrepeatdirection)
  * [`fn withTargets(value)`](#fn-modelwithtargets)
  * [`fn withTargetsMixin(value)`](#fn-modelwithtargetsmixin)
  * [`fn withTimeCompare(value)`](#fn-modelwithtimecompare)
  * [`fn withTimeFrom(value)`](#fn-modelwithtimefrom)
  * [`fn withTimeShift(value)`](#fn-modelwithtimeshift)
  * [`fn withTitle(value)`](#fn-modelwithtitle)
  * [`fn withTransformations(value)`](#fn-modelwithtransformations)
  * [`fn withTransformationsMixin(value)`](#fn-modelwithtransformationsmixin)
  * [`fn withTransparent(value=true)`](#fn-modelwithtransparent)
  * [`fn withType(value)`](#fn-modelwithtype)
  * [`obj datasource`](#obj-modeldatasource)
    * [`fn withType(value)`](#fn-modeldatasourcewithtype)
    * [`fn withUid(value)`](#fn-modeldatasourcewithuid)
  * [`obj fieldConfig`](#obj-modelfieldconfig)
    * [`fn withDefaults(value)`](#fn-modelfieldconfigwithdefaults)
    * [`fn withDefaultsMixin(value)`](#fn-modelfieldconfigwithdefaultsmixin)
    * [`fn withOverrides(value)`](#fn-modelfieldconfigwithoverrides)
    * [`fn withOverridesMixin(value)`](#fn-modelfieldconfigwithoverridesmixin)
    * [`obj defaults`](#obj-modelfieldconfigdefaults)
      * [`fn withActions(value)`](#fn-modelfieldconfigdefaultswithactions)
      * [`fn withActionsMixin(value)`](#fn-modelfieldconfigdefaultswithactionsmixin)
      * [`fn withColor(value)`](#fn-modelfieldconfigdefaultswithcolor)
      * [`fn withColorMixin(value)`](#fn-modelfieldconfigdefaultswithcolormixin)
      * [`fn withCustom(value)`](#fn-modelfieldconfigdefaultswithcustom)
      * [`fn withCustomMixin(value)`](#fn-modelfieldconfigdefaultswithcustommixin)
      * [`fn withDecimals(value)`](#fn-modelfieldconfigdefaultswithdecimals)
      * [`fn withDescription(value)`](#fn-modelfieldconfigdefaultswithdescription)
      * [`fn withDisplayName(value)`](#fn-modelfieldconfigdefaultswithdisplayname)
      * [`fn withDisplayNameFromDS(value)`](#fn-modelfieldconfigdefaultswithdisplaynamefromds)
      * [`fn withFieldMinMax(value=true)`](#fn-modelfieldconfigdefaultswithfieldminmax)
      * [`fn withFilterable(value=true)`](#fn-modelfieldconfigdefaultswithfilterable)
      * [`fn withLinks(value)`](#fn-modelfieldconfigdefaultswithlinks)
      * [`fn withLinksMixin(value)`](#fn-modelfieldconfigdefaultswithlinksmixin)
      * [`fn withMappings(value)`](#fn-modelfieldconfigdefaultswithmappings)
      * [`fn withMappingsMixin(value)`](#fn-modelfieldconfigdefaultswithmappingsmixin)
      * [`fn withMax(value)`](#fn-modelfieldconfigdefaultswithmax)
      * [`fn withMin(value)`](#fn-modelfieldconfigdefaultswithmin)
      * [`fn withNoValue(value)`](#fn-modelfieldconfigdefaultswithnovalue)
      * [`fn withPath(value)`](#fn-modelfieldconfigdefaultswithpath)
      * [`fn withThresholds(value)`](#fn-modelfieldconfigdefaultswiththresholds)
      * [`fn withThresholdsMixin(value)`](#fn-modelfieldconfigdefaultswiththresholdsmixin)
      * [`fn withUnit(value)`](#fn-modelfieldconfigdefaultswithunit)
      * [`fn withWriteable(value=true)`](#fn-modelfieldconfigdefaultswithwriteable)
      * [`obj color`](#obj-modelfieldconfigdefaultscolor)
        * [`fn withFixedColor(value)`](#fn-modelfieldconfigdefaultscolorwithfixedcolor)
        * [`fn withMode(value)`](#fn-modelfieldconfigdefaultscolorwithmode)
        * [`fn withSeriesBy(value)`](#fn-modelfieldconfigdefaultscolorwithseriesby)
      * [`obj mappings`](#obj-modelfieldconfigdefaultsmappings)
        * [`obj RangeMap`](#obj-modelfieldconfigdefaultsmappingsrangemap)
          * [`fn withOptions(value)`](#fn-modelfieldconfigdefaultsmappingsrangemapwithoptions)
          * [`fn withOptionsMixin(value)`](#fn-modelfieldconfigdefaultsmappingsrangemapwithoptionsmixin)
          * [`fn withType(value="range")`](#fn-modelfieldconfigdefaultsmappingsrangemapwithtype)
          * [`fn withTypeMixin(value="range")`](#fn-modelfieldconfigdefaultsmappingsrangemapwithtypemixin)
          * [`obj options`](#obj-modelfieldconfigdefaultsmappingsrangemapoptions)
            * [`fn withFrom(value)`](#fn-modelfieldconfigdefaultsmappingsrangemapoptionswithfrom)
            * [`fn withResult(value)`](#fn-modelfieldconfigdefaultsmappingsrangemapoptionswithresult)
            * [`fn withResultMixin(value)`](#fn-modelfieldconfigdefaultsmappingsrangemapoptionswithresultmixin)
            * [`fn withTo(value)`](#fn-modelfieldconfigdefaultsmappingsrangemapoptionswithto)
            * [`obj result`](#obj-modelfieldconfigdefaultsmappingsrangemapoptionsresult)
              * [`fn withColor(value)`](#fn-modelfieldconfigdefaultsmappingsrangemapoptionsresultwithcolor)
              * [`fn withIcon(value)`](#fn-modelfieldconfigdefaultsmappingsrangemapoptionsresultwithicon)
              * [`fn withIndex(value)`](#fn-modelfieldconfigdefaultsmappingsrangemapoptionsresultwithindex)
              * [`fn withText(value)`](#fn-modelfieldconfigdefaultsmappingsrangemapoptionsresultwithtext)
          * [`obj type`](#obj-modelfieldconfigdefaultsmappingsrangemaptype)
            * [`fn withMappingType(value)`](#fn-modelfieldconfigdefaultsmappingsrangemaptypewithmappingtype)
        * [`obj RegexMap`](#obj-modelfieldconfigdefaultsmappingsregexmap)
          * [`fn withOptions(value)`](#fn-modelfieldconfigdefaultsmappingsregexmapwithoptions)
          * [`fn withOptionsMixin(value)`](#fn-modelfieldconfigdefaultsmappingsregexmapwithoptionsmixin)
          * [`fn withType(value="regex")`](#fn-modelfieldconfigdefaultsmappingsregexmapwithtype)
          * [`fn withTypeMixin(value="regex")`](#fn-modelfieldconfigdefaultsmappingsregexmapwithtypemixin)
          * [`obj options`](#obj-modelfieldconfigdefaultsmappingsregexmapoptions)
            * [`fn withPattern(value)`](#fn-modelfieldconfigdefaultsmappingsregexmapoptionswithpattern)
            * [`fn withResult(value)`](#fn-modelfieldconfigdefaultsmappingsregexmapoptionswithresult)
            * [`fn withResultMixin(value)`](#fn-modelfieldconfigdefaultsmappingsregexmapoptionswithresultmixin)
            * [`obj result`](#obj-modelfieldconfigdefaultsmappingsregexmapoptionsresult)
              * [`fn withColor(value)`](#fn-modelfieldconfigdefaultsmappingsregexmapoptionsresultwithcolor)
              * [`fn withIcon(value)`](#fn-modelfieldconfigdefaultsmappingsregexmapoptionsresultwithicon)
              * [`fn withIndex(value)`](#fn-modelfieldconfigdefaultsmappingsregexmapoptionsresultwithindex)
              * [`fn withText(value)`](#fn-modelfieldconfigdefaultsmappingsregexmapoptionsresultwithtext)
          * [`obj type`](#obj-modelfieldconfigdefaultsmappingsregexmaptype)
            * [`fn withMappingType(value)`](#fn-modelfieldconfigdefaultsmappingsregexmaptypewithmappingtype)
        * [`obj SpecialValueMap`](#obj-modelfieldconfigdefaultsmappingsspecialvaluemap)
          * [`fn withOptions(value)`](#fn-modelfieldconfigdefaultsmappingsspecialvaluemapwithoptions)
          * [`fn withOptionsMixin(value)`](#fn-modelfieldconfigdefaultsmappingsspecialvaluemapwithoptionsmixin)
          * [`fn withType(value="special")`](#fn-modelfieldconfigdefaultsmappingsspecialvaluemapwithtype)
          * [`fn withTypeMixin(value="special")`](#fn-modelfieldconfigdefaultsmappingsspecialvaluemapwithtypemixin)
          * [`obj options`](#obj-modelfieldconfigdefaultsmappingsspecialvaluemapoptions)
            * [`fn withMatch(value)`](#fn-modelfieldconfigdefaultsmappingsspecialvaluemapoptionswithmatch)
            * [`fn withResult(value)`](#fn-modelfieldconfigdefaultsmappingsspecialvaluemapoptionswithresult)
            * [`fn withResultMixin(value)`](#fn-modelfieldconfigdefaultsmappingsspecialvaluemapoptionswithresultmixin)
            * [`obj result`](#obj-modelfieldconfigdefaultsmappingsspecialvaluemapoptionsresult)
              * [`fn withColor(value)`](#fn-modelfieldconfigdefaultsmappingsspecialvaluemapoptionsresultwithcolor)
              * [`fn withIcon(value)`](#fn-modelfieldconfigdefaultsmappingsspecialvaluemapoptionsresultwithicon)
              * [`fn withIndex(value)`](#fn-modelfieldconfigdefaultsmappingsspecialvaluemapoptionsresultwithindex)
              * [`fn withText(value)`](#fn-modelfieldconfigdefaultsmappingsspecialvaluemapoptionsresultwithtext)
          * [`obj type`](#obj-modelfieldconfigdefaultsmappingsspecialvaluemaptype)
            * [`fn withMappingType(value)`](#fn-modelfieldconfigdefaultsmappingsspecialvaluemaptypewithmappingtype)
        * [`obj ValueMap`](#obj-modelfieldconfigdefaultsmappingsvaluemap)
          * [`fn withOptions(value)`](#fn-modelfieldconfigdefaultsmappingsvaluemapwithoptions)
          * [`fn withOptionsMixin(value)`](#fn-modelfieldconfigdefaultsmappingsvaluemapwithoptionsmixin)
          * [`fn withType(value="value")`](#fn-modelfieldconfigdefaultsmappingsvaluemapwithtype)
          * [`fn withTypeMixin(value="value")`](#fn-modelfieldconfigdefaultsmappingsvaluemapwithtypemixin)
          * [`obj type`](#obj-modelfieldconfigdefaultsmappingsvaluemaptype)
            * [`fn withMappingType(value)`](#fn-modelfieldconfigdefaultsmappingsvaluemaptypewithmappingtype)
      * [`obj thresholds`](#obj-modelfieldconfigdefaultsthresholds)
        * [`fn withMode(value)`](#fn-modelfieldconfigdefaultsthresholdswithmode)
        * [`fn withSteps(value)`](#fn-modelfieldconfigdefaultsthresholdswithsteps)
        * [`fn withStepsMixin(value)`](#fn-modelfieldconfigdefaultsthresholdswithstepsmixin)

## Fields

### fn withDescription

```jsonnet
withDescription(value)
```

PARAMETERS:

* **value** (`string`)

Panel description
### fn withFolderUid

```jsonnet
withFolderUid(value)
```

PARAMETERS:

* **value** (`string`)

Folder UID
### fn withMeta

```jsonnet
withMeta(value)
```

PARAMETERS:

* **value** (`object`)

Object storage metadata
### fn withMetaMixin

```jsonnet
withMetaMixin(value)
```

PARAMETERS:

* **value** (`object`)

Object storage metadata
### fn withModel

```jsonnet
withModel(value)
```

PARAMETERS:

* **value** (`object`)

Dashboard panels are the basic visualization building blocks.
### fn withModelMixin

```jsonnet
withModelMixin(value)
```

PARAMETERS:

* **value** (`object`)

Dashboard panels are the basic visualization building blocks.
### fn withName

```jsonnet
withName(value)
```

PARAMETERS:

* **value** (`string`)

Panel name (also saved in the model)
### fn withSchemaVersion

```jsonnet
withSchemaVersion(value)
```

PARAMETERS:

* **value** (`integer`)

Dashboard version when this was saved (zero if unknown)
### fn withType

```jsonnet
withType(value)
```

PARAMETERS:

* **value** (`string`)

The panel type (from inside the model)
### fn withUid

```jsonnet
withUid(value)
```

PARAMETERS:

* **value** (`string`)

Library element UID
### fn withVersion

```jsonnet
withVersion(value)
```

PARAMETERS:

* **value** (`integer`)

panel version, incremented each time the dashboard is updated.
### obj meta


#### fn meta.withConnectedDashboards

```jsonnet
meta.withConnectedDashboards(value)
```

PARAMETERS:

* **value** (`integer`)


#### fn meta.withCreated

```jsonnet
meta.withCreated(value)
```

PARAMETERS:

* **value** (`string`)


#### fn meta.withCreatedBy

```jsonnet
meta.withCreatedBy(value)
```

PARAMETERS:

* **value** (`object`)


#### fn meta.withCreatedByMixin

```jsonnet
meta.withCreatedByMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn meta.withFolderName

```jsonnet
meta.withFolderName(value)
```

PARAMETERS:

* **value** (`string`)


#### fn meta.withFolderUid

```jsonnet
meta.withFolderUid(value)
```

PARAMETERS:

* **value** (`string`)


#### fn meta.withUpdated

```jsonnet
meta.withUpdated(value)
```

PARAMETERS:

* **value** (`string`)


#### fn meta.withUpdatedBy

```jsonnet
meta.withUpdatedBy(value)
```

PARAMETERS:

* **value** (`object`)


#### fn meta.withUpdatedByMixin

```jsonnet
meta.withUpdatedByMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### obj meta.createdBy


##### fn meta.createdBy.withAvatarUrl

```jsonnet
meta.createdBy.withAvatarUrl(value)
```

PARAMETERS:

* **value** (`string`)


##### fn meta.createdBy.withId

```jsonnet
meta.createdBy.withId(value)
```

PARAMETERS:

* **value** (`integer`)


##### fn meta.createdBy.withName

```jsonnet
meta.createdBy.withName(value)
```

PARAMETERS:

* **value** (`string`)


#### obj meta.updatedBy


##### fn meta.updatedBy.withAvatarUrl

```jsonnet
meta.updatedBy.withAvatarUrl(value)
```

PARAMETERS:

* **value** (`string`)


##### fn meta.updatedBy.withId

```jsonnet
meta.updatedBy.withId(value)
```

PARAMETERS:

* **value** (`integer`)


##### fn meta.updatedBy.withName

```jsonnet
meta.updatedBy.withName(value)
```

PARAMETERS:

* **value** (`string`)


### obj model


#### fn model.withCacheTimeout

```jsonnet
model.withCacheTimeout(value)
```

PARAMETERS:

* **value** (`string`)

Sets panel queries cache timeout.
#### fn model.withDatasource

```jsonnet
model.withDatasource(value)
```

PARAMETERS:

* **value** (`object`)

The datasource used in all targets.
#### fn model.withDatasourceMixin

```jsonnet
model.withDatasourceMixin(value)
```

PARAMETERS:

* **value** (`object`)

The datasource used in all targets.
#### fn model.withDescription

```jsonnet
model.withDescription(value)
```

PARAMETERS:

* **value** (`string`)

Panel description.
#### fn model.withFieldConfig

```jsonnet
model.withFieldConfig(value)
```

PARAMETERS:

* **value** (`object`)

The data model used in Grafana, namely the data frame, is a columnar-oriented table structure that unifies both time series and table query results.
Each column within this structure is called a field. A field can represent a single time series or table column.
Field options allow you to change how the data is displayed in your visualizations.
#### fn model.withFieldConfigMixin

```jsonnet
model.withFieldConfigMixin(value)
```

PARAMETERS:

* **value** (`object`)

The data model used in Grafana, namely the data frame, is a columnar-oriented table structure that unifies both time series and table query results.
Each column within this structure is called a field. A field can represent a single time series or table column.
Field options allow you to change how the data is displayed in your visualizations.
#### fn model.withHideTimeOverride

```jsonnet
model.withHideTimeOverride(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Controls if the timeFrom or timeShift overrides are shown in the panel header
#### fn model.withInterval

```jsonnet
model.withInterval(value)
```

PARAMETERS:

* **value** (`string`)

The min time interval setting defines a lower limit for the $__interval and $__interval_ms variables.
This value must be formatted as a number followed by a valid time
identifier like: "40s", "3d", etc.
See: https://grafana.com/docs/grafana/latest/panels-visualizations/query-transform-data/#query-options
#### fn model.withLinks

```jsonnet
model.withLinks(value)
```

PARAMETERS:

* **value** (`array`)

Panel links.
#### fn model.withLinksMixin

```jsonnet
model.withLinksMixin(value)
```

PARAMETERS:

* **value** (`array`)

Panel links.
#### fn model.withMaxDataPoints

```jsonnet
model.withMaxDataPoints(value)
```

PARAMETERS:

* **value** (`number`)

The maximum number of data points that the panel queries are retrieving.
#### fn model.withMaxPerRow

```jsonnet
model.withMaxPerRow(value)
```

PARAMETERS:

* **value** (`number`)

Option for repeated panels that controls max items per row
Only relevant for horizontally repeated panels
#### fn model.withOptions

```jsonnet
model.withOptions(value)
```

PARAMETERS:

* **value** (`object`)

It depends on the panel plugin. They are specified by the Options field in panel plugin schemas.
#### fn model.withOptionsMixin

```jsonnet
model.withOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

It depends on the panel plugin. They are specified by the Options field in panel plugin schemas.
#### fn model.withPluginVersion

```jsonnet
model.withPluginVersion(value)
```

PARAMETERS:

* **value** (`string`)

The version of the plugin that is used for this panel. This is used to find the plugin to display the panel and to migrate old panel configs.
#### fn model.withQueryCachingTTL

```jsonnet
model.withQueryCachingTTL(value)
```

PARAMETERS:

* **value** (`number`)

Overrides the data source configured time-to-live for a query cache item in milliseconds
#### fn model.withRepeat

```jsonnet
model.withRepeat(value)
```

PARAMETERS:

* **value** (`string`)

Name of template variable to repeat for.
#### fn model.withRepeatDirection

```jsonnet
model.withRepeatDirection(value="h")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"h"`
   - valid values: `"h"`, `"v"`

Direction to repeat in if 'repeat' is set.
`h` for horizontal, `v` for vertical.
#### fn model.withTargets

```jsonnet
model.withTargets(value)
```

PARAMETERS:

* **value** (`array`)

Depends on the panel plugin. See the plugin documentation for details.
#### fn model.withTargetsMixin

```jsonnet
model.withTargetsMixin(value)
```

PARAMETERS:

* **value** (`array`)

Depends on the panel plugin. See the plugin documentation for details.
#### fn model.withTimeCompare

```jsonnet
model.withTimeCompare(value)
```

PARAMETERS:

* **value** (`string`)

Compare the current time range with a previous period
For example "1d" to compare current period but shifted back 1 day
#### fn model.withTimeFrom

```jsonnet
model.withTimeFrom(value)
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
#### fn model.withTimeShift

```jsonnet
model.withTimeShift(value)
```

PARAMETERS:

* **value** (`string`)

Overrides the time range for individual panels by shifting its start and end relative to the time picker.
For example, you can shift the time range for the panel to be two hours earlier than the dashboard time picker setting `2h`.
Note: Panel time overrides have no effect when the dashboard’s time range is absolute.
See: https://grafana.com/docs/grafana/latest/panels-visualizations/query-transform-data/#query-options
#### fn model.withTitle

```jsonnet
model.withTitle(value)
```

PARAMETERS:

* **value** (`string`)

Panel title.
#### fn model.withTransformations

```jsonnet
model.withTransformations(value)
```

PARAMETERS:

* **value** (`array`)

List of transformations that are applied to the panel data before rendering.
When there are multiple transformations, Grafana applies them in the order they are listed.
Each transformation creates a result set that then passes on to the next transformation in the processing pipeline.
#### fn model.withTransformationsMixin

```jsonnet
model.withTransformationsMixin(value)
```

PARAMETERS:

* **value** (`array`)

List of transformations that are applied to the panel data before rendering.
When there are multiple transformations, Grafana applies them in the order they are listed.
Each transformation creates a result set that then passes on to the next transformation in the processing pipeline.
#### fn model.withTransparent

```jsonnet
model.withTransparent(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Whether to display the panel without a background.
#### fn model.withType

```jsonnet
model.withType(value)
```

PARAMETERS:

* **value** (`string`)

The panel plugin type id. This is used to find the plugin to display the panel.
#### obj model.datasource


##### fn model.datasource.withType

```jsonnet
model.datasource.withType(value)
```

PARAMETERS:

* **value** (`string`)

The plugin type-id
##### fn model.datasource.withUid

```jsonnet
model.datasource.withUid(value)
```

PARAMETERS:

* **value** (`string`)

Specific datasource instance
#### obj model.fieldConfig


##### fn model.fieldConfig.withDefaults

```jsonnet
model.fieldConfig.withDefaults(value)
```

PARAMETERS:

* **value** (`object`)

The data model used in Grafana, namely the data frame, is a columnar-oriented table structure that unifies both time series and table query results.
Each column within this structure is called a field. A field can represent a single time series or table column.
Field options allow you to change how the data is displayed in your visualizations.
##### fn model.fieldConfig.withDefaultsMixin

```jsonnet
model.fieldConfig.withDefaultsMixin(value)
```

PARAMETERS:

* **value** (`object`)

The data model used in Grafana, namely the data frame, is a columnar-oriented table structure that unifies both time series and table query results.
Each column within this structure is called a field. A field can represent a single time series or table column.
Field options allow you to change how the data is displayed in your visualizations.
##### fn model.fieldConfig.withOverrides

```jsonnet
model.fieldConfig.withOverrides(value)
```

PARAMETERS:

* **value** (`array`)

Overrides are the options applied to specific fields overriding the defaults.
##### fn model.fieldConfig.withOverridesMixin

```jsonnet
model.fieldConfig.withOverridesMixin(value)
```

PARAMETERS:

* **value** (`array`)

Overrides are the options applied to specific fields overriding the defaults.
##### obj model.fieldConfig.defaults


###### fn model.fieldConfig.defaults.withActions

```jsonnet
model.fieldConfig.defaults.withActions(value)
```

PARAMETERS:

* **value** (`array`)

Define interactive HTTP requests that can be triggered from data visualizations.
###### fn model.fieldConfig.defaults.withActionsMixin

```jsonnet
model.fieldConfig.defaults.withActionsMixin(value)
```

PARAMETERS:

* **value** (`array`)

Define interactive HTTP requests that can be triggered from data visualizations.
###### fn model.fieldConfig.defaults.withColor

```jsonnet
model.fieldConfig.defaults.withColor(value)
```

PARAMETERS:

* **value** (`object`)

Map a field to a color.
###### fn model.fieldConfig.defaults.withColorMixin

```jsonnet
model.fieldConfig.defaults.withColorMixin(value)
```

PARAMETERS:

* **value** (`object`)

Map a field to a color.
###### fn model.fieldConfig.defaults.withCustom

```jsonnet
model.fieldConfig.defaults.withCustom(value)
```

PARAMETERS:

* **value** (`object`)

custom is specified by the FieldConfig field
in panel plugin schemas.
###### fn model.fieldConfig.defaults.withCustomMixin

```jsonnet
model.fieldConfig.defaults.withCustomMixin(value)
```

PARAMETERS:

* **value** (`object`)

custom is specified by the FieldConfig field
in panel plugin schemas.
###### fn model.fieldConfig.defaults.withDecimals

```jsonnet
model.fieldConfig.defaults.withDecimals(value)
```

PARAMETERS:

* **value** (`number`)

Specify the number of decimals Grafana includes in the rendered value.
If you leave this field blank, Grafana automatically truncates the number of decimals based on the value.
For example 1.1234 will display as 1.12 and 100.456 will display as 100.
To display all decimals, set the unit to `String`.
###### fn model.fieldConfig.defaults.withDescription

```jsonnet
model.fieldConfig.defaults.withDescription(value)
```

PARAMETERS:

* **value** (`string`)

Human readable field metadata
###### fn model.fieldConfig.defaults.withDisplayName

```jsonnet
model.fieldConfig.defaults.withDisplayName(value)
```

PARAMETERS:

* **value** (`string`)

The display value for this field.  This supports template variables blank is auto
###### fn model.fieldConfig.defaults.withDisplayNameFromDS

```jsonnet
model.fieldConfig.defaults.withDisplayNameFromDS(value)
```

PARAMETERS:

* **value** (`string`)

This can be used by data sources that return and explicit naming structure for values and labels
When this property is configured, this value is used rather than the default naming strategy.
###### fn model.fieldConfig.defaults.withFieldMinMax

```jsonnet
model.fieldConfig.defaults.withFieldMinMax(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


###### fn model.fieldConfig.defaults.withFilterable

```jsonnet
model.fieldConfig.defaults.withFilterable(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

True if data source field supports ad-hoc filters
###### fn model.fieldConfig.defaults.withLinks

```jsonnet
model.fieldConfig.defaults.withLinks(value)
```

PARAMETERS:

* **value** (`array`)

The behavior when clicking on a result
###### fn model.fieldConfig.defaults.withLinksMixin

```jsonnet
model.fieldConfig.defaults.withLinksMixin(value)
```

PARAMETERS:

* **value** (`array`)

The behavior when clicking on a result
###### fn model.fieldConfig.defaults.withMappings

```jsonnet
model.fieldConfig.defaults.withMappings(value)
```

PARAMETERS:

* **value** (`array`)

Convert input values into a display string
###### fn model.fieldConfig.defaults.withMappingsMixin

```jsonnet
model.fieldConfig.defaults.withMappingsMixin(value)
```

PARAMETERS:

* **value** (`array`)

Convert input values into a display string
###### fn model.fieldConfig.defaults.withMax

```jsonnet
model.fieldConfig.defaults.withMax(value)
```

PARAMETERS:

* **value** (`number`)

The maximum value used in percentage threshold calculations. Leave blank for auto calculation based on all series and fields.
###### fn model.fieldConfig.defaults.withMin

```jsonnet
model.fieldConfig.defaults.withMin(value)
```

PARAMETERS:

* **value** (`number`)

The minimum value used in percentage threshold calculations. Leave blank for auto calculation based on all series and fields.
###### fn model.fieldConfig.defaults.withNoValue

```jsonnet
model.fieldConfig.defaults.withNoValue(value)
```

PARAMETERS:

* **value** (`string`)

Alternative to empty string
###### fn model.fieldConfig.defaults.withPath

```jsonnet
model.fieldConfig.defaults.withPath(value)
```

PARAMETERS:

* **value** (`string`)

An explicit path to the field in the datasource.  When the frame meta includes a path,
This will default to `${frame.meta.path}/${field.name}

When defined, this value can be used as an identifier within the datasource scope, and
may be used to update the results
###### fn model.fieldConfig.defaults.withThresholds

```jsonnet
model.fieldConfig.defaults.withThresholds(value)
```

PARAMETERS:

* **value** (`object`)

Thresholds configuration for the panel
###### fn model.fieldConfig.defaults.withThresholdsMixin

```jsonnet
model.fieldConfig.defaults.withThresholdsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Thresholds configuration for the panel
###### fn model.fieldConfig.defaults.withUnit

```jsonnet
model.fieldConfig.defaults.withUnit(value)
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
###### fn model.fieldConfig.defaults.withWriteable

```jsonnet
model.fieldConfig.defaults.withWriteable(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

True if data source can write a value to the path. Auth/authz are supported separately
###### obj model.fieldConfig.defaults.color


####### fn model.fieldConfig.defaults.color.withFixedColor

```jsonnet
model.fieldConfig.defaults.color.withFixedColor(value)
```

PARAMETERS:

* **value** (`string`)

The fixed color value for fixed or shades color modes.
####### fn model.fieldConfig.defaults.color.withMode

```jsonnet
model.fieldConfig.defaults.color.withMode(value)
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
####### fn model.fieldConfig.defaults.color.withSeriesBy

```jsonnet
model.fieldConfig.defaults.color.withSeriesBy(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"min"`, `"max"`, `"last"`

Defines how to assign a series color from "by value" color schemes. For example for an aggregated data points like a timeseries, the color can be assigned by the min, max or last value.
###### obj model.fieldConfig.defaults.mappings


####### obj model.fieldConfig.defaults.mappings.RangeMap


######## fn model.fieldConfig.defaults.mappings.RangeMap.withOptions

```jsonnet
model.fieldConfig.defaults.mappings.RangeMap.withOptions(value)
```

PARAMETERS:

* **value** (`object`)

Range to match against and the result to apply when the value is within the range
######## fn model.fieldConfig.defaults.mappings.RangeMap.withOptionsMixin

```jsonnet
model.fieldConfig.defaults.mappings.RangeMap.withOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Range to match against and the result to apply when the value is within the range
######## fn model.fieldConfig.defaults.mappings.RangeMap.withType

```jsonnet
model.fieldConfig.defaults.mappings.RangeMap.withType(value="range")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"range"`


######## fn model.fieldConfig.defaults.mappings.RangeMap.withTypeMixin

```jsonnet
model.fieldConfig.defaults.mappings.RangeMap.withTypeMixin(value="range")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"range"`


######## obj model.fieldConfig.defaults.mappings.RangeMap.options


######### fn model.fieldConfig.defaults.mappings.RangeMap.options.withFrom

```jsonnet
model.fieldConfig.defaults.mappings.RangeMap.options.withFrom(value)
```

PARAMETERS:

* **value** (`number`)

Min value of the range. It can be null which means -Infinity
######### fn model.fieldConfig.defaults.mappings.RangeMap.options.withResult

```jsonnet
model.fieldConfig.defaults.mappings.RangeMap.options.withResult(value)
```

PARAMETERS:

* **value** (`object`)

Result used as replacement with text and color when the value matches
######### fn model.fieldConfig.defaults.mappings.RangeMap.options.withResultMixin

```jsonnet
model.fieldConfig.defaults.mappings.RangeMap.options.withResultMixin(value)
```

PARAMETERS:

* **value** (`object`)

Result used as replacement with text and color when the value matches
######### fn model.fieldConfig.defaults.mappings.RangeMap.options.withTo

```jsonnet
model.fieldConfig.defaults.mappings.RangeMap.options.withTo(value)
```

PARAMETERS:

* **value** (`number`)

Max value of the range. It can be null which means +Infinity
######### obj model.fieldConfig.defaults.mappings.RangeMap.options.result


########## fn model.fieldConfig.defaults.mappings.RangeMap.options.result.withColor

```jsonnet
model.fieldConfig.defaults.mappings.RangeMap.options.result.withColor(value)
```

PARAMETERS:

* **value** (`string`)

Text to use when the value matches
########## fn model.fieldConfig.defaults.mappings.RangeMap.options.result.withIcon

```jsonnet
model.fieldConfig.defaults.mappings.RangeMap.options.result.withIcon(value)
```

PARAMETERS:

* **value** (`string`)

Icon to display when the value matches. Only specific visualizations.
########## fn model.fieldConfig.defaults.mappings.RangeMap.options.result.withIndex

```jsonnet
model.fieldConfig.defaults.mappings.RangeMap.options.result.withIndex(value)
```

PARAMETERS:

* **value** (`integer`)

Position in the mapping array. Only used internally.
########## fn model.fieldConfig.defaults.mappings.RangeMap.options.result.withText

```jsonnet
model.fieldConfig.defaults.mappings.RangeMap.options.result.withText(value)
```

PARAMETERS:

* **value** (`string`)

Text to display when the value matches
######## obj model.fieldConfig.defaults.mappings.RangeMap.type


######### fn model.fieldConfig.defaults.mappings.RangeMap.type.withMappingType

```jsonnet
model.fieldConfig.defaults.mappings.RangeMap.type.withMappingType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"value"`, `"range"`, `"regex"`, `"special"`

Supported value mapping types
`value`: Maps text values to a color or different display text and color. For example, you can configure a value mapping so that all instances of the value 10 appear as Perfection! rather than the number.
`range`: Maps numerical ranges to a display text and color. For example, if a value is within a certain range, you can configure a range value mapping to display Low or High rather than the number.
`regex`: Maps regular expressions to replacement text and a color. For example, if a value is www.example.com, you can configure a regex value mapping so that Grafana displays www and truncates the domain.
`special`: Maps special values like Null, NaN (not a number), and boolean values like true and false to a display text and color. See SpecialValueMatch to see the list of special values. For example, you can configure a special value mapping so that null values appear as N/A.
####### obj model.fieldConfig.defaults.mappings.RegexMap


######## fn model.fieldConfig.defaults.mappings.RegexMap.withOptions

```jsonnet
model.fieldConfig.defaults.mappings.RegexMap.withOptions(value)
```

PARAMETERS:

* **value** (`object`)

Regular expression to match against and the result to apply when the value matches the regex
######## fn model.fieldConfig.defaults.mappings.RegexMap.withOptionsMixin

```jsonnet
model.fieldConfig.defaults.mappings.RegexMap.withOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Regular expression to match against and the result to apply when the value matches the regex
######## fn model.fieldConfig.defaults.mappings.RegexMap.withType

```jsonnet
model.fieldConfig.defaults.mappings.RegexMap.withType(value="regex")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"regex"`


######## fn model.fieldConfig.defaults.mappings.RegexMap.withTypeMixin

```jsonnet
model.fieldConfig.defaults.mappings.RegexMap.withTypeMixin(value="regex")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"regex"`


######## obj model.fieldConfig.defaults.mappings.RegexMap.options


######### fn model.fieldConfig.defaults.mappings.RegexMap.options.withPattern

```jsonnet
model.fieldConfig.defaults.mappings.RegexMap.options.withPattern(value)
```

PARAMETERS:

* **value** (`string`)

Regular expression to match against
######### fn model.fieldConfig.defaults.mappings.RegexMap.options.withResult

```jsonnet
model.fieldConfig.defaults.mappings.RegexMap.options.withResult(value)
```

PARAMETERS:

* **value** (`object`)

Result used as replacement with text and color when the value matches
######### fn model.fieldConfig.defaults.mappings.RegexMap.options.withResultMixin

```jsonnet
model.fieldConfig.defaults.mappings.RegexMap.options.withResultMixin(value)
```

PARAMETERS:

* **value** (`object`)

Result used as replacement with text and color when the value matches
######### obj model.fieldConfig.defaults.mappings.RegexMap.options.result


########## fn model.fieldConfig.defaults.mappings.RegexMap.options.result.withColor

```jsonnet
model.fieldConfig.defaults.mappings.RegexMap.options.result.withColor(value)
```

PARAMETERS:

* **value** (`string`)

Text to use when the value matches
########## fn model.fieldConfig.defaults.mappings.RegexMap.options.result.withIcon

```jsonnet
model.fieldConfig.defaults.mappings.RegexMap.options.result.withIcon(value)
```

PARAMETERS:

* **value** (`string`)

Icon to display when the value matches. Only specific visualizations.
########## fn model.fieldConfig.defaults.mappings.RegexMap.options.result.withIndex

```jsonnet
model.fieldConfig.defaults.mappings.RegexMap.options.result.withIndex(value)
```

PARAMETERS:

* **value** (`integer`)

Position in the mapping array. Only used internally.
########## fn model.fieldConfig.defaults.mappings.RegexMap.options.result.withText

```jsonnet
model.fieldConfig.defaults.mappings.RegexMap.options.result.withText(value)
```

PARAMETERS:

* **value** (`string`)

Text to display when the value matches
######## obj model.fieldConfig.defaults.mappings.RegexMap.type


######### fn model.fieldConfig.defaults.mappings.RegexMap.type.withMappingType

```jsonnet
model.fieldConfig.defaults.mappings.RegexMap.type.withMappingType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"value"`, `"range"`, `"regex"`, `"special"`

Supported value mapping types
`value`: Maps text values to a color or different display text and color. For example, you can configure a value mapping so that all instances of the value 10 appear as Perfection! rather than the number.
`range`: Maps numerical ranges to a display text and color. For example, if a value is within a certain range, you can configure a range value mapping to display Low or High rather than the number.
`regex`: Maps regular expressions to replacement text and a color. For example, if a value is www.example.com, you can configure a regex value mapping so that Grafana displays www and truncates the domain.
`special`: Maps special values like Null, NaN (not a number), and boolean values like true and false to a display text and color. See SpecialValueMatch to see the list of special values. For example, you can configure a special value mapping so that null values appear as N/A.
####### obj model.fieldConfig.defaults.mappings.SpecialValueMap


######## fn model.fieldConfig.defaults.mappings.SpecialValueMap.withOptions

```jsonnet
model.fieldConfig.defaults.mappings.SpecialValueMap.withOptions(value)
```

PARAMETERS:

* **value** (`object`)


######## fn model.fieldConfig.defaults.mappings.SpecialValueMap.withOptionsMixin

```jsonnet
model.fieldConfig.defaults.mappings.SpecialValueMap.withOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)


######## fn model.fieldConfig.defaults.mappings.SpecialValueMap.withType

```jsonnet
model.fieldConfig.defaults.mappings.SpecialValueMap.withType(value="special")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"special"`


######## fn model.fieldConfig.defaults.mappings.SpecialValueMap.withTypeMixin

```jsonnet
model.fieldConfig.defaults.mappings.SpecialValueMap.withTypeMixin(value="special")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"special"`


######## obj model.fieldConfig.defaults.mappings.SpecialValueMap.options


######### fn model.fieldConfig.defaults.mappings.SpecialValueMap.options.withMatch

```jsonnet
model.fieldConfig.defaults.mappings.SpecialValueMap.options.withMatch(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"true"`, `"false"`, `"null"`, `"nan"`, `"null+nan"`, `"empty"`

Special value types supported by the `SpecialValueMap`
######### fn model.fieldConfig.defaults.mappings.SpecialValueMap.options.withResult

```jsonnet
model.fieldConfig.defaults.mappings.SpecialValueMap.options.withResult(value)
```

PARAMETERS:

* **value** (`object`)

Result used as replacement with text and color when the value matches
######### fn model.fieldConfig.defaults.mappings.SpecialValueMap.options.withResultMixin

```jsonnet
model.fieldConfig.defaults.mappings.SpecialValueMap.options.withResultMixin(value)
```

PARAMETERS:

* **value** (`object`)

Result used as replacement with text and color when the value matches
######### obj model.fieldConfig.defaults.mappings.SpecialValueMap.options.result


########## fn model.fieldConfig.defaults.mappings.SpecialValueMap.options.result.withColor

```jsonnet
model.fieldConfig.defaults.mappings.SpecialValueMap.options.result.withColor(value)
```

PARAMETERS:

* **value** (`string`)

Text to use when the value matches
########## fn model.fieldConfig.defaults.mappings.SpecialValueMap.options.result.withIcon

```jsonnet
model.fieldConfig.defaults.mappings.SpecialValueMap.options.result.withIcon(value)
```

PARAMETERS:

* **value** (`string`)

Icon to display when the value matches. Only specific visualizations.
########## fn model.fieldConfig.defaults.mappings.SpecialValueMap.options.result.withIndex

```jsonnet
model.fieldConfig.defaults.mappings.SpecialValueMap.options.result.withIndex(value)
```

PARAMETERS:

* **value** (`integer`)

Position in the mapping array. Only used internally.
########## fn model.fieldConfig.defaults.mappings.SpecialValueMap.options.result.withText

```jsonnet
model.fieldConfig.defaults.mappings.SpecialValueMap.options.result.withText(value)
```

PARAMETERS:

* **value** (`string`)

Text to display when the value matches
######## obj model.fieldConfig.defaults.mappings.SpecialValueMap.type


######### fn model.fieldConfig.defaults.mappings.SpecialValueMap.type.withMappingType

```jsonnet
model.fieldConfig.defaults.mappings.SpecialValueMap.type.withMappingType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"value"`, `"range"`, `"regex"`, `"special"`

Supported value mapping types
`value`: Maps text values to a color or different display text and color. For example, you can configure a value mapping so that all instances of the value 10 appear as Perfection! rather than the number.
`range`: Maps numerical ranges to a display text and color. For example, if a value is within a certain range, you can configure a range value mapping to display Low or High rather than the number.
`regex`: Maps regular expressions to replacement text and a color. For example, if a value is www.example.com, you can configure a regex value mapping so that Grafana displays www and truncates the domain.
`special`: Maps special values like Null, NaN (not a number), and boolean values like true and false to a display text and color. See SpecialValueMatch to see the list of special values. For example, you can configure a special value mapping so that null values appear as N/A.
####### obj model.fieldConfig.defaults.mappings.ValueMap


######## fn model.fieldConfig.defaults.mappings.ValueMap.withOptions

```jsonnet
model.fieldConfig.defaults.mappings.ValueMap.withOptions(value)
```

PARAMETERS:

* **value** (`object`)

Map with <value_to_match>: ValueMappingResult. For example: { "10": { text: "Perfection!", color: "green" } }
######## fn model.fieldConfig.defaults.mappings.ValueMap.withOptionsMixin

```jsonnet
model.fieldConfig.defaults.mappings.ValueMap.withOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Map with <value_to_match>: ValueMappingResult. For example: { "10": { text: "Perfection!", color: "green" } }
######## fn model.fieldConfig.defaults.mappings.ValueMap.withType

```jsonnet
model.fieldConfig.defaults.mappings.ValueMap.withType(value="value")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"value"`


######## fn model.fieldConfig.defaults.mappings.ValueMap.withTypeMixin

```jsonnet
model.fieldConfig.defaults.mappings.ValueMap.withTypeMixin(value="value")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"value"`


######## obj model.fieldConfig.defaults.mappings.ValueMap.type


######### fn model.fieldConfig.defaults.mappings.ValueMap.type.withMappingType

```jsonnet
model.fieldConfig.defaults.mappings.ValueMap.type.withMappingType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"value"`, `"range"`, `"regex"`, `"special"`

Supported value mapping types
`value`: Maps text values to a color or different display text and color. For example, you can configure a value mapping so that all instances of the value 10 appear as Perfection! rather than the number.
`range`: Maps numerical ranges to a display text and color. For example, if a value is within a certain range, you can configure a range value mapping to display Low or High rather than the number.
`regex`: Maps regular expressions to replacement text and a color. For example, if a value is www.example.com, you can configure a regex value mapping so that Grafana displays www and truncates the domain.
`special`: Maps special values like Null, NaN (not a number), and boolean values like true and false to a display text and color. See SpecialValueMatch to see the list of special values. For example, you can configure a special value mapping so that null values appear as N/A.
###### obj model.fieldConfig.defaults.thresholds


####### fn model.fieldConfig.defaults.thresholds.withMode

```jsonnet
model.fieldConfig.defaults.thresholds.withMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"absolute"`, `"percentage"`

Thresholds can either be `absolute` (specific number) or `percentage` (relative to min or max, it will be values between 0 and 1).
####### fn model.fieldConfig.defaults.thresholds.withSteps

```jsonnet
model.fieldConfig.defaults.thresholds.withSteps(value)
```

PARAMETERS:

* **value** (`array`)

Must be sorted by 'value', first value is always -Infinity
####### fn model.fieldConfig.defaults.thresholds.withStepsMixin

```jsonnet
model.fieldConfig.defaults.thresholds.withStepsMixin(value)
```

PARAMETERS:

* **value** (`array`)

Must be sorted by 'value', first value is always -Infinity
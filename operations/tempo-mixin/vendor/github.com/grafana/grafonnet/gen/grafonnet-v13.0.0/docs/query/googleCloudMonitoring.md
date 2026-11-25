# googleCloudMonitoring

grafonnet.query.googleCloudMonitoring

## Index

* [`fn withAliasBy(value)`](#fn-withaliasby)
* [`fn withDatasource(value)`](#fn-withdatasource)
* [`fn withHide(value=true)`](#fn-withhide)
* [`fn withIntervalMs(value)`](#fn-withintervalms)
* [`fn withPromQLQuery(value)`](#fn-withpromqlquery)
* [`fn withPromQLQueryMixin(value)`](#fn-withpromqlquerymixin)
* [`fn withQueryType(value)`](#fn-withquerytype)
* [`fn withRefId(value)`](#fn-withrefid)
* [`fn withSloQuery(value)`](#fn-withsloquery)
* [`fn withSloQueryMixin(value)`](#fn-withsloquerymixin)
* [`fn withTimeSeriesList(value)`](#fn-withtimeserieslist)
* [`fn withTimeSeriesListMixin(value)`](#fn-withtimeserieslistmixin)
* [`fn withTimeSeriesQuery(value)`](#fn-withtimeseriesquery)
* [`fn withTimeSeriesQueryMixin(value)`](#fn-withtimeseriesquerymixin)
* [`obj datasource`](#obj-datasource)
  * [`fn withType(value)`](#fn-datasourcewithtype)
  * [`fn withUid(value)`](#fn-datasourcewithuid)
* [`obj promQLQuery`](#obj-promqlquery)
  * [`fn withExpr(value)`](#fn-promqlquerywithexpr)
  * [`fn withProjectName(value)`](#fn-promqlquerywithprojectname)
  * [`fn withStep(value)`](#fn-promqlquerywithstep)
* [`obj sloQuery`](#obj-sloquery)
  * [`fn withAlignmentPeriod(value)`](#fn-sloquerywithalignmentperiod)
  * [`fn withGoal(value)`](#fn-sloquerywithgoal)
  * [`fn withLookbackPeriod(value)`](#fn-sloquerywithlookbackperiod)
  * [`fn withPerSeriesAligner(value)`](#fn-sloquerywithperseriesaligner)
  * [`fn withProjectName(value)`](#fn-sloquerywithprojectname)
  * [`fn withSelectorName(value)`](#fn-sloquerywithselectorname)
  * [`fn withServiceId(value)`](#fn-sloquerywithserviceid)
  * [`fn withServiceName(value)`](#fn-sloquerywithservicename)
  * [`fn withSloId(value)`](#fn-sloquerywithsloid)
  * [`fn withSloName(value)`](#fn-sloquerywithsloname)
* [`obj timeSeriesList`](#obj-timeserieslist)
  * [`fn withAlignmentPeriod(value)`](#fn-timeserieslistwithalignmentperiod)
  * [`fn withCrossSeriesReducer(value)`](#fn-timeserieslistwithcrossseriesreducer)
  * [`fn withFilters(value)`](#fn-timeserieslistwithfilters)
  * [`fn withFiltersMixin(value)`](#fn-timeserieslistwithfiltersmixin)
  * [`fn withGroupBys(value)`](#fn-timeserieslistwithgroupbys)
  * [`fn withGroupBysMixin(value)`](#fn-timeserieslistwithgroupbysmixin)
  * [`fn withPerSeriesAligner(value)`](#fn-timeserieslistwithperseriesaligner)
  * [`fn withPreprocessor(value)`](#fn-timeserieslistwithpreprocessor)
  * [`fn withProjectName(value)`](#fn-timeserieslistwithprojectname)
  * [`fn withSecondaryAlignmentPeriod(value)`](#fn-timeserieslistwithsecondaryalignmentperiod)
  * [`fn withSecondaryCrossSeriesReducer(value)`](#fn-timeserieslistwithsecondarycrossseriesreducer)
  * [`fn withSecondaryGroupBys(value)`](#fn-timeserieslistwithsecondarygroupbys)
  * [`fn withSecondaryGroupBysMixin(value)`](#fn-timeserieslistwithsecondarygroupbysmixin)
  * [`fn withSecondaryPerSeriesAligner(value)`](#fn-timeserieslistwithsecondaryperseriesaligner)
  * [`fn withText(value)`](#fn-timeserieslistwithtext)
  * [`fn withTitle(value)`](#fn-timeserieslistwithtitle)
  * [`fn withView(value)`](#fn-timeserieslistwithview)
* [`obj timeSeriesQuery`](#obj-timeseriesquery)
  * [`fn withGraphPeriod(value="disabled")`](#fn-timeseriesquerywithgraphperiod)
  * [`fn withProjectName(value)`](#fn-timeseriesquerywithprojectname)
  * [`fn withQuery(value)`](#fn-timeseriesquerywithquery)

## Fields

### fn withAliasBy

```jsonnet
withAliasBy(value)
```

PARAMETERS:

* **value** (`string`)

Aliases can be set to modify the legend labels. e.g. {{metric.label.xxx}}. See docs for more detail.
### fn withDatasource

```jsonnet
withDatasource(value)
```

PARAMETERS:

* **value** (`string`)

Set the datasource for this query.
### fn withHide

```jsonnet
withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

If hide is set to true, Grafana will filter out the response(s) associated with this query before returning it to the panel.
### fn withIntervalMs

```jsonnet
withIntervalMs(value)
```

PARAMETERS:

* **value** (`number`)

Time interval in milliseconds.
### fn withPromQLQuery

```jsonnet
withPromQLQuery(value)
```

PARAMETERS:

* **value** (`object`)

PromQL sub-query properties.
### fn withPromQLQueryMixin

```jsonnet
withPromQLQueryMixin(value)
```

PARAMETERS:

* **value** (`object`)

PromQL sub-query properties.
### fn withQueryType

```jsonnet
withQueryType(value)
```

PARAMETERS:

* **value** (`string`)

Specify the query flavor
TODO make this required and give it a default
### fn withRefId

```jsonnet
withRefId(value)
```

PARAMETERS:

* **value** (`string`)

A unique identifier for the query within the list of targets.
In server side expressions, the refId is used as a variable name to identify results.
By default, the UI will assign A->Z; however setting meaningful names may be useful.
### fn withSloQuery

```jsonnet
withSloQuery(value)
```

PARAMETERS:

* **value** (`object`)

SLO sub-query properties.
### fn withSloQueryMixin

```jsonnet
withSloQueryMixin(value)
```

PARAMETERS:

* **value** (`object`)

SLO sub-query properties.
### fn withTimeSeriesList

```jsonnet
withTimeSeriesList(value)
```

PARAMETERS:

* **value** (`object`)

Time Series List sub-query properties.
### fn withTimeSeriesListMixin

```jsonnet
withTimeSeriesListMixin(value)
```

PARAMETERS:

* **value** (`object`)

Time Series List sub-query properties.
### fn withTimeSeriesQuery

```jsonnet
withTimeSeriesQuery(value)
```

PARAMETERS:

* **value** (`object`)

Time Series sub-query properties.
### fn withTimeSeriesQueryMixin

```jsonnet
withTimeSeriesQueryMixin(value)
```

PARAMETERS:

* **value** (`object`)

Time Series sub-query properties.
### obj datasource


#### fn datasource.withType

```jsonnet
datasource.withType(value)
```

PARAMETERS:

* **value** (`string`)

The plugin type-id
#### fn datasource.withUid

```jsonnet
datasource.withUid(value)
```

PARAMETERS:

* **value** (`string`)

Specific datasource instance
### obj promQLQuery


#### fn promQLQuery.withExpr

```jsonnet
promQLQuery.withExpr(value)
```

PARAMETERS:

* **value** (`string`)

PromQL expression/query to be executed.
#### fn promQLQuery.withProjectName

```jsonnet
promQLQuery.withProjectName(value)
```

PARAMETERS:

* **value** (`string`)

GCP project to execute the query against.
#### fn promQLQuery.withStep

```jsonnet
promQLQuery.withStep(value)
```

PARAMETERS:

* **value** (`string`)

PromQL min step
### obj sloQuery


#### fn sloQuery.withAlignmentPeriod

```jsonnet
sloQuery.withAlignmentPeriod(value)
```

PARAMETERS:

* **value** (`string`)

Alignment period to use when regularizing data. Defaults to cloud-monitoring-auto.
#### fn sloQuery.withGoal

```jsonnet
sloQuery.withGoal(value)
```

PARAMETERS:

* **value** (`number`)

SLO goal value.
#### fn sloQuery.withLookbackPeriod

```jsonnet
sloQuery.withLookbackPeriod(value)
```

PARAMETERS:

* **value** (`string`)

Specific lookback period for the SLO.
#### fn sloQuery.withPerSeriesAligner

```jsonnet
sloQuery.withPerSeriesAligner(value)
```

PARAMETERS:

* **value** (`string`)

Alignment function to be used. Defaults to ALIGN_MEAN.
#### fn sloQuery.withProjectName

```jsonnet
sloQuery.withProjectName(value)
```

PARAMETERS:

* **value** (`string`)

GCP project to execute the query against.
#### fn sloQuery.withSelectorName

```jsonnet
sloQuery.withSelectorName(value)
```

PARAMETERS:

* **value** (`string`)

SLO selector.
#### fn sloQuery.withServiceId

```jsonnet
sloQuery.withServiceId(value)
```

PARAMETERS:

* **value** (`string`)

ID for the service the SLO is in.
#### fn sloQuery.withServiceName

```jsonnet
sloQuery.withServiceName(value)
```

PARAMETERS:

* **value** (`string`)

Name for the service the SLO is in.
#### fn sloQuery.withSloId

```jsonnet
sloQuery.withSloId(value)
```

PARAMETERS:

* **value** (`string`)

ID for the SLO.
#### fn sloQuery.withSloName

```jsonnet
sloQuery.withSloName(value)
```

PARAMETERS:

* **value** (`string`)

Name of the SLO.
### obj timeSeriesList


#### fn timeSeriesList.withAlignmentPeriod

```jsonnet
timeSeriesList.withAlignmentPeriod(value)
```

PARAMETERS:

* **value** (`string`)

Alignment period to use when regularizing data. Defaults to cloud-monitoring-auto.
#### fn timeSeriesList.withCrossSeriesReducer

```jsonnet
timeSeriesList.withCrossSeriesReducer(value)
```

PARAMETERS:

* **value** (`string`)

Reducer applied across a set of time-series values. Defaults to REDUCE_NONE.
#### fn timeSeriesList.withFilters

```jsonnet
timeSeriesList.withFilters(value)
```

PARAMETERS:

* **value** (`array`)

Array of filters to query data by. Labels that can be filtered on are defined by the metric.
#### fn timeSeriesList.withFiltersMixin

```jsonnet
timeSeriesList.withFiltersMixin(value)
```

PARAMETERS:

* **value** (`array`)

Array of filters to query data by. Labels that can be filtered on are defined by the metric.
#### fn timeSeriesList.withGroupBys

```jsonnet
timeSeriesList.withGroupBys(value)
```

PARAMETERS:

* **value** (`array`)

Array of labels to group data by.
#### fn timeSeriesList.withGroupBysMixin

```jsonnet
timeSeriesList.withGroupBysMixin(value)
```

PARAMETERS:

* **value** (`array`)

Array of labels to group data by.
#### fn timeSeriesList.withPerSeriesAligner

```jsonnet
timeSeriesList.withPerSeriesAligner(value)
```

PARAMETERS:

* **value** (`string`)

Alignment function to be used. Defaults to ALIGN_MEAN.
#### fn timeSeriesList.withPreprocessor

```jsonnet
timeSeriesList.withPreprocessor(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"none"`, `"rate"`, `"delta"`

Types of pre-processor available. Defined by the metric.
#### fn timeSeriesList.withProjectName

```jsonnet
timeSeriesList.withProjectName(value)
```

PARAMETERS:

* **value** (`string`)

GCP project to execute the query against.
#### fn timeSeriesList.withSecondaryAlignmentPeriod

```jsonnet
timeSeriesList.withSecondaryAlignmentPeriod(value)
```

PARAMETERS:

* **value** (`string`)

Only present if a preprocessor is selected. Alignment period to use when regularizing data. Defaults to cloud-monitoring-auto.
#### fn timeSeriesList.withSecondaryCrossSeriesReducer

```jsonnet
timeSeriesList.withSecondaryCrossSeriesReducer(value)
```

PARAMETERS:

* **value** (`string`)

Only present if a preprocessor is selected. Reducer applied across a set of time-series values. Defaults to REDUCE_NONE.
#### fn timeSeriesList.withSecondaryGroupBys

```jsonnet
timeSeriesList.withSecondaryGroupBys(value)
```

PARAMETERS:

* **value** (`array`)

Only present if a preprocessor is selected. Array of labels to group data by.
#### fn timeSeriesList.withSecondaryGroupBysMixin

```jsonnet
timeSeriesList.withSecondaryGroupBysMixin(value)
```

PARAMETERS:

* **value** (`array`)

Only present if a preprocessor is selected. Array of labels to group data by.
#### fn timeSeriesList.withSecondaryPerSeriesAligner

```jsonnet
timeSeriesList.withSecondaryPerSeriesAligner(value)
```

PARAMETERS:

* **value** (`string`)

Only present if a preprocessor is selected. Alignment function to be used. Defaults to ALIGN_MEAN.
#### fn timeSeriesList.withText

```jsonnet
timeSeriesList.withText(value)
```

PARAMETERS:

* **value** (`string`)

Annotation text.
#### fn timeSeriesList.withTitle

```jsonnet
timeSeriesList.withTitle(value)
```

PARAMETERS:

* **value** (`string`)

Annotation title.
#### fn timeSeriesList.withView

```jsonnet
timeSeriesList.withView(value)
```

PARAMETERS:

* **value** (`string`)

Data view, defaults to FULL.
### obj timeSeriesQuery


#### fn timeSeriesQuery.withGraphPeriod

```jsonnet
timeSeriesQuery.withGraphPeriod(value="disabled")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"disabled"`

To disable the graphPeriod, it should explictly be set to 'disabled'.
#### fn timeSeriesQuery.withProjectName

```jsonnet
timeSeriesQuery.withProjectName(value)
```

PARAMETERS:

* **value** (`string`)

GCP project to execute the query against.
#### fn timeSeriesQuery.withQuery

```jsonnet
timeSeriesQuery.withQuery(value)
```

PARAMETERS:

* **value** (`string`)

MQL query to be executed.
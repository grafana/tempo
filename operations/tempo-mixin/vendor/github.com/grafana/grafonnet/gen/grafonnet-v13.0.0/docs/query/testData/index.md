# testData

grafonnet.query.testData

## Subpackages

* [csvWave](csvWave.md)

## Index

* [`fn withAlias(value)`](#fn-withalias)
* [`fn withChannel(value)`](#fn-withchannel)
* [`fn withCsvContent(value)`](#fn-withcsvcontent)
* [`fn withCsvFileName(value)`](#fn-withcsvfilename)
* [`fn withCsvWave(value)`](#fn-withcsvwave)
* [`fn withCsvWaveMixin(value)`](#fn-withcsvwavemixin)
* [`fn withDatasource(value)`](#fn-withdatasource)
* [`fn withDropPercent(value)`](#fn-withdroppercent)
* [`fn withErrorSource(value)`](#fn-witherrorsource)
* [`fn withErrorType(value)`](#fn-witherrortype)
* [`fn withFlamegraphDiff(value=true)`](#fn-withflamegraphdiff)
* [`fn withHide(value=true)`](#fn-withhide)
* [`fn withIntervalMs(value)`](#fn-withintervalms)
* [`fn withLabels(value)`](#fn-withlabels)
* [`fn withLevelColumn(value=true)`](#fn-withlevelcolumn)
* [`fn withLines(value)`](#fn-withlines)
* [`fn withMax(value)`](#fn-withmax)
* [`fn withMaxDataPoints(value)`](#fn-withmaxdatapoints)
* [`fn withMin(value)`](#fn-withmin)
* [`fn withNodes(value)`](#fn-withnodes)
* [`fn withNodesMixin(value)`](#fn-withnodesmixin)
* [`fn withNoise(value)`](#fn-withnoise)
* [`fn withPoints(value)`](#fn-withpoints)
* [`fn withPointsMixin(value)`](#fn-withpointsmixin)
* [`fn withPulseWave(value)`](#fn-withpulsewave)
* [`fn withPulseWaveMixin(value)`](#fn-withpulsewavemixin)
* [`fn withQueryType(value)`](#fn-withquerytype)
* [`fn withRawFrameContent(value)`](#fn-withrawframecontent)
* [`fn withRefId(value)`](#fn-withrefid)
* [`fn withResultAssertions(value)`](#fn-withresultassertions)
* [`fn withResultAssertionsMixin(value)`](#fn-withresultassertionsmixin)
* [`fn withScenarioId(value)`](#fn-withscenarioid)
* [`fn withSeriesCount(value)`](#fn-withseriescount)
* [`fn withSim(value)`](#fn-withsim)
* [`fn withSimMixin(value)`](#fn-withsimmixin)
* [`fn withSpanCount(value)`](#fn-withspancount)
* [`fn withSpread(value)`](#fn-withspread)
* [`fn withStartValue(value)`](#fn-withstartvalue)
* [`fn withStream(value)`](#fn-withstream)
* [`fn withStreamMixin(value)`](#fn-withstreammixin)
* [`fn withStringInput(value)`](#fn-withstringinput)
* [`fn withTimeRange(value)`](#fn-withtimerange)
* [`fn withTimeRangeMixin(value)`](#fn-withtimerangemixin)
* [`fn withUsa(value)`](#fn-withusa)
* [`fn withUsaMixin(value)`](#fn-withusamixin)
* [`fn withWithNil(value=true)`](#fn-withwithnil)
* [`obj datasource`](#obj-datasource)
  * [`fn withType(value)`](#fn-datasourcewithtype)
  * [`fn withUid(value)`](#fn-datasourcewithuid)
* [`obj nodes`](#obj-nodes)
  * [`fn withCount(value)`](#fn-nodeswithcount)
  * [`fn withSeed(value)`](#fn-nodeswithseed)
  * [`fn withType(value)`](#fn-nodeswithtype)
* [`obj pulseWave`](#obj-pulsewave)
  * [`fn withOffCount(value)`](#fn-pulsewavewithoffcount)
  * [`fn withOffValue(value)`](#fn-pulsewavewithoffvalue)
  * [`fn withOnCount(value)`](#fn-pulsewavewithoncount)
  * [`fn withOnValue(value)`](#fn-pulsewavewithonvalue)
  * [`fn withTimeStep(value)`](#fn-pulsewavewithtimestep)
* [`obj resultAssertions`](#obj-resultassertions)
  * [`fn withMaxFrames(value)`](#fn-resultassertionswithmaxframes)
  * [`fn withType(value)`](#fn-resultassertionswithtype)
  * [`fn withTypeVersion(value)`](#fn-resultassertionswithtypeversion)
  * [`fn withTypeVersionMixin(value)`](#fn-resultassertionswithtypeversionmixin)
* [`obj sim`](#obj-sim)
  * [`fn withConfig(value)`](#fn-simwithconfig)
  * [`fn withConfigMixin(value)`](#fn-simwithconfigmixin)
  * [`fn withKey(value)`](#fn-simwithkey)
  * [`fn withKeyMixin(value)`](#fn-simwithkeymixin)
  * [`fn withLast(value=true)`](#fn-simwithlast)
  * [`fn withStream(value=true)`](#fn-simwithstream)
  * [`obj key`](#obj-simkey)
    * [`fn withTick(value)`](#fn-simkeywithtick)
    * [`fn withType(value)`](#fn-simkeywithtype)
    * [`fn withUid(value)`](#fn-simkeywithuid)
* [`obj stream`](#obj-stream)
  * [`fn withBands(value)`](#fn-streamwithbands)
  * [`fn withNoise(value)`](#fn-streamwithnoise)
  * [`fn withSpeed(value)`](#fn-streamwithspeed)
  * [`fn withSpread(value)`](#fn-streamwithspread)
  * [`fn withType(value)`](#fn-streamwithtype)
  * [`fn withUrl(value)`](#fn-streamwithurl)
* [`obj timeRange`](#obj-timerange)
  * [`fn withFrom(value="now-6h")`](#fn-timerangewithfrom)
  * [`fn withTo(value="now")`](#fn-timerangewithto)
* [`obj usa`](#obj-usa)
  * [`fn withFields(value)`](#fn-usawithfields)
  * [`fn withFieldsMixin(value)`](#fn-usawithfieldsmixin)
  * [`fn withMode(value)`](#fn-usawithmode)
  * [`fn withPeriod(value)`](#fn-usawithperiod)
  * [`fn withStates(value)`](#fn-usawithstates)
  * [`fn withStatesMixin(value)`](#fn-usawithstatesmixin)

## Fields

### fn withAlias

```jsonnet
withAlias(value)
```

PARAMETERS:

* **value** (`string`)


### fn withChannel

```jsonnet
withChannel(value)
```

PARAMETERS:

* **value** (`string`)

Used for live query
### fn withCsvContent

```jsonnet
withCsvContent(value)
```

PARAMETERS:

* **value** (`string`)


### fn withCsvFileName

```jsonnet
withCsvFileName(value)
```

PARAMETERS:

* **value** (`string`)


### fn withCsvWave

```jsonnet
withCsvWave(value)
```

PARAMETERS:

* **value** (`array`)


### fn withCsvWaveMixin

```jsonnet
withCsvWaveMixin(value)
```

PARAMETERS:

* **value** (`array`)


### fn withDatasource

```jsonnet
withDatasource(value)
```

PARAMETERS:

* **value** (`string`)

Set the datasource for this query.
### fn withDropPercent

```jsonnet
withDropPercent(value)
```

PARAMETERS:

* **value** (`number`)

Drop percentage (the chance we will lose a point 0-100)
### fn withErrorSource

```jsonnet
withErrorSource(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"plugin"`, `"downstream"`

Possible enum values:
 - `"plugin"` 
 - `"downstream"` 
### fn withErrorType

```jsonnet
withErrorType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"frontend_exception"`, `"frontend_observable"`, `"server_panic"`

Possible enum values:
 - `"frontend_exception"` 
 - `"frontend_observable"` 
 - `"server_panic"` 
### fn withFlamegraphDiff

```jsonnet
withFlamegraphDiff(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


### fn withHide

```jsonnet
withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

true if query is disabled (ie should not be returned to the dashboard)
NOTE: this does not always imply that the query should not be executed since
the results from a hidden query may be used as the input to other queries (SSE etc)
### fn withIntervalMs

```jsonnet
withIntervalMs(value)
```

PARAMETERS:

* **value** (`number`)

Interval is the suggested duration between time points in a time series query.
NOTE: the values for intervalMs is not saved in the query model.  It is typically calculated
from the interval required to fill a pixels in the visualization
### fn withLabels

```jsonnet
withLabels(value)
```

PARAMETERS:

* **value** (`string`)


### fn withLevelColumn

```jsonnet
withLevelColumn(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


### fn withLines

```jsonnet
withLines(value)
```

PARAMETERS:

* **value** (`integer`)


### fn withMax

```jsonnet
withMax(value)
```

PARAMETERS:

* **value** (`number`)


### fn withMaxDataPoints

```jsonnet
withMaxDataPoints(value)
```

PARAMETERS:

* **value** (`integer`)

MaxDataPoints is the maximum number of data points that should be returned from a time series query.
NOTE: the values for maxDataPoints is not saved in the query model.  It is typically calculated
from the number of pixels visible in a visualization
### fn withMin

```jsonnet
withMin(value)
```

PARAMETERS:

* **value** (`number`)


### fn withNodes

```jsonnet
withNodes(value)
```

PARAMETERS:

* **value** (`object`)


### fn withNodesMixin

```jsonnet
withNodesMixin(value)
```

PARAMETERS:

* **value** (`object`)


### fn withNoise

```jsonnet
withNoise(value)
```

PARAMETERS:

* **value** (`number`)


### fn withPoints

```jsonnet
withPoints(value)
```

PARAMETERS:

* **value** (`array`)


### fn withPointsMixin

```jsonnet
withPointsMixin(value)
```

PARAMETERS:

* **value** (`array`)


### fn withPulseWave

```jsonnet
withPulseWave(value)
```

PARAMETERS:

* **value** (`object`)


### fn withPulseWaveMixin

```jsonnet
withPulseWaveMixin(value)
```

PARAMETERS:

* **value** (`object`)


### fn withQueryType

```jsonnet
withQueryType(value)
```

PARAMETERS:

* **value** (`string`)

QueryType is an optional identifier for the type of query.
It can be used to distinguish different types of queries.
### fn withRawFrameContent

```jsonnet
withRawFrameContent(value)
```

PARAMETERS:

* **value** (`string`)


### fn withRefId

```jsonnet
withRefId(value)
```

PARAMETERS:

* **value** (`string`)

RefID is the unique identifier of the query, set by the frontend call.
### fn withResultAssertions

```jsonnet
withResultAssertions(value)
```

PARAMETERS:

* **value** (`object`)

Optionally define expected query result behavior
### fn withResultAssertionsMixin

```jsonnet
withResultAssertionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Optionally define expected query result behavior
### fn withScenarioId

```jsonnet
withScenarioId(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"annotations"`, `"arrow"`, `"csv_content"`, `"csv_file"`, `"csv_metric_values"`, `"datapoints_outside_range"`, `"error_with_source"`, `"exponential_heatmap_bucket_data"`, `"flame_graph"`, `"grafana_api"`, `"linear_heatmap_bucket_data"`, `"live"`, `"logs"`, `"manual_entry"`, `"no_data_points"`, `"node_graph"`, `"predictable_csv_wave"`, `"predictable_pulse"`, `"random_walk"`, `"random_walk_table"`, `"random_walk_with_error"`, `"raw_frame"`, `"server_error_500"`, `"simulation"`, `"slow_query"`, `"streaming_client"`, `"table_static"`, `"trace"`, `"usa"`, `"variables-query"`

Possible enum values:
 - `"annotations"` 
 - `"arrow"` 
 - `"csv_content"` 
 - `"csv_file"` 
 - `"csv_metric_values"` 
 - `"datapoints_outside_range"` 
 - `"error_with_source"` 
 - `"exponential_heatmap_bucket_data"` 
 - `"flame_graph"` 
 - `"grafana_api"` 
 - `"linear_heatmap_bucket_data"` 
 - `"live"` 
 - `"logs"` 
 - `"manual_entry"` 
 - `"no_data_points"` 
 - `"node_graph"` 
 - `"predictable_csv_wave"` 
 - `"predictable_pulse"` 
 - `"random_walk"` 
 - `"random_walk_table"` 
 - `"random_walk_with_error"` 
 - `"raw_frame"` 
 - `"server_error_500"` 
 - `"simulation"` 
 - `"slow_query"` 
 - `"streaming_client"` 
 - `"table_static"` 
 - `"trace"` 
 - `"usa"` 
 - `"variables-query"` 
### fn withSeriesCount

```jsonnet
withSeriesCount(value)
```

PARAMETERS:

* **value** (`integer`)


### fn withSim

```jsonnet
withSim(value)
```

PARAMETERS:

* **value** (`object`)


### fn withSimMixin

```jsonnet
withSimMixin(value)
```

PARAMETERS:

* **value** (`object`)


### fn withSpanCount

```jsonnet
withSpanCount(value)
```

PARAMETERS:

* **value** (`integer`)


### fn withSpread

```jsonnet
withSpread(value)
```

PARAMETERS:

* **value** (`number`)


### fn withStartValue

```jsonnet
withStartValue(value)
```

PARAMETERS:

* **value** (`number`)


### fn withStream

```jsonnet
withStream(value)
```

PARAMETERS:

* **value** (`object`)


### fn withStreamMixin

```jsonnet
withStreamMixin(value)
```

PARAMETERS:

* **value** (`object`)


### fn withStringInput

```jsonnet
withStringInput(value)
```

PARAMETERS:

* **value** (`string`)

common parameter used by many query types
### fn withTimeRange

```jsonnet
withTimeRange(value)
```

PARAMETERS:

* **value** (`object`)

TimeRange represents the query range
NOTE: unlike generic /ds/query, we can now send explicit time values in each query
NOTE: the values for timeRange are not saved in a dashboard, they are constructed on the fly
### fn withTimeRangeMixin

```jsonnet
withTimeRangeMixin(value)
```

PARAMETERS:

* **value** (`object`)

TimeRange represents the query range
NOTE: unlike generic /ds/query, we can now send explicit time values in each query
NOTE: the values for timeRange are not saved in a dashboard, they are constructed on the fly
### fn withUsa

```jsonnet
withUsa(value)
```

PARAMETERS:

* **value** (`object`)


### fn withUsaMixin

```jsonnet
withUsaMixin(value)
```

PARAMETERS:

* **value** (`object`)


### fn withWithNil

```jsonnet
withWithNil(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


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
### obj nodes


#### fn nodes.withCount

```jsonnet
nodes.withCount(value)
```

PARAMETERS:

* **value** (`integer`)


#### fn nodes.withSeed

```jsonnet
nodes.withSeed(value)
```

PARAMETERS:

* **value** (`integer`)


#### fn nodes.withType

```jsonnet
nodes.withType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"random"`, `"random edges"`, `"response_medium"`, `"response_small"`, `"feature_showcase"`

Possible enum values:
 - `"random"` 
 - `"random edges"` 
 - `"response_medium"` 
 - `"response_small"` 
 - `"feature_showcase"` 
### obj pulseWave


#### fn pulseWave.withOffCount

```jsonnet
pulseWave.withOffCount(value)
```

PARAMETERS:

* **value** (`integer`)


#### fn pulseWave.withOffValue

```jsonnet
pulseWave.withOffValue(value)
```

PARAMETERS:

* **value** (`number`)


#### fn pulseWave.withOnCount

```jsonnet
pulseWave.withOnCount(value)
```

PARAMETERS:

* **value** (`integer`)


#### fn pulseWave.withOnValue

```jsonnet
pulseWave.withOnValue(value)
```

PARAMETERS:

* **value** (`number`)


#### fn pulseWave.withTimeStep

```jsonnet
pulseWave.withTimeStep(value)
```

PARAMETERS:

* **value** (`integer`)


### obj resultAssertions


#### fn resultAssertions.withMaxFrames

```jsonnet
resultAssertions.withMaxFrames(value)
```

PARAMETERS:

* **value** (`integer`)

Maximum frame count
#### fn resultAssertions.withType

```jsonnet
resultAssertions.withType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `""`, `"timeseries-wide"`, `"timeseries-long"`, `"timeseries-many"`, `"timeseries-multi"`, `"directory-listing"`, `"table"`, `"numeric-wide"`, `"numeric-multi"`, `"numeric-long"`, `"log-lines"`

Type asserts that the frame matches a known type structure.
Possible enum values:
 - `""` 
 - `"timeseries-wide"` 
 - `"timeseries-long"` 
 - `"timeseries-many"` 
 - `"timeseries-multi"` 
 - `"directory-listing"` 
 - `"table"` 
 - `"numeric-wide"` 
 - `"numeric-multi"` 
 - `"numeric-long"` 
 - `"log-lines"` 
#### fn resultAssertions.withTypeVersion

```jsonnet
resultAssertions.withTypeVersion(value)
```

PARAMETERS:

* **value** (`array`)

TypeVersion is the version of the Type property. Versions greater than 0.0 correspond to the dataplane
contract documentation https://grafana.github.io/dataplane/contract/.
#### fn resultAssertions.withTypeVersionMixin

```jsonnet
resultAssertions.withTypeVersionMixin(value)
```

PARAMETERS:

* **value** (`array`)

TypeVersion is the version of the Type property. Versions greater than 0.0 correspond to the dataplane
contract documentation https://grafana.github.io/dataplane/contract/.
### obj sim


#### fn sim.withConfig

```jsonnet
sim.withConfig(value)
```

PARAMETERS:

* **value** (`object`)


#### fn sim.withConfigMixin

```jsonnet
sim.withConfigMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn sim.withKey

```jsonnet
sim.withKey(value)
```

PARAMETERS:

* **value** (`object`)


#### fn sim.withKeyMixin

```jsonnet
sim.withKeyMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn sim.withLast

```jsonnet
sim.withLast(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


#### fn sim.withStream

```jsonnet
sim.withStream(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


#### obj sim.key


##### fn sim.key.withTick

```jsonnet
sim.key.withTick(value)
```

PARAMETERS:

* **value** (`number`)


##### fn sim.key.withType

```jsonnet
sim.key.withType(value)
```

PARAMETERS:

* **value** (`string`)


##### fn sim.key.withUid

```jsonnet
sim.key.withUid(value)
```

PARAMETERS:

* **value** (`string`)


### obj stream


#### fn stream.withBands

```jsonnet
stream.withBands(value)
```

PARAMETERS:

* **value** (`integer`)


#### fn stream.withNoise

```jsonnet
stream.withNoise(value)
```

PARAMETERS:

* **value** (`number`)


#### fn stream.withSpeed

```jsonnet
stream.withSpeed(value)
```

PARAMETERS:

* **value** (`number`)


#### fn stream.withSpread

```jsonnet
stream.withSpread(value)
```

PARAMETERS:

* **value** (`number`)


#### fn stream.withType

```jsonnet
stream.withType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"fetch"`, `"logs"`, `"signal"`, `"traces"`

Possible enum values:
 - `"fetch"` 
 - `"logs"` 
 - `"signal"` 
 - `"traces"` 
#### fn stream.withUrl

```jsonnet
stream.withUrl(value)
```

PARAMETERS:

* **value** (`string`)


### obj timeRange


#### fn timeRange.withFrom

```jsonnet
timeRange.withFrom(value="now-6h")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"now-6h"`

From is the start time of the query.
#### fn timeRange.withTo

```jsonnet
timeRange.withTo(value="now")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"now"`

To is the end time of the query.
### obj usa


#### fn usa.withFields

```jsonnet
usa.withFields(value)
```

PARAMETERS:

* **value** (`array`)


#### fn usa.withFieldsMixin

```jsonnet
usa.withFieldsMixin(value)
```

PARAMETERS:

* **value** (`array`)


#### fn usa.withMode

```jsonnet
usa.withMode(value)
```

PARAMETERS:

* **value** (`string`)


#### fn usa.withPeriod

```jsonnet
usa.withPeriod(value)
```

PARAMETERS:

* **value** (`string`)


#### fn usa.withStates

```jsonnet
usa.withStates(value)
```

PARAMETERS:

* **value** (`array`)


#### fn usa.withStatesMixin

```jsonnet
usa.withStatesMixin(value)
```

PARAMETERS:

* **value** (`array`)

